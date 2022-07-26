package houndify

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const houndifyVoiceURL = "https://api.houndify.com:443/v1/audio"
const houndifyTextURL = "https://api.houndify.com:443/v1/text"

// Default user agent set by the SDK
const SDKUserAgent = "Go Houndify SDK"

type (
	// A Client holds the configuration and state, which is used for
	// sending all outgoing Houndify requests and appropriately saving their responses.
	Client struct {
		// The ClientID comes from the Houndify site.
		ClientID string
		// The ClientKey comes from the Houndify site.
		// Keep the key secret.
		ClientKey               string
		enableConversationState bool
		conversationState       interface{}
		// If Verbose is true, all data sent from the server is printed to stdout, unformatted and unparsed.
		// This includes partial transcripts, errors, HTTP headers details (status code, headers, etc.), and final response JSON.
		Verbose           bool
		HttpClient        *http.Client
		RequestInfoInBody bool
	}

	// all of the Hound server JSON messages have these basic fields
	houndServerMessage struct {
		Format  string `json:"Format"`
		Version string `json:"FormatVersion"`
	}
	houndServerPartialTranscript struct {
		houndServerMessage
		PartialTranscript string `json:"PartialTranscript"`
		DurationMS        int64  `json:"DurationMS"`
		Done              bool   `json:"Done"`
		SafeToStopAudio   *bool  `json:"SafeToStopAudio"`
	}
)

// EnableConversationState enables conversation state for future queries
func (c *Client) EnableConversationState() {
	c.enableConversationState = true
}

// DisableConversationState disables conversation state for future queries
func (c *Client) DisableConversationState() {
	c.enableConversationState = false
}

// ClearConversationState removes, or "forgets", the current conversation state
func (c *Client) ClearConversationState() {
	var emptyConvState interface{}
	c.conversationState = emptyConvState
}

// GetConversationState returns the current conversation state, useful for saving
func (c *Client) GetConversationState() interface{} {
	return c.conversationState
}

// SetConversationState sets the conversation state, useful for resuming from a saved point
func (c *Client) SetConversationState(newState interface{}) {
	c.conversationState = newState
}

// TextSearch sends a text request and returns the body of the Hound server response.
//
// An error is returned if there is a failure to create the request, failure to
// connect, failure to parse the response, or failure to update the conversation
// state (if applicable).
func (c *Client) TextSearch(textReq TextRequest) (string, error) {

	req, err := BuildRequest(&textReq, *c)

	// Add the TexRequest's context to the http request
	if textReq.ctx != nil {
		req = req.WithContext(textReq.ctx)
	}

	// Set the extra client headers
	for k, v := range textReq.headers {
		req.Header.Set(k, v)
	}

	if err != nil {
		return "", err
	}

	if c.HttpClient == nil {
		c.HttpClient = &http.Client{}
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", errors.New("failed to successfully run request: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("failed to read body: " + err.Error())
	}
	defer resp.Body.Close()

	bodyStr := string(body)

	if c.Verbose {
		fmt.Println(resp.Proto, resp.StatusCode)
		fmt.Println("Headers: ", resp.Header)
		fmt.Println(bodyStr)
	}

	//don't try to parse out conversation state from a bad response
	if resp.StatusCode >= 400 {
		return bodyStr, errors.New("error response")
	}
	// update with new conversation state
	if c.enableConversationState {
		newConvState, err := parseConversationState(bodyStr)
		if err != nil {
			return bodyStr, errors.Wrap(err, "unable to parse new conversation state from response")
		}
		c.conversationState = newConvState
	}

	return bodyStr, nil
}

// VoiceSearch sends an audio request and returns the body of the Hound server response.
//
// The partialTranscriptChan parameter allows the caller to receive for PartialTranscripts
// while the Hound server is listening to the voice search. If partial transcripts are not
// needed, create a throwaway channel that listens and discards all the PartialTranscripts
// sent.
//
// An error is returned if there is a failure to create the request, failure to
// connect, failure to parse the response, or failure to update the conversation
// state (if applicable).
func (c *Client) VoiceSearch(voiceReq VoiceRequest, partialTranscriptChan chan PartialTranscript) (string, error) {

	//so the partial transcript channel doesn't get closed before all transcripts are sent
	partialChanWait := sync.WaitGroup{}

	defer func() {
		go func() {
			//don't close the open partial transcript channel
			partialChanWait.Wait()
			close(partialTranscriptChan)
		}()
	}()

	// Ensure that RequestInfoInBody isn't set for VoiceRequests because the Audio stream
	// has to go into the body
	c.RequestInfoInBody = false
	req, err := BuildRequest(&voiceReq, *c)
	if voiceReq.ctx != nil {
		req = req.WithContext(voiceReq.ctx)
	}

	// Set the extra client headers
	for k, v := range voiceReq.headers {
		req.Header.Set(k, v)
	}

	if err != nil {
		return "", err
	}
	req.Body = ioutil.NopCloser(voiceReq.AudioStream)

	if c.HttpClient == nil {
		c.HttpClient = &http.Client{}
	}

	// send the request
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", errors.New("failed to successfully run request: " + err.Error())
	}

	if c.Verbose {
		fmt.Println(resp.Proto, resp.StatusCode)
		fmt.Println("Headers: ", resp.Header)
	}

	// partial transcript parsing

	reader := bufio.NewReader(resp.Body)
	var line string
	for {
		bytes, err := reader.ReadBytes('\n')
		line = strings.TrimSpace(string(bytes))
		if c.Verbose {
			fmt.Println(line)
		}
		if err != nil {
			if err != io.EOF {
				fmt.Println(err)
				return "", errors.New("error reading Houndify server response")
			}
			//EOF means this line must be the final response, done with partial transcripts
			break
		}
		if line == "" {
			continue
		}
		if _, convertErr := strconv.Atoi(line); convertErr == nil {
			// this is an integer, so one of the ObjectByteCountPrefixes, skip it
			continue
		}
		// attempt to parse incoming json into partial transcript
		incoming := houndServerPartialTranscript{}
		if err := json.Unmarshal([]byte(line), &incoming); err != nil {
			fmt.Println("fail reading hound server message")
			continue
		}
		if incoming.Format == "HoundVoiceQueryPartialTranscript" || incoming.Format == "SoundHoundVoiceSearchParialTranscript" {
			// convert from houndify server's struct to SDK's simplified struct
			partialDuration, err := time.ParseDuration(fmt.Sprintf("%d", incoming.DurationMS) + "ms")
			if err != nil {
				fmt.Println("failed reading the time in partial transcript")
				continue
			}
			partialChanWait.Add(1)
			go func() {
				partialTranscriptChan <- PartialTranscript{
					Message:         incoming.PartialTranscript,
					Duration:        partialDuration,
					Done:            incoming.Done,
					SafeToStopAudio: incoming.SafeToStopAudio,
				}
				partialChanWait.Done()
			}()
			continue
		}
		if incoming.Format == "SoundHoundVoiceSearchResult" {
			//this line is the final response, done with partial transcripts
			break
		}
	}

	bodyStr := line
	defer resp.Body.Close()

	//don't try to parse out conversation state from a bad response
	if resp.StatusCode >= 400 {
		return bodyStr, errors.New("error response")
	}
	// update with new conversation state
	if c.enableConversationState {
		newConvState, err := parseConversationState(bodyStr)
		if err != nil {
			return bodyStr, errors.Wrap(err, "unable to parse new conversation state from response")
		}
		c.conversationState = newConvState
	}

	return bodyStr, nil
}
