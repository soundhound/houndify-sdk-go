package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http/httptrace"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/go-audio/wav"
	houndify "github.com/soundhound/houndify-sdk-go"
)

const (
	// This is not the clientId. This is the app user, so many will likely exist per clientId.
	// This value can be any string.
	// See https://www.houndify.com/docs/ for more details.
	userID = "exampleUser"

	envClientIDKey  = "HOUNDIFY_CLIENT_ID"
	envClientKeyKey = "HOUNDIFY_CLIENT_KEY"

	cliClientIDKey  = "id"
	cliClientKeyKey = "key"
)

func main() {
	clientIDFlag := flag.String(cliClientIDKey, "", "Client ID")
	clientKeyFlag := flag.String(cliClientKeyKey, "", "Client Key")
	voiceFlag := flag.String("voice", "", "Audio file to use for voice query")
	textFlag := flag.String("text", "", "Message to use for text query")
	stdinFlag := flag.Bool("stdin", false, "Text query via stdin messages")
	streamFlag := flag.Bool("stream", false, "Stream audio file in real time to server, used with --voice")
	verboseFlag := flag.Bool("v", false, "Verbose mode, which prints raw server data")
	traceFlag := flag.Bool("trace", false, "Enable http client tracing")
	flag.Parse()

	// Make log not print out time info
	log.SetFlags(0)

	clientID := derefOrFetchFromEnv(clientIDFlag, envClientIDKey)
	clientKey := derefOrFetchFromEnv(clientKeyFlag, envClientKeyKey)

	var errsList []string
	if clientID == "" {
		msg := fmt.Sprintf("must set the client ID in environment variable: %q or via commmandline flag: -%s", envClientIDKey, cliClientIDKey)
		errsList = append(errsList, msg)
	}
	if clientKey == "" {
		msg := fmt.Sprintf("must set the client key in environment variable: %q or via commandline flag: -%s", envClientKeyKey, cliClientKeyKey)
		errsList = append(errsList, msg)
	}
	if len(errsList) > 0 {
		log.Fatalf("%s", strings.Join(errsList, "\n"))
	}

	// create a new client
	client := houndify.Client{
		ClientID:  clientID,
		ClientKey: clientKey,
		Verbose:   *verboseFlag,
	}
	client.EnableConversationState()

	switch {
	default:
		log.Fatalf("must choose either voice, text or stdin")

	case *voiceFlag != "" && !*streamFlag:
		// voice query
		audioFilePath := *voiceFlag
		fileContents, err := os.ReadFile(audioFilePath)
		if err != nil {
			log.Fatalf("failed to read contents of file %q, err: %v", audioFilePath, err)
		}

		req := houndify.VoiceRequest{
			AudioStream:       bytes.NewReader(fileContents),
			UserID:            userID,
			RequestID:         createRequestID(),
			RequestInfoFields: make(map[string]interface{}),
		}

		ctx := context.Background()
		req.WithContext(ctx)
		if *traceFlag {
			req.WithContext(httptrace.WithClientTrace(ctx, getDefaultClientTrace()))
		}

		// listen for partial transcript responses
		partialTranscripts := make(chan houndify.PartialTranscript)
		go func() {
			for partial := range partialTranscripts {
				if partial.Message != "" { // ignore the "" partial transcripts, not really useful
					fmt.Println(partial.Message)
				}
			}
		}()

		serverResponse, err := client.VoiceSearch(req, partialTranscripts)
		if err != nil {
			log.Fatalf("failed to make voice request: %v\n%s\n", err, serverResponse)
		}
		writtenResponse, err := houndify.ParseWrittenResponse(serverResponse)
		if err != nil {
			log.Fatalf("failed to decode hound response\n%s\n", serverResponse)
		}
		fmt.Println(writtenResponse)

	case *textFlag != "":
		// text query
		req := houndify.TextRequest{
			Query:             *textFlag,
			UserID:            userID,
			RequestID:         createRequestID(),
			RequestInfoFields: make(map[string]interface{}),
		}
		ctx := context.Background()
		req.WithContext(ctx)
		if *traceFlag {
			req.WithContext(httptrace.WithClientTrace(ctx, getDefaultClientTrace()))
		}
		serverResponse, err := client.TextSearch(req)
		if err != nil {
			log.Fatalf("failed to make text request: %v\n%s\n", err, serverResponse)
		}
		writtenResponse, err := houndify.ParseWrittenResponse(serverResponse)
		if err != nil {
			log.Fatalf("failed to decode hound response\n%s\n", serverResponse)
		}
		fmt.Println(writtenResponse)

	case *stdinFlag:
		// text queries in succession, demonstrating conversation state
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Enter a text query: ")
		for scanner.Scan() {
			req := houndify.TextRequest{
				Query:             scanner.Text(),
				UserID:            userID,
				RequestID:         createRequestID(),
				RequestInfoFields: make(map[string]interface{}),
			}
			serverResponse, err := client.TextSearch(req)
			if err != nil {
				fmt.Printf("failed to make text request: %v\n%s\nEnter another text query:", err, serverResponse)
				continue
			}
			writtenResponse, err := houndify.ParseWrittenResponse(serverResponse)
			if err != nil {
				log.Fatalf("failed to decode hound response\n%s\n", serverResponse)
			}
			fmt.Print(writtenResponse, "\n\n")
			fmt.Println("Enter another text query:")
		}

	case *voiceFlag != "" && *streamFlag:
		StreamAudio(client, *voiceFlag, userID)
	}
}

// Streams audio to the server using a WAV file as the source. While this example
// uses a file, the same pattern can be used for other sources like a microphone.
//
// Audio is sent in frame-aligned chunks at a realtime interval to better reflect
// live streaming behavior.
//
// The request body is backed by an io.Pipe, which allows arbitrary data to be
// written as a stream. This makes it easy to feed any WAV audio source directly
// into the request as it becomes available.
//
// The function also shows how to use the SafeToStopAudio signal to determine when
// the server has received enough audio and no more data needs to be sent.
func StreamAudio(client houndify.Client, fname, uid string) {
	f, err := os.Open(fname)
	if err != nil {
		log.Fatalf("failed to read contents of file %q, err: %v\n", fname, err)
	}
	defer f.Close()

	// Read WAV file data, determine bytes per second
	d := wav.NewDecoder(f)
	d.ReadInfo()

	targetStreamingIntervalMs := 20
	streamInfo, err := houndify.GetLPCMStreamInfo(int(d.NumChans),
		int(d.BitDepth), int(d.SampleRate), targetStreamingIntervalMs)
	if err != nil {
		log.Fatalf("failed to get LPCM chunk info: %v", err)
	}

	// Build pipe that lets us write into the io.Reader that is in the request
	rp, wp := io.Pipe()

	req := houndify.VoiceRequest{
		AudioStream: rp,
		UserID:      uid,
		RequestID:   createRequestID(),
	}

	// Start the function to stream audio in realtime
	// Note that using the .Read() function results
	// in the header portion of the file not being read. We have to use the ReadAt()
	// function to specify starting at the very first position of the actual file, or the
	// header isn't read.
	done := make(chan bool)
	go func(wp *io.PipeWriter) {
		defer wp.Close()

		var (
			loc    int64 = 0
			buf          = make([]byte, streamInfo.ChunkSize())
			ticker       = time.NewTicker(streamInfo.StreamingInterval())
		)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				fmt.Println("Context received done, exiting write loop")
				return

			case <-ticker.C:

				n, err := f.ReadAt(buf, loc)

				if n > 0 {
					loc += int64(n)
					// Write the amount of bytes that were read in
					wp.Write(buf[:n])
				}

				if err != nil {
					if err != io.EOF {
						// handle error
					} else {
						fmt.Println("Reached end of file")
					}
					return
				}
			}
		}
	}(wp)

	// listen for partial transcript responses
	partialTranscripts := make(chan houndify.PartialTranscript)
	go func() {
		for partial := range partialTranscripts {
			if partial.SafeToStopAudio != nil && *partial.SafeToStopAudio == true {
				fmt.Println("Safe to stop audio recieved")
				if done != nil {
					done <- true
				}
				return
			}
			if partial.Message != "" { // ignore the "" partial transcripts, not really useful
				fmt.Println(partial.Message)
			}
		}
	}()

	serverResponse, err := client.VoiceSearch(req, partialTranscripts)
	if err != nil {
		log.Fatalf("failed to make voice request: %v\n%s\n", err, serverResponse)
	}
	writtenResponse, err := houndify.ParseWrittenResponse(serverResponse)
	if err != nil {
		log.Fatalf("failed to decode hound response\n%s\n", serverResponse)
	}
	fmt.Println(writtenResponse)
}

// Creates a pseudo unique/random request ID.
//
// SDK users should do something similar so each request to the Hound server
// is signed differently to prevent replay attacks.
func createRequestID() string {
	n := 10
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%X", b)
}

// derefOrFetchFromEnv tries to dereference and retrieve a non-empty
// string stored in the string pointer, otherwise it falls back
// to retrieving the value stored in the environment keyed by envKey.
func derefOrFetchFromEnv(strPtr *string, envKey string) string {
	if strPtr != nil && *strPtr != "" {
		return *strPtr
	}
	return os.Getenv(envKey)
}

func getDefaultClientTrace() *httptrace.ClientTrace {
	traceLogger := log.New(os.Stdout, "[httptrace] ", log.Ltime|log.Lmicroseconds)
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			traceLogger.Println("GotConn: ", info)
		},
		PutIdleConn: func(err error) {
			traceLogger.Println("PutIdleConn: ", err)
		},
		GotFirstResponseByte: func() {
			traceLogger.Println("GotFirstResponseByte")
		},
		Got100Continue: func() {
			traceLogger.Println("Got100Continue")
		},
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			traceLogger.Println("Got1xxResponse: ", code, header)
			return nil
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			traceLogger.Println("DNSStart: ", info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			traceLogger.Println("DNSDone: ", info)
		},
		ConnectStart: func(network, addr string) {
			traceLogger.Println("ConnectStart: ", addr)
		},
		ConnectDone: func(network, addr string, err error) {
			traceLogger.Println("ConnectDone: ", network, addr, err)
		},
		TLSHandshakeStart: func() {
			traceLogger.Println("TLSHandshakeStart")
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			traceLogger.Println("TLSHandshakeDone: ", state, err)
		},
		WroteHeaderField: func(key string, value []string) {
			traceLogger.Println("WroteHeaderField: ", key, value)
		},
		WroteHeaders: func() {
			traceLogger.Println("WroteHeaders")
		},
		Wait100Continue: func() {
			traceLogger.Println("Wait100Continue")
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			traceLogger.Println("WroteRequest: ", info)
		},
	}
	return trace
}
