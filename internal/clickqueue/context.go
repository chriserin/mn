// Copyright 2021 The Oto Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// NOTICE: This file has been modified from its original form (part of
// github.com/hajimehoshi/oto's macOS AudioQueue driver). See README.md in
// this directory for a summary of what changed and why.

// Package clickqueue is a stripped-down fork of github.com/hajimehoshi/oto's
// macOS AudioQueue driver, vendored so its internals can be instrumented and
// extended directly.
//
// Unlike upstream oto, this fork has no mux/Player layer, no io.Reader
// source, and no format conversion — a single Fill callback (see
// NewContextOptions.Fill) is asked to write each buffer's worth of
// samples directly. It also reports, via Events, the predicted wall-clock
// instant a click actually becomes audible, computed via
// AudioQueueGetCurrentTime (see driver_darwin.go's predictAudibleTime)
// rather than a separate wall-clock timer — ported from the mndriverpoc
// POC, which validated this approach (including empirically, via
// mic-loopback measurement) before it was brought into mn.
package clickqueue

import (
	"fmt"
	"sync"
	"time"
)

var (
	contextCreated       bool
	contextCreationMutex sync.Mutex
)

// Context is the main object of this package. Creating a Context starts an
// AudioQueue immediately and keeps it fed via Fill until Suspend is called
// or the process exits.
//
// Creating multiple contexts is NOT supported.
type Context struct {
	context *context
}

// FillFunc is asked to fill frames (interleaved, channelCount channels per
// frame, all zeroed before the call) with the next len(frames)/channelCount
// frames of audio, starting at startFrame (a running count of frames
// produced since the stream began — never resets). It returns whether a
// click's onset falls within this buffer and, if so, the frame offset
// within it and a caller-defined index identifying the click (e.g. a beat
// number), which are echoed back in the corresponding ClickEvent.
//
// Fill is only ever called from this package's single fill goroutine,
// strictly in playback order — implementations don't need their own
// locking as long as they aren't shared across contexts.
type FillFunc func(startFrame int64, frames []float32, channelCount int) (offsetFrames int, clickIndex int, hasClick bool)

// ClickEvent reports that a click scheduled via FillFunc is about to
// become audible.
type ClickEvent struct {
	// ClickIndex is whatever FillFunc returned as clickIndex.
	ClickIndex int

	// PredictedAudible is the estimated wall-clock instant the click
	// actually reaches the speaker: the render() callback timestamp for
	// the buffer's start, plus the click's offset within that buffer,
	// plus the device's measured output pipeline latency (see
	// hwinfo_darwin.go). It's a prediction, not a confirmation — there's
	// no feedback path that verifies actual playback.
	PredictedAudible time.Time
}

// NewContextOptions represents options for NewContext.
type NewContextOptions struct {
	// SampleRate specifies the number of samples that should be played during one second.
	// Usual numbers are 44100 or 48000.
	SampleRate int

	// ChannelCount specifies the number of channels. One channel is mono playback. Two
	// channels are stereo playback. No other values are supported.
	ChannelCount int

	// BufferSize specifies the total buffered duration across all rotating
	// buffers (i.e. BufferSize / BufferCount is the duration of one
	// buffer). If 0, the driver's default per-buffer size is used instead.
	BufferSize time.Duration

	// BufferCount specifies how many buffers rotate through the AudioQueue.
	// If 0, a default of 4 is used. Fewer buffers means lower latency
	// (see the (N-1)*bufferInterval discussion this package's README
	// covers) but less slack before a slow Fill call causes an audible
	// underrun.
	BufferCount int

	// DisableHardwareAlignment disables rounding the per-buffer duration up
	// to the nearest multiple of the default output device's actual
	// hardware I/O period, as queried via CoreAudio's HAL (see
	// hwinfo_darwin.go). Alignment is enabled by default: a per-buffer
	// duration smaller than that period causes CoreAudio to hand back
	// several buffers to render() in one burst instead of one per period,
	// which is visible as near-zero rotate intervals in the logs. Disable
	// this to observe that un-aligned behavior directly.
	DisableHardwareAlignment bool

	// Fill is called once per buffer rotation to produce the next chunk
	// of audio. If nil, every buffer is silence (all-zero) and no
	// ClickEvents are ever emitted.
	Fill FillFunc

	// PredictionOffset is added to every ClickEvent.PredictedAudible as a
	// manual calibration constant, on top of everything predictAudibleTime
	// already accounts for. Positive shifts predictions later (use this
	// if the UI is firing ahead of the audible click, as measured via
	// mic-loopback — see internal/oto/recorder_darwin.go); negative
	// shifts them earlier. Exists because measured drift (consistently
	// ~40ms on the machine this was tuned on, and confirmed independent
	// of BufferCount) isn't yet explained by any queryable CoreAudio
	// property, so this is an empirical fudge factor, not a principled
	// correction — it may need to be re-measured on different hardware.
	PredictionOffset time.Duration
}

// NewContext creates a new context with given options and immediately
// starts pulling audio from options.Fill. It returns a context, a channel
// that is closed when the context is ready, and an error if one occurred.
//
// Creating multiple contexts is NOT supported.
func NewContext(options *NewContextOptions) (*Context, chan struct{}, error) {
	contextCreationMutex.Lock()
	defer contextCreationMutex.Unlock()

	if contextCreated {
		return nil, nil, fmt.Errorf("oto: context is already created")
	}
	contextCreated = true

	bufferCount := options.BufferCount
	if bufferCount == 0 {
		bufferCount = defaultBufferCount
	}

	bytesPerSample := options.ChannelCount * float32SizeInBytes

	var oneBufferDuration time.Duration
	if options.BufferSize != 0 {
		oneBufferDuration = options.BufferSize / time.Duration(bufferCount)
	} else {
		frames := defaultOneBufferSizeInBytes / bytesPerSample
		oneBufferDuration = time.Duration(int64(frames) * int64(time.Second) / int64(options.SampleRate))
	}

	// Query the default output device's actual hardware I/O period and
	// pipeline latency. The period is independent of SampleRate/BufferSize
	// above — it's the real quantum the device pulls audio in, and is
	// what render() callbacks end up paced against once our own buffers
	// are smaller than it. The pipeline latency is reused later to
	// compute ClickEvent.PredictedAudible.
	hw, hwErr := queryDefaultOutputDeviceInfo()
	if hwErr == nil && hw.Quantum > 0 && !options.DisableHardwareAlignment {
		oneBufferDuration = alignUp(oneBufferDuration, hw.Quantum)
	}

	oneBufferSizeInBytes := int(int64(oneBufferDuration) * int64(options.SampleRate) * int64(bytesPerSample) / int64(time.Second))
	oneBufferSizeInBytes = oneBufferSizeInBytes / bytesPerSample * bytesPerSample

	ctx, ready, err := newContext(options.SampleRate, options.ChannelCount, oneBufferSizeInBytes, bufferCount, hw.PipelineLatency, options.PredictionOffset, options.Fill)
	if err != nil {
		return nil, nil, err
	}
	return &Context{context: ctx}, ready, nil
}

// Suspend suspends the entire audio play.
//
// Suspend is concurrent-safe.
func (c *Context) Suspend() error {
	return c.context.Suspend()
}

// Resume resumes the entire audio play, which was suspended by Suspend.
//
// Resume is concurrent-safe.
func (c *Context) Resume() error {
	return c.context.Resume()
}

// Err returns the current error.
//
// Err is concurrent-safe.
func (c *Context) Err() error {
	return c.context.Err()
}

// Events returns the channel ClickEvents are delivered on. It's buffered
// and non-blocking on the send side (see render() in driver_darwin.go):
// if a receiver isn't keeping up, events are dropped rather than stalling
// the audio callback.
func (c *Context) Events() <-chan ClickEvent {
	return c.context.events
}

// alignUp rounds d up to the nearest multiple of quantum. If quantum or d
// is non-positive, d is returned unchanged.
func alignUp(d, quantum time.Duration) time.Duration {
	if quantum <= 0 || d <= 0 {
		return d
	}
	n := (d + quantum - 1) / quantum
	if n < 1 {
		n = 1
	}
	return n * quantum
}

type atomicError struct {
	err error
	m   sync.Mutex
}

func (a *atomicError) TryStore(err error) {
	a.m.Lock()
	defer a.m.Unlock()
	if a.err == nil {
		a.err = err
	}
}

func (a *atomicError) Load() error {
	a.m.Lock()
	defer a.m.Unlock()
	return a.err
}
