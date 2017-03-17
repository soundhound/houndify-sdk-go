package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	houndify "github.com/soundhound/houndify-sdk-go/houndify"
)

// This is not the clientId. This is the app user, so many will likely exist per clientId.
// This value can be any string.
// See https://www.houndify.com/docs/ for more details.
const userID = "exampleUser"

func main() {
	clientIDFlag := flag.String("id", "", "Client ID")
	clientKeyFlag := flag.String("key", "", "Client Key")
	voiceFlag := flag.String("voice", "", "Audio file to use for voice query")
	textFlag := flag.String("text", "", "Message to use for text query")
	stdinFlag := flag.Bool("stdin", false, "Text query via stdin messages")
	flag.Parse()

	clientID := *clientIDFlag
	clientKey := *clientKeyFlag

	if clientID == "" {
		//check environment variable
		if os.Getenv("HOUNDIFY_CLIENT_ID") != "" {
			clientID = os.Getenv("HOUNDIFY_CLIENT_ID")
		}
	}
	if clientKey == "" {
		//check environment variable
		if os.Getenv("HOUNDIFY_CLIENT_KEY") != "" {
			clientKey = os.Getenv("HOUNDIFY_CLIENT_KEY")
		}
	}

	if clientID == "" {
		fmt.Println("Need to set the client ID")
		os.Exit(1)
	}
	if clientKey == "" {
		fmt.Println("Need to set the client key")
		os.Exit(1)
	}

	//create a new client
	client := houndify.Client{
		ClientID:  clientID,
		ClientKey: clientKey,
	}
	client.EnableConversationState()

	if *voiceFlag != "" {
		// voice query
		audioFilePath := *voiceFlag
		fileContents, err := ioutil.ReadFile(audioFilePath)
		if err != nil {
			fmt.Println("failed to read contents of file " + audioFilePath + ": " + err.Error())
			os.Exit(1)
		}

		req := houndify.VoiceRequest{
			AudioStream:       bytes.NewReader(fileContents),
			UserID:            userID,
			RequestID:         createRequestID(),
			RequestInfoFields: make(map[string]interface{}),
		}

		//listen for partial transcript responses
		partialTranscripts := make(chan houndify.PartialTranscript)
		go func() {
			for partial := range partialTranscripts {
				if partial.Message != "" { //ignore the "" partial transcripts, not really useful
					fmt.Println(partial.Message)
				}
			}
		}()
		serverResponse, err := client.VoiceSearch(req, partialTranscripts)
		if err != nil {
			fmt.Println("failed to make voice request: " + err.Error())
			fmt.Println(serverResponse)
			os.Exit(1)
		}
		writtenResponse, err := houndify.ParseWrittenResponse(serverResponse)
		if err != nil {
			fmt.Println("failed to decode hound response")
			fmt.Println(serverResponse)
			os.Exit(1)
		}
		fmt.Println(writtenResponse)
	} else if *textFlag != "" {
		// text query
		req := houndify.TextRequest{
			Query:             *textFlag,
			UserID:            userID,
			RequestID:         createRequestID(),
			RequestInfoFields: make(map[string]interface{}),
		}
		serverResponse, err := client.TextSearch(req)
		if err != nil {
			fmt.Println("failed to make text request: " + err.Error())
			fmt.Println(serverResponse)
			os.Exit(1)
		}
		writtenResponse, err := houndify.ParseWrittenResponse(serverResponse)
		if err != nil {
			fmt.Println("failed to decode hound response")
			fmt.Println(serverResponse)
			os.Exit(1)
		}
		fmt.Println(writtenResponse)
	} else if *stdinFlag {
		// text queries in succession, demonstrating conversation state
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Enter a text query:")
		for scanner.Scan() {
			req := houndify.TextRequest{
				Query:             scanner.Text(),
				UserID:            userID,
				RequestID:         createRequestID(),
				RequestInfoFields: make(map[string]interface{}),
			}
			serverResponse, err := client.TextSearch(req)
			if err != nil {
				fmt.Println("failed to make text request: " + err.Error())
				fmt.Println(serverResponse)
				os.Exit(1)
			}
			writtenResponse, err := houndify.ParseWrittenResponse(serverResponse)
			if err != nil {
				fmt.Println("failed to decode hound response")
				fmt.Println(serverResponse)
				os.Exit(1)
			}
			fmt.Print(writtenResponse, "\n\n")
			fmt.Println("Enter another text query:")
		}
	} else {
		fmt.Println("must choose voice, text or stdin")
		os.Exit(1)
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
