package houndify

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type HoundifyDisambiguationChoice struct {
	Transcription      string
	ConfidenceScore    float64
	FixedTranscription string
}

type HoundifyDisambiguation struct {
	NumToShow  int64
	ChoiceData []HoundifyDisambiguationChoice
}

type HoundifyBuildInfo struct {
	User        string
	Date        string
	Machine     string
	SVNRevision string
	SVNBranch   string
	BuildNumber string
	King        string
	Variant     string
}

type HoundifyDomain struct {
	Domain         string
	DomainUniqueId string
	CreditsUsed    float64
}

type HoundifyResponseResult struct {
	CommandKind               string
	SpokenResponse            string
	SpokenResponseLong        string
	WrittenResponse           string
	WrittenResponseLong       string
	SpokenResponseSSML        *string   `json:"SpokenResponseSSML,omitempty"`
	SpokenResponseSSMLLong    *string   `json:"SpokenResponseSSMLLong,omitempty"`
	SmallScreenHTML           *string   `json:"SmallScreenHTML,omitempty"`
	LargeScreenHTML           *string   `json:"LargeScreenHTML,omitempty"`
	UnderstandingConfidence   *float64  `json:"UnderstandingConfidence,omitempty"`
	OutputOverrideDiagnostics *[]string `json:"OutputOverrideDiagnostics,omitempty"`
	AutoListen                bool
	ConversationState         interface{}
}

type HoundifyResponse struct {
	Format            string
	FormatVersion     string
	Status            string
	NumToReturn       int64
	AllResults        []HoundifyResponseResult
	ErrorMessage      *string                 `json:"ErrorMessage,omitempty"`
	ResultsAreFinal   *[]bool                 `json:"ResultsAreFinal,omitempty"`
	Disambiguation    *HoundifyDisambiguation `json:"Disambiguation,omitempty"`
	QueryID           *string                 `json:"QueryID,omitempty"`
	ServerGeneratedId *string                 `json:"ServerGeneratedId,omitempty"`
	AudioLength       *float64                `json:"AudioLength,omitempty"`
	RealSpeechTime    *float64                `json:"RealSpeechTime,omitempty"`
	RealTime          *float64                `json:"RealTime,omitempty"`
	BuildInfo         HoundifyBuildInfo
	DomainUsage       []HoundifyDomain
}

// ParseWrittenResponse will take final server response JSON (as a string)
// and parse out the human readable text to be displayed to the end user.
// If the string is invalid JSON, the server had an error, or there was nothing
// to reply with, an error is returned.
func ParseWrittenResponse(serverResponseJSON string) (string, error) {
	result := HoundifyResponse{}
	err := json.Unmarshal([]byte(serverResponseJSON), &result)
	if err != nil {
		fmt.Println(err.Error())
		return "", errors.New("failed to decode json")
	}
	if !strings.EqualFold(result.Status, "OK") {
		return "", errors.New(*result.ErrorMessage)
	}
	if result.NumToReturn < 1 || len(result.AllResults) < 1 {
		return "", errors.New("no results to return")
	}
	return result.AllResults[0].WrittenResponseLong, nil
}

// ParseSpokenResponse will take final server response JSON (as a string)
// and parse out the human readable text to be spoken to the end user.
// If the string is invalid JSON, the server had an error, or there was nothing
// to reply with, an error is returned.
func ParseSpokenResponse(serverResponseJSON string) (string, error) {
	result := HoundifyResponse{}
	err := json.Unmarshal([]byte(serverResponseJSON), &result)
	if err != nil {
		fmt.Println(err.Error())
		return "", errors.New("failed to decode json")
	}
	if !strings.EqualFold(result.Status, "OK") {
		return "", errors.New(*result.ErrorMessage)
	}
	if result.NumToReturn < 1 || len(result.AllResults) < 1 {
		return "", errors.New("no results to return")
	}
	return result.AllResults[0].SpokenResponseLong, nil
}

func ParseFirstHypothesis(serverResponseJSON string) (string, error) {
	result := HoundifyResponse{}
	err := json.Unmarshal([]byte(serverResponseJSON), &result)
	if err != nil {
		fmt.Println(err.Error())
		return "", errors.New("failed to decode json")
	}
	if !strings.EqualFold(result.Status, "OK") {
		return "", errors.New(*result.ErrorMessage)
	}
	if result.Disambiguation == nil {
		return "", errors.New("no Disabiguation listed")
	}
	if result.Disambiguation.NumToShow < 1 || len(result.Disambiguation.ChoiceData) < 1 {
		return "", errors.New("no Choices listed")
	}
	return result.Disambiguation.ChoiceData[0].Transcription, nil
}

func parseConversationState(serverResponseJSON string) (interface{}, error) {
	result := HoundifyResponse{}
	err := json.Unmarshal([]byte(serverResponseJSON), &result)
	if err != nil {
		fmt.Println(err.Error())
		return "", errors.New("failed to decode json")
	}
	if !strings.EqualFold(result.Status, "OK") {
		return "", errors.New(*result.ErrorMessage)
	}
	if result.NumToReturn < 1 || len(result.AllResults) < 1 {
		return "", errors.New("no results to return")
	}
	return result.AllResults[0].ConversationState, nil
}
