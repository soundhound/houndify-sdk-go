package houndify

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

// A TextRequest holds all the information needed to make a Houndify request.
// Create one of these per request to send and use a Client to send it.
type TextRequest struct {
	// The text query, e.g. "what time is it in london"
	Query             string
	UserID            string
	RequestID         string
	RequestInfoFields map[string]interface{}
	URL               string
}

// A VoiceRequest holds all the information needed to make a Houndify request.
// Create one of these per request to send and use a Client to send it.
type VoiceRequest struct {
	// Stream of audio in bytes. It must already be in correct encoding.
	// See the Houndify docs for details.
	AudioStream       io.Reader
	UserID            string
	RequestID         string
	RequestInfoFields map[string]interface{}
	URL               string
}

// Generic interface for the different types of requests
type requestable interface {

	// Create a new *http.Request with the given request type and URL. This creates a
	// request with an empty body that should be filled in later.
	NewRequest() (*http.Request, error)

	// Wrapper for generateAuthValues, as this function requires information specific to
	// the underlying struct and isn't accessible through the interface.
	AuthInfo(Client) (authInfo, error)

	// Wrapper for the createRequestInfo() function call, as like generateAuthValues() it
	// requires information from the underlying struct
	RequestInfo(Client) (requestInfo, error)

	// Return the underlying RequestInfo representation. Note that since it's held as a
	// map changing this will also change the underlying struct's values.
	GetRequestInfo() map[string]interface{}
}

// Take a generic requestable interface and create a http.Request from it using the built
// Client.
func BuildRequest(houndReq requestable, c Client) (*http.Request, error) {
	req, err := houndReq.NewRequest()
	if err != nil {
		return nil, err
	}

	// auth headers
	req.Header.Set("User-Agent", SDKUserAgent)
	auth, err := houndReq.AuthInfo(c)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Hound-Request-Authentication", auth.houndRequestAuth)
	req.Header.Set("Hound-Client-Authentication", auth.houndClientAuth)

	reqInfo := houndReq.GetRequestInfo()
	reqInfo["TimeStamp"] = auth.timeStamp

	// Set optional Language headers
	if val, ok := reqInfo["InputLanguageEnglishName"]; ok {
		req.Header.Set("InputLanguageEnglishName", val.(string))
	}
	if val, ok := reqInfo["InputLanguageIETFTag"]; ok {
		req.Header.Set("InputLanguageIETFTag", val.(string))
	}

	// Enable conversation state
	if c.enableConversationState {
		reqInfo["ConversationState"] = c.conversationState
	} else {
		var emptyConvState interface{}
		reqInfo["ConversationState"] = emptyConvState
	}

	requestInfo, err := houndReq.RequestInfo(c)
	if err != nil {
		return nil, err
	}

	requestInfoJSON, err := json.Marshal(requestInfo)
	if err != nil {
		return nil, errors.New("failed to create request info: " + err.Error())
	}

	if !c.RequestInfoInBody {
		req.Header.Set("Hound-Request-Info", string(requestInfoJSON))
	} else {

		// RequestInfo data in the body requires a header that specifies how long the
		// content will be.
		requestInfoJSON, err = json.Marshal(requestInfo)
		length := len(requestInfoJSON)
		strlen := strconv.Itoa(length)
		req.Header.Set("Hound-Request-Info-Length", strlen)

		if err != nil {
			return nil, errors.New("failed to create request info: " + err.Error())
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(requestInfoJSON))
	}
	return req, nil
}

func (r *TextRequest) NewRequest() (*http.Request, error) {
	// Use set URL, or fallback to default
	if len(r.URL) == 0 {
		r.URL = houndifyTextURL
	}

	// setup http request
	body := []byte(``)
	req, err := http.NewRequest("POST", r.URL+"?query="+url.PathEscape(r.Query), bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.New("failed to build http request: " + err.Error())
	}
	return req, nil
}

func (r *TextRequest) AuthInfo(c Client) (authInfo, error) {
	clientAuth, requestAuth, timestamp, err := generateAuthValues(c.ClientID, c.ClientKey, r.UserID, r.RequestID)
	return authInfo{
		houndClientAuth:  clientAuth,
		houndRequestAuth: requestAuth,
		timeStamp:        timestamp,
	}, err
}

func (r *TextRequest) RequestInfo(c Client) (requestInfo, error) {
	timestamp := r.RequestInfoFields["TimeStamp"].(int64)
	return createRequestInfo(c.ClientID, r.RequestID, timestamp, r.RequestInfoFields)
}

func (r *TextRequest) GetRequestInfo() map[string]interface{} {
	return r.RequestInfoFields
}

func (r *VoiceRequest) NewRequest() (*http.Request, error) {
	// Use set URL, or fallback to default
	if len(r.URL) == 0 {
		r.URL = houndifyVoiceURL
	}

	// setup http request
	req, err := http.NewRequest("POST", r.URL, nil)
	if err != nil {
		return nil, errors.New("failed to build http request: " + err.Error())
	}
	return req, nil
}

func (r *VoiceRequest) AuthInfo(c Client) (authInfo, error) {
	clientAuth, requestAuth, timestamp, err := generateAuthValues(c.ClientID, c.ClientKey, r.UserID, r.RequestID)
	return authInfo{
		houndClientAuth:  clientAuth,
		houndRequestAuth: requestAuth,
		timeStamp:        timestamp,
	}, err
}

func (r *VoiceRequest) RequestInfo(c Client) (requestInfo, error) {
	timestamp := r.RequestInfoFields["TimeStamp"].(int64)
	return createRequestInfo(c.ClientID, r.RequestID, timestamp, r.RequestInfoFields)
}

func (r *VoiceRequest) GetRequestInfo() map[string]interface{} {
	return r.RequestInfoFields
}
