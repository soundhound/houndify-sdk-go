package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	houndify "github.com/soundhound/houndify-sdk-go/houndify"
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

	case *voiceFlag != "":
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
			fmt.Println("Enter another text query: ")
		}
	}
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
