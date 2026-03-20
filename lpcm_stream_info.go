package houndify

import (
	"fmt"
	"time"
)

type LPCMStreamInfo struct {
	numChans                  int
	bitDepth                  int
	sampleRate                int
	targetStreamingIntervalMs int

	// Calculated output fields:
	idealChunkSize int // raw, unaligned byte count for the target streaming interval.

	// The client should stream one chunk of chunkSize bytes every streamingInterval.
	chunkSize         int           // frame-aligned byte count that should be streamed each tick.
	streamingInterval time.Duration // the duration represented by chunkSize
}

func (info *LPCMStreamInfo) NumChans() int {
	return info.numChans
}

func (info *LPCMStreamInfo) BitDepth() int {
	return info.bitDepth
}

func (info *LPCMStreamInfo) SampleRate() int {
	return info.sampleRate
}

func (info *LPCMStreamInfo) TargetStreamingIntervalMs() int {
	return info.targetStreamingIntervalMs
}

func (info *LPCMStreamInfo) IdealChunkSize() int {
	return info.idealChunkSize
}

func (info *LPCMStreamInfo) ChunkSize() int {
	return info.chunkSize
}

func (info *LPCMStreamInfo) StreamingInterval() time.Duration {
	return info.streamingInterval
}

// GetLPCMStreamInfo computes the appropriate chunk size and streaming interval
// for streaming linear PCM audio data.
//
// It takes audio parameters (number of channels, bit depth, target streaming interval in
// milliseconds, and sample rate) and calculates the frame-aligned chunk size and the actual
// streaming interval implied by that aligned chunk size. The resulting interval is chosen
// to be as close as possible to the requested target and will often match it exactly.
//
// The function performs three steps:
// 1. Calculates the ideal number of bytes needed to represent the target streaming interval
// 2. Aligns the byte count to full audio frames to ensure valid audio boundaries
// 3. Derives the actual streaming interval from the aligned byte count
//
// Parameters:
//   - numChans: Number of audio channels
//   - bitDepth: Bit depth of each audio sample (e.g., 8, 16, 24, 32)
//   - sampleRate: Sample rate in Hz (e.g., 16000, 44100)
//   - targetStreamingIntervalMs: Target streaming interval in milliseconds
//
// Returns an LPCMStreamInfo containing both the source audio metadata and the
// calculated streaming values:
//   - numChans: the number of audio channels in the source stream
//   - bitDepth: the bits per sample for each channel
//   - sampleRate: the sampling rate in Hz
//   - targetStreamingIntervalMs: the requested streaming cadence in milliseconds
//   - idealChunkSize: the exact byte count for the requested interval before frame alignment
//   - chunkSize: the frame-aligned byte count to write for each chunk
//   - streamingInterval: the duration represented by actualChunkSize
func GetLPCMStreamInfo(
	numChans int,
	bitDepth int,
	sampleRate int,
	targetStreamingIntervalMs int,
) (*LPCMStreamInfo, error) {

	if numChans < 1 {
		return nil,
			fmt.Errorf("invalid input: numChans must be >= 1, got %d", numChans)
	}
	if bitDepth < 8 || (bitDepth%8 != 0) {
		return nil,
			fmt.Errorf("invalid input: bitDepth must be >= 8 and multiple of 8, got %d", bitDepth)
	}
	if sampleRate < 8000 {
		return nil,
			fmt.Errorf("invalid input: sampleRate must be >= 8000, got %d", sampleRate)
	}
	if targetStreamingIntervalMs < 1 {
		return nil,
			fmt.Errorf("invalid input: targetStreamingIntervalMs must be >= 1, got %d",
				targetStreamingIntervalMs)
	}

	bytesPerFrame := numChans * (bitDepth / 8)
	bytesPerSecond := sampleRate * bytesPerFrame

	// Step 1: ideal (non-aligned) byte size
	idealChunkSize := (bytesPerSecond * targetStreamingIntervalMs) / 1000

	// Step 2: align to full frames
	chunkSize := (idealChunkSize / bytesPerFrame) * bytesPerFrame

	// Step 3: derive the actual streaming interval from bytes
	streamingInterval := (time.Duration(chunkSize) * time.Second) / time.Duration(bytesPerSecond)

	return &LPCMStreamInfo{
		numChans:                  numChans,
		bitDepth:                  bitDepth,
		sampleRate:                sampleRate,
		targetStreamingIntervalMs: targetStreamingIntervalMs,

		idealChunkSize:    idealChunkSize,
		chunkSize:         chunkSize,
		streamingInterval: streamingInterval,
	}, nil
}
