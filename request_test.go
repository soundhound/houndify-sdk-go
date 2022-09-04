package houndify_test

import (
	"bytes"
	. "github.com/soundhound/houndify-sdk-go"
	"gotest.tools/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

type RoundTripFunc func(req *http.Request) *http.Response

// Satisfy the RoundTripper Interface that can act as a mock for making requests and
// returning responses. Any function with the matching prototype as RoundTripFunc can be
// used.
//
// This means to test a request, simply write a function that verifies the request, then
// returns a mock response. This response is then tested outside of the RoundTripFunc.
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// Return a client with the mock RoundTripper
func NewTestClient(f RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(f),
	}
}

// Return a Client with the mock http Client
func NewTestHoundifyClient(c *http.Client) Client {
	return Client{
		ClientID:   "9M22RyQGeu4bk1ToWkjX4g==",
		ClientKey:  "vHSRCJhQa6cIzZ6hCrQHwcKDQbdyBuV6mqFXuBG9vAQe3MqjVIEheNDoaTP6n-DQSzhoBsOJwOP5IrWM2pF1fg==",
		HttpClient: c,
	}
}

// Return a basic Text Request
func NewTestTextRequest() TextRequest {
	return TextRequest{
		URL:               "http://test.com/v1/text",
		Query:             "what is the time",
		UserID:            "TestUserID",
		RequestID:         "TestRequestID",
		RequestInfoFields: make(map[string]interface{}),
	}
}

// Return a basic Text Request
func NewTestVoiceRequest() VoiceRequest {
	return VoiceRequest{
		URL:               "http://test.com/v1/voice",
		UserID:            "TestUserID",
		RequestID:         "TestRequestID",
		RequestInfoFields: make(map[string]interface{}),
	}
}

// Tests TextRequest.NewRequest()
func TestNewTextRequest(t *testing.T) {

	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.Method, "POST")
		assert.Equal(t, req.URL.String(), "http://test.com/v1/text?query=what%20is%20the%20time")
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`No clue`)),
			Header:     make(http.Header),
		}
	})

	textReq := NewTestTextRequest()
	req, err := textReq.NewRequest()
	assert.NilError(t, err)
	mockClient.Do(req)
}

// Tests VoiceRequest.NewRequest()
func TestNewVoiceRequest(t *testing.T) {

	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.Method, "POST")
		assert.Equal(t, req.URL.String(), "http://test.com/v1/voice")
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`No clue`)),
			Header:     make(http.Header),
		}
	})

	voiceReq := NewTestVoiceRequest()
	req, err := voiceReq.NewRequest()
	assert.NilError(t, err)
	mockClient.Do(req)
}

// Tests BuildRequest(TextRequest, Client), ensure the following:
// - URL is set to the proper URL configured in the textReq
// - User Agent is set properly
// - Headers all exist that are set
// - TODO:
//  	- RequestInfo verification
//  	- Find way to mock Auth stuff so dynamic auth headers (they change with time etc)
func TestBuildTextRequest(t *testing.T) {

	var expectedVals = map[string]string{
		"User-Agent": "Go Houndify SDK",
	}

	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.Method, "POST")
		assert.Equal(t, req.URL.String(), "http://test.com/v1/text?query=what%20is%20the%20time")

		for k, v := range expectedVals {
			assert.Equal(t, req.Header.Get(k), v)
		}

		return &http.Response{}
	})

	// Make the mock call after getting test structs
	textReq := NewTestTextRequest()
	houndifyClient := NewTestHoundifyClient(mockClient)
	req, err := BuildRequest(&textReq, houndifyClient)
	assert.NilError(t, err)
	mockClient.Do(req)
}
