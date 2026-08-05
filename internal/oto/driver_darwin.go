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

package oto

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

const (
	float32SizeInBytes = 4

	defaultBufferCount = 4

	// defaultOneBufferSizeInBytes is the default per-buffer size in bytes,
	// carried over from upstream oto (see its comment: 12288 was found
	// necessary at least on iPod touch (7th) and MacBook Pro 2020).
	defaultOneBufferSizeInBytes = 12288

	noErr = 0
)

func newAudioQueue(sampleRate, channelCount, oneBufferSizeInBytes, bufferCount int) (_AudioQueueRef, []_AudioQueueBufferRef, error) {
	desc := _AudioStreamBasicDescription{
		mSampleRate:       float64(sampleRate),
		mFormatID:         uint32(kAudioFormatLinearPCM),
		mFormatFlags:      uint32(kAudioFormatFlagIsFloat),
		mBytesPerPacket:   uint32(channelCount * float32SizeInBytes),
		mFramesPerPacket:  1,
		mBytesPerFrame:    uint32(channelCount * float32SizeInBytes),
		mChannelsPerFrame: uint32(channelCount),
		mBitsPerChannel:   uint32(8 * float32SizeInBytes),
	}

	var audioQueue _AudioQueueRef
	if osstatus := _AudioQueueNewOutput(
		&desc,
		render,
		nil,
		0, //CFRunLoopRef
		0, //CFStringRef
		0,
		&audioQueue); osstatus != noErr {
		return 0, nil, fmt.Errorf("oto: AudioQueueNewFormat with StreamFormat failed: %d", osstatus)
	}

	bufs := make([]_AudioQueueBufferRef, 0, bufferCount)
	for len(bufs) < cap(bufs) {
		var buf _AudioQueueBufferRef
		if osstatus := _AudioQueueAllocateBuffer(audioQueue, uint32(oneBufferSizeInBytes), &buf); osstatus != noErr {
			return 0, nil, fmt.Errorf("oto: AudioQueueAllocateBuffer failed: %d", osstatus)
		}
		buf.mAudioDataByteSize = uint32(oneBufferSizeInBytes)
		bufs = append(bufs, buf)
	}

	return audioQueue, bufs, nil
}

type context struct {
	audioQueue      _AudioQueueRef
	unqueuedBuffers []_AudioQueueBufferRef

	sampleRate           int
	channelCount         int
	oneBufferSizeInBytes int

	// pipelineLatency is the queried device+safety-offset delay between a
	// buffer being handed to CoreAudio and its audio reaching the speaker
	// (see hwinfo_darwin.go). Added in predictAudibleTime.
	pipelineLatency time.Duration

	// predictionOffset is NewContextOptions.PredictionOffset: a manual
	// calibration constant added on top of pipelineLatency in
	// predictAudibleTime, for whatever gap mic measurement finds that
	// queried device properties don't explain (see that field's doc
	// comment in context.go).
	predictionOffset time.Duration

	// fill produces each buffer's audio; nil means "always silence".
	fill FillFunc

	// framesWritten is a running count of frames handed to fill so far,
	// used as each call's startFrame. It only ever increases.
	framesWritten int64

	// events is where ClickEvents are delivered; nil if Fill was nil.
	events chan ClickEvent

	cond *sync.Cond

	toPause  bool
	toResume bool

	err atomicError
}

// TODO: Convert the error code correctly.
// See https://stackoverflow.com/questions/2196869/how-do-you-convert-an-iphone-osstatus-code-to-something-useful

var theContext *context

func newContext(sampleRate int, channelCount int, oneBufferSizeInBytes int, bufferCount int, pipelineLatency time.Duration, predictionOffset time.Duration, fill FillFunc) (*context, chan struct{}, error) {
	ready := make(chan struct{})

	var events chan ClickEvent
	if fill != nil {
		events = make(chan ClickEvent, 16)
	}

	c := &context{
		cond:                 sync.NewCond(&sync.Mutex{}),
		sampleRate:           sampleRate,
		channelCount:         channelCount,
		oneBufferSizeInBytes: oneBufferSizeInBytes,
		pipelineLatency:      pipelineLatency,
		predictionOffset:     predictionOffset,
		fill:                 fill,
		events:               events,
	}
	theContext = c

	if err := initializeAPI(); err != nil {
		return nil, nil, err
	}
	if fill != nil {
		if err := initializeMachTimeAPI(); err != nil {
			return nil, nil, err
		}
	}

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		defer func() {
			if ready != nil {
				close(ready)
			}
		}()

		q, bs, err := newAudioQueue(sampleRate, channelCount, oneBufferSizeInBytes, bufferCount)
		if err != nil {
			c.err.TryStore(err)
			return
		}
		c.audioQueue = q
		c.unqueuedBuffers = bs

		if err := setNotificationHandler(); err != nil {
			c.err.TryStore(err)
			return
		}

		var retryCount int
	try:
		if osstatus := _AudioQueueStart(c.audioQueue, nil); osstatus != noErr {
			if osstatus == avAudioSessionErrorCodeCannotStartPlaying && retryCount < 100 {
				// TODO: use sleepTime() after investigating when this error happens.
				time.Sleep(10 * time.Millisecond)
				retryCount++
				goto try
			}
			c.err.TryStore(fmt.Errorf("oto: AudioQueueStart failed at newContext: %d", osstatus))
			return
		}

		close(ready)
		ready = nil

		c.loop()
	}()

	return c, ready, nil
}

func (c *context) wait() bool {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	for len(c.unqueuedBuffers) == 0 && c.err.Load() == nil && !c.toPause && !c.toResume {
		c.cond.Wait()
	}
	return c.err.Load() == nil
}

func (c *context) loop() {
	// buf32 is reused across every fill call, so it must be cleared before
	// each one: otherwise a buffer that got a click last time around
	// would still have that click's tail sitting in it this time, even if
	// this call's Fill doesn't write anything new.
	buf32 := make([]float32, c.oneBufferSizeInBytes/float32SizeInBytes)
	for {
		if !c.wait() {
			return
		}
		c.appendBuffer(buf32)
	}
}

func (c *context) appendBuffer(buf32 []float32) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	if c.err.Load() != nil {
		return
	}

	if c.toPause {
		if err := c.pause(); err != nil {
			c.err.TryStore(err)
		}
		c.toPause = false
		return
	}

	if c.toResume {
		if err := c.resume(); err != nil {
			c.err.TryStore(err)
		}
		c.toResume = false
		return
	}

	buf := c.unqueuedBuffers[0]
	copy(c.unqueuedBuffers, c.unqueuedBuffers[1:])
	c.unqueuedBuffers = c.unqueuedBuffers[:len(c.unqueuedBuffers)-1]

	for i := range buf32 {
		buf32[i] = 0
	}
	startFrame := c.framesWritten
	var offsetFrames, clickIndex int
	var hasClick bool
	if c.fill != nil {
		offsetFrames, clickIndex, hasClick = c.fill(startFrame, buf32, c.channelCount)
	}
	c.framesWritten += int64(len(buf32) / c.channelCount)

	copy(unsafe.Slice((*float32)(unsafe.Pointer(buf.mAudioData)), buf.mAudioDataByteSize/float32SizeInBytes), buf32)

	if osstatus := _AudioQueueEnqueueBuffer(c.audioQueue, buf, 0, nil); osstatus != noErr {
		c.err.TryStore(fmt.Errorf("oto: AudioQueueEnqueueBuffer failed: %d", osstatus))
		return
	}

	// Deliberately done here, right when Fill discovers a click, rather
	// than waiting for that buffer to actually start playing: the click's
	// absolute sample position is already known now, and
	// AudioQueueGetCurrentTime can map it to a predicted wall-clock time
	// immediately. That also means this sample is always at most one
	// buffer-interval away from when it'll actually play (Fill only ever
	// looks one buffer ahead), which keeps the extrapolation window short
	// enough that clock drift within it is negligible — see
	// predictAudibleTime.
	// predictAudibleTime can fail for the very first buffer of the whole
	// stream: AudioQueueGetCurrentTime has no valid sample/host-time
	// mapping to report until the device has actually started running,
	// which it hasn't yet at the instant the first buffer is enqueued
	// (confirmed empirically: AudioQueueGetCurrentTime returns osstatus
	// -66678 in exactly that case). A click landing in buffer 0 is
	// silently dropped as a result — the same practical gap the previous
	// render()-based design had, now for a different underlying reason.
	if hasClick && c.events != nil {
		if predicted, err := c.predictAudibleTime(startFrame + int64(offsetFrames)); err == nil {
			select {
			case c.events <- ClickEvent{ClickIndex: clickIndex, PredictedAudible: predicted}:
			default:
			}
		}
	}
}

// predictAudibleTime maps sampleIndex (an absolute frame count since the
// stream began) to a predicted wall-clock time, using
// AudioQueueGetCurrentTime's live (sample position, host time, rate
// scalar) snapshot rather than our own render()-callback timestamps.
// mRateScalar corrects for the device's actual playback rate versus
// nominal, which matters for correctness but — because this is always
// called with sampleIndex at most one buffer-interval in the future
// (see appendBuffer) — isn't load-bearing for accuracy here the way it
// would be for a long extrapolation.
func (c *context) predictAudibleTime(sampleIndex int64) (time.Time, error) {
	var ts _AudioTimeStamp
	if osstatus := _AudioQueueGetCurrentTime(c.audioQueue, 0, &ts, nil); osstatus != noErr {
		return time.Time{}, fmt.Errorf("oto: AudioQueueGetCurrentTime failed: %d", osstatus)
	}

	nowWall := hostTimeToWallClock(ts.mHostTime)

	effectiveSampleRate := float64(c.sampleRate) * ts.mRateScalar
	framesAhead := float64(sampleIndex) - ts.mSampleTime
	secondsAhead := framesAhead / effectiveSampleRate

	return nowWall.
		Add(time.Duration(secondsAhead * float64(time.Second))).
		Add(c.pipelineLatency).
		Add(c.predictionOffset), nil
}

func (c *context) Suspend() error {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	if err := c.err.Load(); err != nil {
		return err.(error)
	}
	c.toPause = true
	c.toResume = false
	c.cond.Signal()
	return nil
}

func (c *context) Resume() error {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	if err := c.err.Load(); err != nil {
		return err.(error)
	}
	c.toPause = false
	c.toResume = true
	c.cond.Signal()
	return nil
}

func (c *context) pause() error {
	if osstatus := _AudioQueuePause(c.audioQueue); osstatus != noErr {
		return fmt.Errorf("oto: AudioQueuePause failed: %d", osstatus)
	}
	return nil
}

func (c *context) resume() error {
	var retryCount int
try:
	if osstatus := _AudioQueueStart(c.audioQueue, nil); osstatus != noErr {
		if (osstatus == avAudioSessionErrorCodeCannotStartPlaying ||
			osstatus == avAudioSessionErrorCodeCannotInterruptOthers) &&
			retryCount < 30 {
			// It is uncertain that this error is temporary or not. Then let's use exponential-time sleeping.
			time.Sleep(sleepTime(retryCount))
			retryCount++
			goto try
		}
		if osstatus == avAudioSessionErrorCodeSiriIsRecording {
			// As this error should be temporary, it should be OK to use a short time for sleep anytime.
			time.Sleep(10 * time.Millisecond)
			goto try
		}
		return fmt.Errorf("oto: AudioQueueStart failed at Resume: %d", osstatus)
	}
	return nil
}

func (c *context) Err() error {
	if err := c.err.Load(); err != nil {
		return err.(error)
	}
	return nil
}

// render is CoreAudio's callback, invoked on its own real-time audio thread
// each time it finishes consuming a buffer and needs it back. It does
// nothing but hand the buffer back to appendBuffer — ClickEvents are
// computed and sent from appendBuffer instead (see predictAudibleTime),
// using AudioQueueGetCurrentTime rather than this callback's own timing,
// so render() has no reason to do anything beyond buffer bookkeeping.
func render(inUserData unsafe.Pointer, inAQ _AudioQueueRef, inBuffer _AudioQueueBufferRef) {
	theContext.cond.L.Lock()
	theContext.unqueuedBuffers = append(theContext.unqueuedBuffers, inBuffer)
	theContext.cond.Signal()
	theContext.cond.L.Unlock()
}

func setGlobalPause(self objc.ID, _cmd objc.SEL, notification objc.ID) {
	theContext.Suspend()
}

func setGlobalResume(self objc.ID, _cmd objc.SEL, notification objc.ID) {
	theContext.Resume()
}

func sleepTime(count int) time.Duration {
	switch count {
	case 0:
		return 10 * time.Millisecond
	case 1:
		return 20 * time.Millisecond
	case 2:
		return 50 * time.Millisecond
	default:
		return 100 * time.Millisecond
	}
}
