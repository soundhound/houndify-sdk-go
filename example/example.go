package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/go-audio/wav"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

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
		fileContents, err := ioutil.ReadFile(audioFilePath)
		if err != nil {
			log.Fatalf("failed to read contents of file %q, err: %v", audioFilePath, err)
		}

		req := houndify.VoiceRequest{
			AudioStream:       bytes.NewReader(fileContents),
			UserID:            userID,
			RequestID:         createRequestID(),
			RequestInfoFields: make(map[string]interface{}),
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

// Stream an audio file to the server. This example demonstrates streaming a wav file,
// however this could easily be changed to stream audio from a microphone or something.
// Basically it just writes data from a buffer to the Request body every 1 second. The
// advantage of how golang has the http.Request's Body field is it's a Reader, so using
// io.Pipe() you can actually write any data into it. That means any stream of WAV data
// can just be piped in, and the requests will be made.
//
// This function also demonstrates how you can use the SafeToStopAudio flag to know when
// the server has all the data it needs.
func StreamAudio(client houndify.Client, fname, uid string) {
	f, err := os.Open(fname)
	defer f.Close()
	if err != nil {
		log.Fatalf("failed to read contents of file %q, err: %v\n", fname, err)
	}

	// Read WAV file data, determine bytes per second
	d := wav.NewDecoder(f)
	d.ReadInfo()

	// Use 1 second chunks
	bps := int(d.AvgBytesPerSec) * 1

	// Build pipe that lets us write into the io.Reader that is in the request
	rp, wp := io.Pipe()

	req := houndify.VoiceRequest{
		AudioStream: rp,
		UserID:      uid,
		RequestID:   createRequestID(),
	}

	// Start the function to write 1 second of data per 1 real second, by using a buffer
	// that is the size of 1 second of data. Note that using the .Read() function results
	// in the header portion of the file not being read. We have to use the ReadAt()
	// function to specify starting at the very first position of the actual file, or the
	// header isn't read.
	var loc int64 = 0
	buf := make([]byte, bps)
	done := make(chan bool)
	go func(wp *io.PipeWriter) {
		defer wp.Close()

		for {
			select {
			case <-done:
				//fmt.Println("Exiting write loop")
				return
			default:
				n, err := f.ReadAt(buf, loc)
				loc += int64(n)

				// At the EOF, the buffer will still have bytes read into it, have to write
				// those out before breaking the loop
				if err == io.EOF {
					wp.Write(buf[:n])
					return
				}

				// Write the amount of bytes that were read in
				wp.Write(buf[:n])
				time.Sleep(time.Duration(1) * time.Second)
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
