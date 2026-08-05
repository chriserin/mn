package oto

import (
	"fmt"
	"time"

	"github.com/ebitengine/purego"
)

// This file exists to convert AudioTimeStamp.mHostTime (a raw
// mach_absolute_time tick count) into a time.Time comparable to Go's
// time.Now(). mach_absolute_time's tick duration isn't fixed across
// hardware (on Apple Silicon it's typically 1 tick = 1ns, but that's not
// guaranteed) — mach_timebase_info gives the numer/denom ratio needed to
// convert ticks to nanoseconds correctly on whatever machine this runs on.

type _MachTimebaseInfo struct {
	numer uint32
	denom uint32
}

var (
	_mach_absolute_time func() uint64
	_mach_timebase_info func(info *_MachTimebaseInfo) int32
)

var machTimebase _MachTimebaseInfo

func initializeMachTimeAPI() error {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}
	purego.RegisterLibFunc(&_mach_absolute_time, lib, "mach_absolute_time")
	purego.RegisterLibFunc(&_mach_timebase_info, lib, "mach_timebase_info")

	if kr := _mach_timebase_info(&machTimebase); kr != 0 {
		return fmt.Errorf("oto: mach_timebase_info failed: %d", kr)
	}
	return nil
}

// hostTimeToWallClock converts an mHostTime reading (as returned by
// AudioQueueGetCurrentTime) into a wall-clock time.Time. It does this by
// taking a fresh (time.Now(), mach_absolute_time()) reference pair right
// now and offsetting from it — called immediately after
// AudioQueueGetCurrentTime returns, so hostTime is always very close to
// "now" and the reference pairing's own jitter (a handful of
// microseconds, from the gap between the two calls below) is negligible.
func hostTimeToWallClock(hostTime uint64) time.Time {
	refWall := time.Now()
	refTicks := _mach_absolute_time()
	deltaTicks := int64(hostTime) - int64(refTicks)
	return refWall.Add(machTicksToDuration(deltaTicks))
}

func machTicksToDuration(ticks int64) time.Duration {
	return time.Duration(ticks * int64(machTimebase.numer) / int64(machTimebase.denom))
}
