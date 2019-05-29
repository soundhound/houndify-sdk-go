package houndify

import (
	"time"
)

type PartialTranscript struct {
	// The text of the partial transcript
	Message string
	// Length of audio this partial transcript applies to
	Duration time.Duration
	// If this is the last partial transcript
	Done bool
}
