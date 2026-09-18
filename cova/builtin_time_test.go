package cova

import (
	"math"
	"testing"
)

func TestTimeTickAcrossRuns(t *testing.T) {
	linked := mustLinkProgram(t, `
float32 script_main() {
	return time::tickTime();
}
`, 0, 0)
	vm := NewVM(testFrameCapacityBytes)

	if status := vm.Run(linked, 1000); status != VMStatusOK {
		t.Fatalf("first Run failed: %v", status)
	}
	if got, status := vm.PopFloat32(); status != VMStatusOK || got != 0 {
		t.Fatalf("first tickTime = %v, want 0 (status %v)", got, status)
	}

	if status := vm.RunLoaded(1250); status != VMStatusOK {
		t.Fatalf("second RunLoaded failed: %v", status)
	}
	if got, status := vm.PopFloat32(); status != VMStatusOK || got != 0.25 {
		t.Fatalf("second tickTime = %v, want 0.25 (status %v)", got, status)
	}

	if status := vm.RunLoaded(1250); status != VMStatusOK {
		t.Fatalf("equal-time RunLoaded failed: %v", status)
	}
	if got, status := vm.PopFloat32(); status != VMStatusOK || got != 0 {
		t.Fatalf("equal-time tickTime = %v, want 0 (status %v)", got, status)
	}

	if status := vm.RunLoaded(1249); status != VMStatusInvalidParameter {
		t.Fatalf("backwards RunLoaded status = %v, want %v", status, VMStatusInvalidParameter)
	}
}

func TestTimerLifecycleAcrossRuns(t *testing.T) {
	start := mustLinkProgram(t, `
void script_main() {
	time::timerStart(0, 100);
}
`, 0, 0)
	stop := mustLinkProgram(t, `
void script_main() {
	time::timerStop(0);
}
`, 0, 0)
	reset := mustLinkProgram(t, `
void script_main() {
	time::timerReset(0);
}
`, 0, 0)
	done := mustLinkProgram(t, `
bool script_main() {
	return time::timerDone(0);
}
`, 0, 0)
	vm := NewVM(testFrameCapacityBytes)

	if status := vm.Run(start, 1000); status != VMStatusOK {
		t.Fatalf("timerStart failed: %v", status)
	}
	assertTimerDone(t, vm, done, 1100, false)
	assertTimerDone(t, vm, done, 1101, true)
	assertTimerDone(t, vm, done, 1200, true)

	if status := vm.Run(stop, 1200); status != VMStatusOK {
		t.Fatalf("timerStop failed: %v", status)
	}
	assertTimerDone(t, vm, done, 1300, false)

	if status := vm.Run(reset, 1300); status != VMStatusOK {
		t.Fatalf("timerReset failed: %v", status)
	}
	assertTimerDone(t, vm, done, 1400, false)
	assertTimerDone(t, vm, done, 1401, true)

	if status := vm.Run(start, 1500); status != VMStatusOK {
		t.Fatalf("timer restart failed: %v", status)
	}
	assertTimerDone(t, vm, done, 1600, false)
}

func TestTimerSlotsAreIndependent(t *testing.T) {
	startZero := mustLinkProgram(t, `
void script_main() {
	time::timerStart(0, 0);
}
`, 0, 0)
	startOne := mustLinkProgram(t, `
void script_main() {
	time::timerStart(1, 100);
}
`, 0, 0)
	doneZero := mustLinkProgram(t, `
bool script_main() {
	return time::timerDone(0);
}
`, 0, 0)
	doneOne := mustLinkProgram(t, `
bool script_main() {
	return time::timerDone(1);
}
`, 0, 0)
	vm := NewVM(testFrameCapacityBytes)

	if status := vm.Run(startZero, 10); status != VMStatusOK {
		t.Fatalf("zero-duration timerStart failed: %v", status)
	}
	if status := vm.Run(startOne, 10); status != VMStatusOK {
		t.Fatalf("second timerStart failed: %v", status)
	}
	assertTimerDone(t, vm, doneZero, 10, false)
	assertTimerDone(t, vm, doneZero, 11, true)
	assertTimerDone(t, vm, doneOne, 11, false)
}

func TestTimerCapacityAndDeadlineSaturation(t *testing.T) {
	if got := len(NewVM(testFrameCapacityBytes).timers); got != int(defaultTimerCapacity) {
		t.Fatalf("default timer capacity = %d, want %d", got, defaultTimerCapacity)
	}

	config := VMConfig{
		FrameCapacity:     testFrameCapacityBytes,
		StackCapacity:     256,
		CallFrameCapacity: 8,
		TimerCapacity:     1,
	}
	vm := NewVMWithConfig(config)
	invalid := mustLinkProgram(t, `
bool script_main() {
	return time::timerDone(1);
}
`, 0, 0)
	if status := vm.Run(invalid, 0); status != VMStatusInvalidParameter {
		t.Fatalf("out-of-range timer status = %v, want %v", status, VMStatusInvalidParameter)
	}

	start := mustLinkProgram(t, `
void script_main() {
	time::timerStart(0, 20);
}
`, 0, 0)
	if status := vm.Run(start, math.MaxUint64-10); status != VMStatusOK {
		t.Fatalf("overflowing timerStart failed: %v", status)
	}
	if got := vm.timers[0].endTimeMS; got != math.MaxUint64 {
		t.Fatalf("saturated deadline = %d, want %d", got, uint64(math.MaxUint64))
	}
}

func TestTimeBuiltInsRejectWrongEncodedKinds(t *testing.T) {
	vm := NewVM(testFrameCapacityBytes)
	if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInTickTime, KindBool)); status != VMStatusInvalidValueKind {
		t.Fatalf("tickTime wrong-kind status = %v, want %v", status, VMStatusInvalidValueKind)
	}
	if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInTimerStart, KindBool)); status != VMStatusInvalidValueKind {
		t.Fatalf("timerStart wrong-kind status = %v, want %v", status, VMStatusInvalidValueKind)
	}
	if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInTimerDone, KindVoid)); status != VMStatusInvalidValueKind {
		t.Fatalf("timerDone wrong-kind status = %v, want %v", status, VMStatusInvalidValueKind)
	}
}

func assertTimerDone(t *testing.T, vm *VM, program *LinkedProgram, nowMS uint64, want bool) {
	t.Helper()
	if status := vm.Run(program, nowMS); status != VMStatusOK {
		t.Fatalf("timerDone at %d failed: %v", nowMS, status)
	}
	if got, status := vm.PopBool(); status != VMStatusOK || got != want {
		t.Fatalf("timerDone at %d = %v, want %v (status %v)", nowMS, got, want, status)
	}
}
