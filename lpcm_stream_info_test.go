package houndify_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	houndify "github.com/soundhound/houndify-sdk-go"
)

func TestGetLPCMStreamInfo(t *testing.T) {
	const (
		numChans = 1
		bitDepth = 16
	)

	testCases := []struct {
		name                      string
		targetStreamingIntervalMs int
		sampleRate                int
		idealChunkSize            int
		chunkSize                 int
		streamingInterval         time.Duration
	}{
		{
			name:                      "20ms_8000hz",
			targetStreamingIntervalMs: 20,
			sampleRate:                8000,
			idealChunkSize:            320,
			chunkSize:                 320,
			streamingInterval:         20 * time.Millisecond,
		},
		{
			name:                      "20ms_11025hz",
			targetStreamingIntervalMs: 20,
			sampleRate:                11025,
			idealChunkSize:            441,
			chunkSize:                 440,
			streamingInterval:         19954648 * time.Nanosecond,
		},
		{
			name:                      "20ms_16000hz",
			targetStreamingIntervalMs: 20,
			sampleRate:                16000,
			idealChunkSize:            640,
			chunkSize:                 640,
			streamingInterval:         20 * time.Millisecond,
		},
		{
			name:                      "20ms_20500hz",
			targetStreamingIntervalMs: 20,
			sampleRate:                20500,
			idealChunkSize:            820,
			chunkSize:                 820,
			streamingInterval:         20 * time.Millisecond,
		},
		{
			name:                      "20ms_21000hz",
			targetStreamingIntervalMs: 20,
			sampleRate:                21000,
			idealChunkSize:            840,
			chunkSize:                 840,
			streamingInterval:         20 * time.Millisecond,
		},
		{
			name:                      "20ms_32000hz",
			targetStreamingIntervalMs: 20,
			sampleRate:                32000,
			idealChunkSize:            1280,
			chunkSize:                 1280,
			streamingInterval:         20 * time.Millisecond,
		},
		{
			name:                      "20ms_44100hz",
			targetStreamingIntervalMs: 20,
			sampleRate:                44100,
			idealChunkSize:            1764,
			chunkSize:                 1764,
			streamingInterval:         20 * time.Millisecond,
		},
		{
			name:                      "20ms_48000hz",
			targetStreamingIntervalMs: 20,
			sampleRate:                48000,
			idealChunkSize:            1920,
			chunkSize:                 1920,
			streamingInterval:         20 * time.Millisecond,
		},
		{
			name:                      "30ms_8000hz",
			targetStreamingIntervalMs: 30,
			sampleRate:                8000,
			idealChunkSize:            480,
			chunkSize:                 480,
			streamingInterval:         30 * time.Millisecond,
		},
		{
			name:                      "30ms_11025hz",
			targetStreamingIntervalMs: 30,
			sampleRate:                11025,
			idealChunkSize:            661,
			chunkSize:                 660,
			streamingInterval:         29931972 * time.Nanosecond,
		},
		{
			name:                      "30ms_16000hz",
			targetStreamingIntervalMs: 30,
			sampleRate:                16000,
			idealChunkSize:            960,
			chunkSize:                 960,
			streamingInterval:         30 * time.Millisecond,
		},
		{
			name:                      "30ms_20500hz",
			targetStreamingIntervalMs: 30,
			sampleRate:                20500,
			idealChunkSize:            1230,
			chunkSize:                 1230,
			streamingInterval:         30 * time.Millisecond,
		},
		{
			name:                      "30ms_21000hz",
			targetStreamingIntervalMs: 30,
			sampleRate:                21000,
			idealChunkSize:            1260,
			chunkSize:                 1260,
			streamingInterval:         30 * time.Millisecond,
		},
		{
			name:                      "30ms_32000hz",
			targetStreamingIntervalMs: 30,
			sampleRate:                32000,
			idealChunkSize:            1920,
			chunkSize:                 1920,
			streamingInterval:         30 * time.Millisecond,
		},
		{
			name:                      "30ms_44100hz",
			targetStreamingIntervalMs: 30,
			sampleRate:                44100,
			idealChunkSize:            2646,
			chunkSize:                 2646,
			streamingInterval:         30 * time.Millisecond,
		},
		{
			name:                      "30ms_48000hz",
			targetStreamingIntervalMs: 30,
			sampleRate:                48000,
			idealChunkSize:            2880,
			chunkSize:                 2880,
			streamingInterval:         30 * time.Millisecond,
		},
		{
			name:                      "33ms_8000hz",
			targetStreamingIntervalMs: 33,
			sampleRate:                8000,
			idealChunkSize:            528,
			chunkSize:                 528,
			streamingInterval:         33 * time.Millisecond,
		},
		{
			name:                      "33ms_11025hz",
			targetStreamingIntervalMs: 33,
			sampleRate:                11025,
			idealChunkSize:            727,
			chunkSize:                 726,
			streamingInterval:         32925170 * time.Nanosecond,
		},
		{
			name:                      "33ms_16000hz",
			targetStreamingIntervalMs: 33,
			sampleRate:                16000,
			idealChunkSize:            1056,
			chunkSize:                 1056,
			streamingInterval:         33 * time.Millisecond,
		},
		{
			name:                      "33ms_20500hz",
			targetStreamingIntervalMs: 33,
			sampleRate:                20500,
			idealChunkSize:            1353,
			chunkSize:                 1352,
			streamingInterval:         32975609 * time.Nanosecond,
		},
		{
			name:                      "33ms_21000hz",
			targetStreamingIntervalMs: 33,
			sampleRate:                21000,
			idealChunkSize:            1386,
			chunkSize:                 1386,
			streamingInterval:         33 * time.Millisecond,
		},
		{
			name:                      "33ms_32000hz",
			targetStreamingIntervalMs: 33,
			sampleRate:                32000,
			idealChunkSize:            2112,
			chunkSize:                 2112,
			streamingInterval:         33 * time.Millisecond,
		},
		{
			name:                      "33ms_44100hz",
			targetStreamingIntervalMs: 33,
			sampleRate:                44100,
			idealChunkSize:            2910,
			chunkSize:                 2910,
			streamingInterval:         32993197 * time.Nanosecond,
		},
		{
			name:                      "33ms_48000hz",
			targetStreamingIntervalMs: 33,
			sampleRate:                48000,
			idealChunkSize:            3168,
			chunkSize:                 3168,
			streamingInterval:         33 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chunkInfo, err := houndify.GetLPCMStreamInfo(
				numChans,
				bitDepth,
				tc.sampleRate,
				tc.targetStreamingIntervalMs,
			)
			assert.NoError(t, err)

			assert.Equal(t, chunkInfo.NumChans(), numChans)
			assert.Equal(t, chunkInfo.BitDepth(), bitDepth)
			assert.Equal(t, chunkInfo.SampleRate(), tc.sampleRate)
			assert.Equal(t, chunkInfo.TargetStreamingIntervalMs(), tc.targetStreamingIntervalMs)
			assert.Equal(t, chunkInfo.IdealChunkSize(), tc.idealChunkSize)
			assert.Equal(t, chunkInfo.ChunkSize(), tc.chunkSize)
			assert.Equal(t, chunkInfo.StreamingInterval(), tc.streamingInterval)
		})
	}
}

func TestGetLPCMStreamInfoRejectsInvalidInput(t *testing.T) {
	testCases := []struct {
		name                      string
		numChans                  int
		bitDepth                  int
		sampleRate                int
		targetStreamingIntervalMs int
		errSubstring              string
	}{
		{
			name:                      "zero_channels",
			numChans:                  0,
			bitDepth:                  16,
			sampleRate:                16000,
			targetStreamingIntervalMs: 20,
			errSubstring:              "invalid input: numChans must be >= 1",
		},
		{
			name:                      "bit_depth_too_small",
			numChans:                  1,
			bitDepth:                  0,
			sampleRate:                16000,
			targetStreamingIntervalMs: 20,
			errSubstring:              "invalid input: bitDepth must be >= 8 and multiple of 8",
		},
		{
			name:                      "bit_depth_not_multiple_of_8",
			numChans:                  1,
			bitDepth:                  10,
			sampleRate:                16000,
			targetStreamingIntervalMs: 20,
			errSubstring:              "invalid input: bitDepth must be >= 8 and multiple of 8",
		},
		{
			name:                      "zero_sample_rate",
			numChans:                  1,
			bitDepth:                  16,
			sampleRate:                0,
			targetStreamingIntervalMs: 20,
			errSubstring:              "invalid input: sampleRate must be >= 8000",
		},
		{
			name:                      "sample_rate_too_small",
			numChans:                  1,
			bitDepth:                  16,
			sampleRate:                1000,
			targetStreamingIntervalMs: 20,
			errSubstring:              "invalid input: sampleRate must be >= 8000",
		},
		{
			name:                      "zero_target_interval",
			numChans:                  1,
			bitDepth:                  16,
			sampleRate:                16000,
			targetStreamingIntervalMs: 0,
			errSubstring:              "invalid input: targetStreamingIntervalMs must be >= 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chunkInfo, err := houndify.GetLPCMStreamInfo(
				tc.numChans,
				tc.bitDepth,
				tc.sampleRate,
				tc.targetStreamingIntervalMs,
			)
			assert.Error(t, err)
			assert.Nil(t, chunkInfo)
			assert.ErrorContains(t, err, tc.errSubstring)
		})
	}
}
