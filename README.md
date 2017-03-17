# Houndify SDK for Go [![Build Status](https://travis-ci.org/soundhound/houndify-sdk-go.svg?branch=master)](https://travis-ci.org/soundhound/houndify-sdk-go)

houndify-sdk-go is the official Houndify SDK for the Go programming language.

The SDK allows you to make voice and text queries to the Houndify API. The SDK comes with a fully functional example app that demonstrates usage and the various SDK features. The SDK has no third party dependencies.

## Requirements

- Go v1.8+
- Houndify account available from [Houndify.com](https://www.houndify.com)

## Installing

To use the SDK and/or example app, you will need a client ID and client key. You can get those after creating a [Houndify](https://www.houndify.com) account and registering a client.

Once you have set your `$GOPATH`, you can add both the Go Houndify SDK and example app.

```
go get github.com/soundhound/houndify-sdk-go
```

The example app will be compiled and available at `$GOPATH/bin/houndify-sdk-go` and the SDK will be ready to import and use.

## Example App

`example.go` provides a working example of how to use the SDK.

The example app features three modes of interacting with the Houndify API:

1. Voice
2. Text
3. Stdin Interactive Text

To run the voice search:

```
houndify-sdk-go --id YOUR_CLIENT_ID --key YOUR_CLIENT_KEY --voice ./test_audio/whattimeisitindenver.wav
```

To run a text search:

```
houndify-sdk-go --id YOUR_CLIENT_ID --key YOUR_CLIENT_KEY --text "tell me a joke"
```

To run an interactive text search:

```
houndify-sdk-go --id YOUR_CLIENT_ID --key YOUR_CLIENT_KEY --stdin
```

You will then be prompted to type a query.

After Houndify replies with a response, you can follow up with additional text queries. Houndify will keep track of the conversation history, and interpret new queries in the context of previous ones.

An example set of queries:
 - "what is two plus six"
 - "minus 4"
 - "what is the square root of that"

Instead of using the `--id` and `--key` flags, you may set the environment variables `HOUNDIFY_CLIENT_ID` and `HOUNDIFY_CLIENT_KEY`.

## Using the SDK

To use the SDK, you must import the package

```go
import (
    houndify "github.com/soundhound/houndify-sdk-go/houndify"
)
```

Create a new client

```go
client := houndify.Client{
    ClientID:  "YOUR_CLIENT_ID",
    ClientKey: "YOUR_CLIENT_KEY",
}
```

For a voice search, create a VoiceRequest and channel for partial transcripts

```go
req := houndify.VoiceRequest{
    AudioStream:       bytes.NewReader(audioDataByteArray),
    UserID:            "appUser123",
    RequestID:         "uniqueRequest456",
    RequestInfoFields: make(map[string]interface{}),
}

//listen for partial transcripts while audio is streaming
partialTranscripts := make(chan houndify.PartialTranscript)
go func() {
    for partial := range partialTranscripts {
        fmt.Println(partial.Message)
    }
}()

serverResponse, err := client.VoiceSearch(req, partialTranscripts)
```

For a text search, create a TextRequest

```go
req := houndify.TextRequest{
    Query:             "what time is it in paris",
    UserID:            "appUser123",
    RequestID:         "uniqueRequest456",
    RequestInfoFields: make(map[string]interface{}),
}

serverResponse, err := client.TextSearch(req)
```

### Conversation State

Houndified domains can use context to enable a conversational user interaction. For example, users can say "show me coffee shops near me", "which ones have wifi?", "sort by rating", "navigate to the first one". You can enable, disable, clear, set and get the client's conversation state with the following houndify.Client methods.

```go
client.EnableConversationState()

client.DisableConversationState()

client.ClearConversationState()

currentState := client.GetConversationState()

client.SetConversationState(newState)
```

## Contributing

There are multiple ways to contribute to the SDK.

If you found a bug or have a feature request, please open an [Issue](https://github.com/soundhound/houndify-sdk-go/issues).

If you would like to make a code contribution, please sign the [CLA](https://cla-assistant.io/soundhound/houndify-sdk-go), and make a [Pull Request](https://github.com/soundhound/houndify-sdk-go/pulls) with your changes.

For account issues, security issues, or if you are unable to post publicly, please [contact us](https://www.houndify.com/contact) directly.

## License

The Houndify SDK for Go is distributed under the MIT License. See the LICENSE file for more information.
