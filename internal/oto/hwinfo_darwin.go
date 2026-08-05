// Copyright 2026 The Oto Authors
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
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

// This file has nothing to do with driving playback — it exists purely to
// ask CoreAudio's HAL, via AudioObjectGetPropertyData, what the default
// output device is actually doing: its native buffer size/sample rate
// (hwQuantum, used to align our own buffer size so render() fires once per
// hardware period instead of in bursts — see context.go's alignUp) and its
// fixed device+safety-offset latency (pipelineLatency, the delay between a
// buffer being handed back by render() and its audio reaching the speaker).
// Both are queried once at startup and reused from context.go/driver_darwin.go.

const (
	kAudioObjectSystemObject        = 1
	kAudioObjectPropertyElementMain = 0

	kAudioObjectPropertyScopeGlobal = 0x676C6F62 // 'glob'
	kAudioObjectPropertyScopeOutput = 0x6F757470 // 'outp'
	kAudioObjectPropertyScopeInput  = 0x696E7074 // 'inpt'

	kAudioHardwarePropertyDefaultOutputDevice = 0x644F7574 // 'dOut'
	kAudioHardwarePropertyDefaultInputDevice  = 0x64496E20 // 'dIn '
	kAudioDevicePropertyActualSampleRate      = 0x61737274 // 'asrt'
	kAudioDevicePropertyBufferFrameSize       = 0x6673697A // 'fsiz'

	// kAudioDevicePropertyLatency is the device's own fixed processing
	// delay (in frames) between handing CoreAudio a buffer and that
	// buffer's audio reaching the output. kAudioDevicePropertySafetyOffset
	// is an additional, separate frame count CoreAudio holds back as a
	// margin against underruns. Both are per-scope (input vs output can
	// differ), hence kAudioObjectPropertyScopeOutput rather than Global.
	kAudioDevicePropertyLatency      = 0x6C746E63 // 'ltnc'
	kAudioDevicePropertySafetyOffset = 0x73616674 // 'saft'
)

type _AudioObjectPropertyAddress struct {
	mSelector uint32
	mScope    uint32
	mElement  uint32
}

var _AudioObjectGetPropertyData func(inObjectID uint32, inAddress *_AudioObjectPropertyAddress, inQualifierDataSize uint32, inQualifierData unsafe.Pointer, ioDataSize *uint32, outData unsafe.Pointer) uintptr

func initializeHWInfoAPI() error {
	coreAudio, err := purego.Dlopen("/System/Library/Frameworks/CoreAudio.framework/CoreAudio", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}
	purego.RegisterLibFunc(&_AudioObjectGetPropertyData, coreAudio, "AudioObjectGetPropertyData")
	return nil
}

func getUint32Property(objectID uint32, selector uint32, scope uint32) (uint32, error) {
	addr := _AudioObjectPropertyAddress{mSelector: selector, mScope: scope, mElement: kAudioObjectPropertyElementMain}
	var value uint32
	size := uint32(unsafe.Sizeof(value))
	if osstatus := _AudioObjectGetPropertyData(objectID, &addr, 0, nil, &size, unsafe.Pointer(&value)); osstatus != noErr {
		return 0, fmt.Errorf("AudioObjectGetPropertyData failed for selector %#x: %d", selector, osstatus)
	}
	return value, nil
}

func getFloat64Property(objectID uint32, selector uint32, scope uint32) (float64, error) {
	addr := _AudioObjectPropertyAddress{mSelector: selector, mScope: scope, mElement: kAudioObjectPropertyElementMain}
	var value float64
	size := uint32(unsafe.Sizeof(value))
	if osstatus := _AudioObjectGetPropertyData(objectID, &addr, 0, nil, &size, unsafe.Pointer(&value)); osstatus != noErr {
		return 0, fmt.Errorf("AudioObjectGetPropertyData failed for selector %#x: %d", selector, osstatus)
	}
	return value, nil
}

// hwInfo is what queryDefaultOutputDeviceInfo discovers about the default
// output device.
type hwInfo struct {
	// Quantum is the device's real I/O period (bufferFrameSize /
	// actualSampleRate) — buffer sizes should be aligned to a multiple of
	// this (see context.go's alignUp) for render() to fire once per
	// period instead of in bursts.
	Quantum time.Duration

	// PipelineLatency is the fixed delay between a buffer being handed
	// back by render() and its audio actually reaching the speaker
	// (device latency + safety offset, both in frames, converted using
	// the device's actual sample rate).
	PipelineLatency time.Duration
}

// queryDefaultOutputDeviceInfo queries the default output device's actual
// sample rate, hardware I/O buffer frame size, and output pipeline
// latency, returning them as an hwInfo. A zero-value hwInfo and non-nil
// error are returned if the query fails for any reason (e.g. sandboxing);
// callers should treat that as "unavailable", not fatal.
func queryDefaultOutputDeviceInfo() (hwInfo, error) {
	if err := initializeHWInfoAPI(); err != nil {
		return hwInfo{}, err
	}

	deviceID, err := getUint32Property(kAudioObjectSystemObject, kAudioHardwarePropertyDefaultOutputDevice, kAudioObjectPropertyScopeGlobal)
	if err != nil {
		return hwInfo{}, err
	}

	frameSize, err := getUint32Property(deviceID, kAudioDevicePropertyBufferFrameSize, kAudioObjectPropertyScopeGlobal)
	if err != nil {
		return hwInfo{}, err
	}

	sampleRate, err := getFloat64Property(deviceID, kAudioDevicePropertyActualSampleRate, kAudioObjectPropertyScopeGlobal)
	if err != nil {
		return hwInfo{}, err
	}

	quantum := time.Duration(float64(frameSize) / sampleRate * float64(time.Second))

	var pipelineLatency time.Duration
	latencyFrames, latErr := getUint32Property(deviceID, kAudioDevicePropertyLatency, kAudioObjectPropertyScopeOutput)
	safetyFrames, safErr := getUint32Property(deviceID, kAudioDevicePropertySafetyOffset, kAudioObjectPropertyScopeOutput)
	if latErr == nil && safErr == nil {
		pipelineLatency = time.Duration(float64(latencyFrames+safetyFrames) / sampleRate * float64(time.Second))
	}

	return hwInfo{Quantum: quantum, PipelineLatency: pipelineLatency}, nil
}

// queryDefaultInputPipelineLatency is queryDefaultOutputDeviceInfo's
// counterpart for the default input device: the fixed delay between sound
// actually reaching the mic and a sample carrying it becoming available
// to us (device latency + safety offset, input scope, converted using the
// input device's own actual sample rate — which need not match the
// output device's). recorder_darwin.go subtracts this from detected
// onset times, mirroring how pipelineLatency is added on the output side.
func queryDefaultInputPipelineLatency() (time.Duration, error) {
	if err := initializeHWInfoAPI(); err != nil {
		return 0, err
	}

	deviceID, err := getUint32Property(kAudioObjectSystemObject, kAudioHardwarePropertyDefaultInputDevice, kAudioObjectPropertyScopeGlobal)
	if err != nil {
		return 0, err
	}

	sampleRate, err := getFloat64Property(deviceID, kAudioDevicePropertyActualSampleRate, kAudioObjectPropertyScopeGlobal)
	if err != nil {
		return 0, err
	}

	latencyFrames, latErr := getUint32Property(deviceID, kAudioDevicePropertyLatency, kAudioObjectPropertyScopeInput)
	safetyFrames, safErr := getUint32Property(deviceID, kAudioDevicePropertySafetyOffset, kAudioObjectPropertyScopeInput)
	if latErr != nil || safErr != nil {
		return 0, fmt.Errorf("could not query input pipeline latency: latencyErr=%v safetyOffsetErr=%v", latErr, safErr)
	}

	return time.Duration(float64(latencyFrames+safetyFrames) / sampleRate * float64(time.Second)), nil
}
