package houndify

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const houndifyVoiceURL = "https://api.houndify.com:443/v1/audio"
const houndifyTextURL = "https://api.houndify.com:443/v1/text"

type (
	Client struct {
		ClientID                string
		ClientKey               string
		enableConversationState bool
		conversationState       interface{}
	}
	//create one of these per request you want to send
	TextRequest struct {
		Query             string //e.g. "what time is it in london"
		UserID            string
		RequestID         string
		RequestInfoFields map[string]interface{}
	}
	VoiceRequest struct {
		AudioStream       io.Reader //stream of audio in bytes - must already be in correct encoding
		UserID            string
		RequestID         string
		RequestInfoFields map[string]interface{}
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
	}
)

// EnableConversationState enables conversation state for future queries
func (h *Client) EnableConversationState() {
	h.enableConversationState = true
}

// DisableConversationState disables conversation state for future queries
func (h *Client) DisableConversationState() {
	h.enableConversationState = false
}

// ClearConversationState removes, or "forgets", the current conversation state
func (h *Client) ClearConversationState() {
	var emptyConvState interface{}
	h.conversationState = emptyConvState
}

// GetConversationState returns the current conversation state, useful for saving
func (h *Client) GetConversationState() interface{} {
	return h.conversationState
}

// SetConversationState sets the conversation state, useful for resuming from a saved point
func (h *Client) SetConversationState(newState interface{}) {
	h.conversationState = newState
}

// TextSearch sends a text request and returns the body of the Hound server response.
//
// An error is returned if there is a failure to create the request, failure to
// connect, failure to parse the response, or failure to update the conversation
// state (if applicable).
func (h *Client) TextSearch(textReq TextRequest) (string, error) {
	//setup http request
	body := []byte(``)
	req, err := http.NewRequest("POST", houndifyTextURL+"?query="+url.PathEscape(textReq.Query), bytes.NewBuffer(body))
	if err != nil {
		return "", errors.New("failed to build http request: " + err.Error())
	}
	//auth headers
	req.Header.Set("User-Agent", "Go Houndify SDK")
	clientAuth, requestAuth, timestamp, err := generateAuthValues(h.ClientID, h.ClientKey, textReq.UserID, textReq.RequestID)
	if err != nil {
		return "", errors.New("failed to create auth headers: " + err.Error())
	}
	req.Header.Set("Hound-Request-Authentication", requestAuth)
	req.Header.Set("Hound-Client-Authentication", clientAuth)

	//optional language headers
	if val, ok := textReq.RequestInfoFields["InputLanguageEnglishName"]; ok {
		req.Header.Set("InputLanguageEnglishName", val.(string))
	}
	if val, ok := textReq.RequestInfoFields["InputLanguageIETFTag"]; ok {
		req.Header.Set("InputLanguageIETFTag", val.(string))
	}

	//conversation state
	if h.enableConversationState {
		textReq.RequestInfoFields["ConversationState"] = h.conversationState
	} else {
		var emptyConvState interface{}
		textReq.RequestInfoFields["ConversationState"] = emptyConvState
	}

	//request info json
	requestInfo, err := createRequestInfo(h.ClientID, textReq.RequestID, timestamp, textReq.RequestInfoFields)
	if err != nil {
		return "", errors.New("failed to create request info: " + err.Error())
	}
	requestInfoJSON, err := json.Marshal(requestInfo)
	if err != nil {
		return "", errors.New("failed to create request info: " + err.Error())
	}
	req.Header.Set("Hound-Request-Info", string(requestInfoJSON))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("failed to successfully run request: " + err.Error())
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("failed to ready body: " + err.Error())
	}
	defer resp.Body.Close()

	//update with new conversation state
	if h.enableConversationState {
		newConvState, err := parseConversationState(string(body))
		if err != nil {
			return string(body), errors.New("unable to parse new conversation state from response")
		}
		h.conversationState = newConvState
	}

	return string(body), nil
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
func (h *Client) VoiceSearch(voiceReq VoiceRequest, partialTranscriptChan chan PartialTranscript) (string, error) {
	//setup http request
	req, err := http.NewRequest("POST", houndifyVoiceURL, nil)
	if err != nil {
		return "", errors.New("failed to build http request: " + err.Error())
	}
	//auth headers
	req.Header.Set("User-Agent", "Go Houndify SDK")
	clientAuth, requestAuth, timestamp, err := generateAuthValues(h.ClientID, h.ClientKey, voiceReq.UserID, voiceReq.RequestID)
	if err != nil {
		return "", errors.New("failed to create auth headers: " + err.Error())
	}
	req.Header.Set("Hound-Request-Authentication", requestAuth)
	req.Header.Set("Hound-Client-Authentication", clientAuth)

	//optional language headers
	if val, ok := voiceReq.RequestInfoFields["InputLanguageEnglishName"]; ok {
		req.Header.Set("InputLanguageEnglishName", val.(string))
	}
	if val, ok := voiceReq.RequestInfoFields["InputLanguageIETFTag"]; ok {
		req.Header.Set("InputLanguageIETFTag", val.(string))
	}

	//conversation state
	if h.enableConversationState {
		voiceReq.RequestInfoFields["ConversationState"] = h.conversationState
	} else {
		var emptyConvState interface{}
		voiceReq.RequestInfoFields["ConversationState"] = emptyConvState
	}

	//request info json
	requestInfo, err := createRequestInfo(h.ClientID, voiceReq.RequestID, timestamp, voiceReq.RequestInfoFields)
	if err != nil {
		return "", errors.New("failed to create request info: " + err.Error())
	}
	requestInfoJSON, err := json.Marshal(requestInfo)
	if err != nil {
		return "", errors.New("failed to create request info: " + err.Error())
	}
	req.Header.Set("Hound-Request-Info", string(requestInfoJSON))

	req.Body = ioutil.NopCloser(voiceReq.AudioStream)
	client := &http.Client{}

	//send the request

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("failed to successfully run request: " + err.Error())
	}

	//partial transcript parsing

	scanner := bufio.NewScanner(resp.Body)
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		if line == "" {
			continue
		}
		if _, convertErr := strconv.Atoi(line); convertErr == nil {
			//this is an integer, so one of the ObjectByteCountPrefixes, skip it
			continue
		}
		//attempt to parse incoming json into partial transcript
		incoming := houndServerPartialTranscript{}
		if err := json.Unmarshal([]byte(line), &incoming); err != nil {
			fmt.Println("fail reading hound server message")
			continue
		}
		if incoming.Format == "HoundVoiceQueryPartialTranscript" || incoming.Format == "SoundHoundVoiceSearchParialTranscript" {
			//convert from houndify server's struct to SDK's simplified struct
			partialDuration, err := time.ParseDuration(fmt.Sprintf("%d", incoming.DurationMS) + "ms")
			if err != nil {
				fmt.Println("failed reading the time in partial transcript")
				continue
			}
			go func() {
				partialTranscriptChan <- PartialTranscript{
					Message:  incoming.PartialTranscript,
					Duration: partialDuration,
					Done:     incoming.Done,
				}
			}()
		} else if incoming.Format == "SoundHoundVoiceSearchResult" {
			//it wasn't actually a partial transcript, it was a final message with everything
			//we're done with partial transcripts now
			break
		}
	}
	close(partialTranscriptChan)

	body := line
	defer resp.Body.Close()

	//update with new conversation state
	if h.enableConversationState {
		newConvState, err := parseConversationState(string(body))
		if err != nil {
			return string(body), errors.New("unable to parse new conversation state from response")
		}
		h.conversationState = newConvState
	}

	return body, nil
}
