package houndify

import (
	"bytes"
	"context"
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

	// Extra header that should be added to http request
	headers map[string]string

	// Context variable, should only be set through the WithContext() function
	ctx context.Context
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

	// Extra header that should be added to http request
	headers map[string]string

	// Context variable, should only be set through the WithContext() function
	ctx context.Context
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
	RequestInfo(Client, requestInfo) (requestInfo, error)

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

	//
	reqInfo := houndReq.GetRequestInfo()
	if reqInfo == nil {
		reqInfo = make(map[string]interface{})
	}

	reqInfo["TimeStamp"] = auth.timeStamp

	// Set the language headers based on provided fields in reqInfo
	// The header names have a slightly different format, so transform them if they exist
	// in the reqInfo.
	langHeaders := map[string]string{
		"InputLanguageEnglishName": "Hound-Input-Language-English-Name",
		"InputLanguageIETFTag":     "Hound-Input-Language-IETF-Tag",
	}

	for input, output := range langHeaders {
		if val, ok := reqInfo[input]; ok {
			req.Header.Set(output, val.(string))
		}
	}

	// Enable conversation state
	if c.enableConversationState {
		reqInfo["ConversationState"] = c.conversationState
	} else {
		var emptyConvState interface{}
		reqInfo["ConversationState"] = emptyConvState
	}

	requestInfo, err := houndReq.RequestInfo(c, reqInfo)
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

func (r *TextRequest) RequestInfo(c Client, reqInfo requestInfo) (requestInfo, error) {
	if r.RequestInfoFields == nil {
		r.RequestInfoFields = reqInfo
	}
	timestamp := r.RequestInfoFields["TimeStamp"].(int64)
	return createRequestInfo(c.ClientID, r.RequestID, timestamp, r.RequestInfoFields)
}

func (r *TextRequest) GetRequestInfo() map[string]interface{} {
	return r.RequestInfoFields
}

func (r *TextRequest) WithContext(ctx context.Context) {
	r.ctx = ctx
}

func (r *TextRequest) Headers(headers map[string]string) {
	r.headers = headers
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

func (r *VoiceRequest) RequestInfo(c Client, reqInfo requestInfo) (requestInfo, error) {
	if r.RequestInfoFields == nil {
		r.RequestInfoFields = reqInfo
	}
	timestamp := r.RequestInfoFields["TimeStamp"].(int64)
	return createRequestInfo(c.ClientID, r.RequestID, timestamp, r.RequestInfoFields)
}

func (r *VoiceRequest) GetRequestInfo() map[string]interface{} {
	return r.RequestInfoFields
}

func (r *VoiceRequest) WithContext(ctx context.Context) {
	r.ctx = ctx
}

func (r *VoiceRequest) Headers(headers map[string]string) {
	r.headers = headers
}
