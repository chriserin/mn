package main

import (
	"strings"
	"testing"
)

func elapseMeasures(m Model, n int) Model {
	return elapseBeats(m, n*beatsPerMeasure)
}

func elapseBeats(m Model, n int) Model {
	for i := 0; i < n; i++ {
		next, _ := m.Update(beatMsg{})
		m = next.(Model)
	}
	return m
}

// elapseMeasuresThenLand elapses n full measures (where a pending tempo-
// training step may be marked) and then strikes beat 1 of the next measure,
// where any pending step actually lands and the BPM readout updates.
func elapseMeasuresThenLand(m Model, n int) Model {
	return elapseBeats(elapseMeasures(m, n), 1)
}

func assertRow(t *testing.T, m Model, label, want string) {
	t.Helper()
	got := m.tempoTrainingRow(label)
	if got != want {
		t.Errorf("expected %s row %q, got %q", label, want, got)
	}
}

func assertHeader(t *testing.T, m Model, want string) {
	t.Helper()
	got := m.tempoTrainingHeader()
	if got != want {
		t.Errorf("expected tempo training header %q, got %q", want, got)
	}
}

func assertTableShown(t *testing.T, m Model, shown bool) {
	t.Helper()
	if m.tempoTrainingOn != shown {
		t.Errorf("expected tempo training table shown=%v, got tempoTrainingOn=%v", shown, m.tempoTrainingOn)
	}
	view := m.View().Content
	hasTable := strings.Contains(view, "Start")
	if hasTable != shown {
		t.Errorf("expected tempo training blocks rendered=%v, got rendered=%v in view:\n%s", shown, hasTable, view)
	}
}

// @ft:16
func TestEnableTempoTraining(t *testing.T) {
	m := New()
	m.playing = true
	assertHeader(t, m, "Tempo Training: off")

	m = press(m, "t")

	assertHeader(t, m, "Tempo Training: on")
	assertTableShown(t, m, true)
}

// Measures that elapse before tempo training is turned on must not count
// toward the first interval — otherwise the first step lands early.
// @ft:61
func TestMeasuresElapsedBeforeEnablingDoNotCountTowardFirstInterval(t *testing.T) {
	m := New()
	m.stepBPM = 10
	m.stepIntervalMeasures = 2
	m.bpm = 120
	m.playing = true

	m = elapseMeasures(m, 5) // playing for a while with training off

	m = press(m, "t") // now enable training

	m = elapseMeasures(m, 1)
	assertBPM(t, m, 120) // 1 of 2 measures since enabling: not yet

	m = elapseMeasuresThenLand(m, 1)
	assertBPM(t, m, 130) // full 2 measures since enabling: now it steps
}

// Toggling training off mid-interval and back on must restart the count
// from zero, not resume the stale partial count.
// @ft:62
func TestTogglingOffAndOnResetsTheInterval(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 2
	m.bpm = 120
	m.playing = true

	m = elapseMeasures(m, 1) // 1 of 2 measures into the interval

	m = press(m, "t") // off
	m = press(m, "t") // back on: should restart the interval

	m = elapseMeasures(m, 1)
	assertBPM(t, m, 120) // 1 of 2 measures since re-enabling: not yet

	m = elapseMeasuresThenLand(m, 1)
	assertBPM(t, m, 130) // full 2 measures since re-enabling: now it steps
}

// @ft:19
func TestDisableTempoTraining(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true

	m = press(m, "t")

	assertHeader(t, m, "Tempo Training: off")
	assertTableShown(t, m, false)
}

// @ft:20
func TestTempoTrainingIncreasesBPMAfterConfiguredMeasures(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.bpm = 120
	m.playing = true

	m = elapseMeasures(m, 8)
	assertBPM(t, m, 120) // step landed on beat 4, but hasn't applied until beat 1

	m = elapseBeats(m, 1) // beat 1 of the next measure: step applies now
	assertBPM(t, m, 130)
}

// @ft:22
func TestTempoTrainingHoldsOnceTargetReached(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.targetBPM = 130
	m.bpm = 120
	m.playing = true

	m = elapseMeasuresThenLand(m, 8)
	assertBPM(t, m, 130)

	m = elapseMeasures(m, 8)
	assertBPM(t, m, 130)
	assertHeader(t, m, "Tempo Training: target reached (130 bpm)")

	if m.playingStatus() != "PLAYING" {
		t.Errorf("expected status bar to still show PLAYING, got %q", m.playingStatus())
	}
}

// @ft:17
func TestDefaultTempoTrainingHeaderTableHidden(t *testing.T) {
	m := New()

	assertHeader(t, m, "Tempo Training: off")
	assertTableShown(t, m, false)
}

// @ft:53
func TestTempoTrainingTableShowsDefaultValuesOnceEnabled(t *testing.T) {
	m := New()

	m = press(m, "t")

	assertTableShown(t, m, true)
	assertRow(t, m, "Step", "10 bpm")
	assertRow(t, m, "Interval", "8 measures")
	assertRow(t, m, "Target", "180 bpm")
	assertRow(t, m, "Start", "120 bpm")
}

// @ft:18
func TestIncreaseTempoTrainingStepSize(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 2

	m = press(m, "]")

	assertRow(t, m, "Step", "3 bpm")
}

// @ft:26
func TestDecreaseTempoTrainingStepSize(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 2

	m = press(m, "[")

	assertRow(t, m, "Step", "1 bpm")
}

// @ft:27
func TestTempoTrainingStepSizeCannotGoBelowMinimum(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = minStepBPM

	m = press(m, "[")

	assertRow(t, m, "Step", "1 bpm")
}

// @ft:21
func TestTempoTrainingStepSizeCannotGoAboveMaximum(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = maxStepBPM

	m = press(m, "]")

	assertRow(t, m, "Step", "20 bpm")
}

// @ft:29
func TestAdjustingStepSizeWorksWhileStopped(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.playing = false
	m.stepBPM = 2

	m = press(m, "]")

	assertRow(t, m, "Step", "3 bpm")
}

// @ft:23
func TestChangedStepSizeUsedOnNextIncrease(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.bpm = 120
	m.playing = true

	m = press(m, "]")
	m = elapseMeasuresThenLand(m, 8)

	assertBPM(t, m, 131)
}

// @ft:24
func TestIncreaseTempoTrainingInterval(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepIntervalMeasures = 8

	m = press(m, "}")

	assertRow(t, m, "Interval", "9 measures")
}

// @ft:25
func TestDecreaseTempoTrainingInterval(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepIntervalMeasures = 8

	m = press(m, "{")

	assertRow(t, m, "Interval", "7 measures")
}

// @ft:33
func TestTempoTrainingIntervalCannotGoBelowMinimum(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepIntervalMeasures = minInterval

	m = press(m, "{")

	assertRow(t, m, "Interval", "1 measure")
}

// @ft:34
func TestTempoTrainingIntervalCannotGoAboveMaximum(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepIntervalMeasures = maxInterval

	m = press(m, "}")

	assertRow(t, m, "Interval", "32 measures")
}

// @ft:28
func TestAdjustingIntervalWorksWhileStopped(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.playing = false
	m.stepIntervalMeasures = 8

	m = press(m, "}")

	assertRow(t, m, "Interval", "9 measures")
}

// @ft:36
func TestChangedIntervalUsedForNextIncrease(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.bpm = 120
	m.playing = true

	m = press(m, "}")
	m = elapseMeasures(m, 8)
	assertBPM(t, m, 120)

	m = elapseMeasuresThenLand(m, 1)
	assertBPM(t, m, 130)
}

// @ft:30
func TestIncreaseTempoTrainingTarget(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.targetBPM = 180

	m = press(m, "m")

	assertRow(t, m, "Target", "181 bpm")
}

// @ft:31
func TestDecreaseTempoTrainingTarget(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.targetBPM = 180

	m = press(m, "n")

	assertRow(t, m, "Target", "179 bpm")
}

// @ft:32
func TestTempoTrainingTargetCannotGoBelowMinimum(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.targetBPM = minBPM

	m = press(m, "n")

	assertRow(t, m, "Target", "20 bpm")
}

// @ft:40
func TestTempoTrainingTargetCannotGoAboveMaximum(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.targetBPM = maxBPM

	m = press(m, "m")

	assertRow(t, m, "Target", "300 bpm")
}

// @ft:41
func TestAdjustingTargetWorksWhileStopped(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.playing = false
	m.targetBPM = 180

	m = press(m, "m")

	assertRow(t, m, "Target", "181 bpm")
}

// @ft:45
func TestIncreaseTempoTrainingTargetByLargeIncrement(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.targetBPM = 180

	m = press(m, "shift+m")

	assertRow(t, m, "Target", "190 bpm")
}

// @ft:46
func TestDecreaseTempoTrainingTargetByLargeIncrement(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.targetBPM = 180

	m = press(m, "shift+n")

	assertRow(t, m, "Target", "170 bpm")
}

// @ft:47
func TestTempoTrainingTargetCannotGoBelowMinimumWithLargeIncrement(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.targetBPM = minBPM

	m = press(m, "shift+n")

	assertRow(t, m, "Target", "20 bpm")
}

// @ft:48
func TestTempoTrainingTargetCannotGoAboveMaximumWithLargeIncrement(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.targetBPM = maxBPM

	m = press(m, "shift+m")

	assertRow(t, m, "Target", "300 bpm")
}

// @ft:35
func TestTempoTrainingStepsDownwardWhenTargetBelowCurrentBPM(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.targetBPM = 100
	m.bpm = 130
	m.playing = true

	m = elapseMeasuresThenLand(m, 8)

	assertBPM(t, m, 120)
}

// @ft:69
func TestTempoTrainingHoldsOnceSteppedDownToTarget(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.targetBPM = 110
	m.bpm = 120
	m.playing = true

	m = elapseMeasuresThenLand(m, 8)
	assertBPM(t, m, 110)

	m = elapseMeasures(m, 8)
	assertBPM(t, m, 110)
	assertHeader(t, m, "Tempo Training: target reached (110 bpm)")
}

// @ft:37
func TestTempoTrainingDoesNotOvershootTargetSteppingUp(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.targetBPM = 125
	m.bpm = 120
	m.playing = true

	m = elapseMeasuresThenLand(m, 8)

	assertBPM(t, m, 125)
}

// @ft:38
func TestTempoTrainingDoesNotOvershootTargetSteppingDown(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.targetBPM = 115
	m.bpm = 120
	m.playing = true

	m = elapseMeasuresThenLand(m, 8)

	assertBPM(t, m, 115)
}

// @ft:39
func TestTempoTrainingHoldsWhenBPMAlreadyEqualsTarget(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.targetBPM = 120
	m.bpm = 120
	m.playing = true

	m = elapseMeasures(m, 8)

	assertBPM(t, m, 120)
	assertHeader(t, m, "Tempo Training: target reached (120 bpm)")
}

// @ft:42
func TestDefaultStartRowMatchesDefaultBPMWhileStopped(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true

	assertRow(t, m, "Start", "120 bpm")
}

// @ft:43
func TestStartRowMirrorsManualBPMChangesWhileStopped(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.bpm = 120
	m.playing = false

	m = press(m, "up")

	assertRow(t, m, "Start", "121 bpm")
}

// @ft:44
func TestStartRowCapturedAndHeldFixedOncePlaying(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.bpm = 120
	m.playing = false
	assertRow(t, m, "Start", "120 bpm")

	m = press(m, "space")

	assertRow(t, m, "Start", "120 bpm")
}

// @ft:54
func TestStartRowStaysFixedWhileTempoTrainingSteps(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.bpm = 120
	m.playing = true

	m = elapseMeasuresThenLand(m, 8)

	assertBPM(t, m, 130)
	assertRow(t, m, "Start", "120 bpm")
}

// @ft:55
func TestStartRowRecapturedAfterStoppingAndRestarting(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.playing = true
	m.bpm = 130
	m.startBPM = 130 // this run's start was already captured at 130 when it began playing
	assertRow(t, m, "Start", "130 bpm")

	m = press(m, "space")
	m = press(m, "shift+up")
	m = press(m, "shift+up")
	m = press(m, "space")

	assertRow(t, m, "Start", "150 bpm")
}

// @ft:58
func TestBPMRevertsToStartWhenMetronomeStops(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.bpm = 120
	m.playing = true

	m = elapseMeasuresThenLand(m, 8)
	assertBPM(t, m, 130)

	m = press(m, "space")

	assertBPM(t, m, 120)
	if m.playingStatus() != "STOPPED" {
		t.Errorf("expected status bar to show STOPPED, got %q", m.playingStatus())
	}
}

// @ft:59
func TestStartDoesNotRevertToDriftedBPMWhenMetronomeStops(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.bpm = 120
	m.playing = true

	m = elapseMeasuresThenLand(m, 8)
	assertBPM(t, m, 130) // drift landed
	m = press(m, "space")

	assertRow(t, m, "Start", "120 bpm")
}

// A tempo-training step lands on beat 4 (the last beat of the measure, the
// natural "measure complete" signal), but must not take effect until beat 1
// of the next measure: neither the BPM readout nor the tick interval that
// follows beat 4 should change yet. Only once beat 1 lands should the BPM
// readout update and the following tick be paced at the new tempo. Without
// this, the tempo audibly/visually changes a beat early.
// @ft:60
func TestTempoChangeTakesEffectStartingAtBeatOneOfNextMeasure(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.bpm = 120
	m.playing = true

	m = elapseMeasures(m, 7) // 7 full measures: no step yet
	m = elapseBeats(m, 3)    // beats 1-3 of the 8th measure

	// Beat 4 of the 8th measure: the step lands here, but isn't applied yet.
	// (beat=0: self-advancing, the same fallback-path sentinel
	// nextBeatCmd/tickCmd use when no audio engine is driving beats — see
	// model.go's beatMsg doc comment.)
	m = m.advanceBeat(0)
	if m.bpm != 120 {
		t.Fatalf("expected bpm to still be paced at 120 immediately after beat 4, got %d", m.bpm)
	}
	assertBPM(t, m, 120) // BPM readout unchanged until beat 1

	// Beat 1 of the next measure: the pending step is applied now.
	m = m.advanceBeat(0)
	if m.bpm != 130 {
		t.Fatalf("expected bpm to be paced at the NEW bpm 130 after beat 1 of the next measure, got %d", m.bpm)
	}
	assertBPM(t, m, 130)
}

// @ft:70
func TestTempoTrainingTableHiddenByDefault(t *testing.T) {
	m := New()

	assertTableShown(t, m, false)
}

// @ft:71
func TestTempoTrainingTableAppearsWhenTurnedOn(t *testing.T) {
	m := New()

	m = press(m, "t")

	assertTableShown(t, m, true)
}

// @ft:56
func TestTempoTrainingTableDisappearsWhenTurnedOff(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true

	m = press(m, "t")

	assertTableShown(t, m, false)
}

// @ft:57
func TestTempoTrainingHeaderAlwaysShownEvenWhileOff(t *testing.T) {
	m := New()

	assertHeader(t, m, "Tempo Training: off")
}

// @ft:63
func TestMeasureCounterHiddenWhenTempoTrainingOff(t *testing.T) {
	m := New()

	assertStatusBar(t, m, "◥ mn ◥◥ STOPPED ◥◥ ♩ 120 bpm ◥◤ tempo training (t) ◤")
}

// @ft:64
func TestMeasureCounterAppearsOnceTempoTrainingOn(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.playing = true

	assertStatusBar(t, m, "◥ mn ◥◥ PLAYING ◥◥ ♩ 120 bpm ◥◤ 1/8 ◤")
}

// @ft:65
func TestMeasureCounterIncrementsAsMeasuresComplete(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.playing = true

	m = elapseMeasures(m, 3)

	assertStatusBar(t, m, "◥ mn ◥◥ PLAYING ◥◥ ♩ 120 bpm ◥◤ 3/8 ◤")
}

// @ft:66
func TestMeasureCounterHoldsAtIntervalBetweenBeatFourAndBeatOne(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.playing = true

	m = elapseMeasures(m, 8)

	assertStatusBar(t, m, "◥ mn ◥◥ PLAYING ◥◥ ♩ 120 bpm ◥◤ 8/8 ◤")
}

// @ft:72
func TestMeasureCounterDoesNotAdvanceUntilBeatOne(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.playing = true

	m = elapseMeasures(m, 1)
	assertStatusBar(t, m, "◥ mn ◥◥ PLAYING ◥◥ ♩ 120 bpm ◥◤ 1/8 ◤")

	m = elapseBeats(m, 1)
	assertStatusBar(t, m, "◥ mn ◥◥ PLAYING ◥◥ ♩ 120 bpm ◥◤ 2/8 ◤")
}

// @ft:67
func TestMeasureCounterResetsOncePendingStepLands(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.playing = true

	m = elapseMeasuresThenLand(m, 8)

	assertStatusBar(t, m, "◥ mn ◥◥ PLAYING ◥◥ ♩ 130 bpm ◥◤ 1/8 ◤")
}

// @ft:68
func TestMeasureCounterResetsOnStop(t *testing.T) {
	m := New()
	m.tempoTrainingOn = true
	m.stepBPM = 10
	m.stepIntervalMeasures = 8
	m.playing = true

	m = elapseMeasures(m, 3)
	m = press(m, "space")

	assertStatusBar(t, m, "◥ mn ◥◥ STOPPED ◥◥ ♩ 120 bpm ◥◤ 1/8 ◤")
}
