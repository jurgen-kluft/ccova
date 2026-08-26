package cova

import (
	"math"
	"strings"
	"testing"
)

func TestCompileAndRunMathBuiltInsFloat64(t *testing.T) {
	script := `
float64 script_main() {
	return math::abs(-2) + math::sin(0) + math::cos(0) + math::tan(0) + math::asin(0) + math::acos(1) + math::atan(0) + math::pow(2, 3) + math::sqrt(9);
}
`
	linked := mustLinkProgram(t, script, 0, 0)
	vm := NewVM(testFrameCapacityBytes)
	if status := vm.Run(linked); status != VMStatusOK {
		t.Fatalf("Run failed: %v", status)
	}
	if got, status := vm.PopFloat64(); status != VMStatusOK || got != 14 {
		t.Fatalf("math built-ins returned %v, want 14 (status %v)", got, status)
	}
}

func TestCompileAndRunMathBuiltInsFloat32(t *testing.T) {
	script := `
float32 script_main() {
	return math::sin(0.0f) + math::sqrt(9.0f) + math::pow(2.0f, 3.0f);
}
`
	linked := mustLinkProgram(t, script, 0, 0)
	vm := NewVM(testFrameCapacityBytes)
	if status := vm.Run(linked); status != VMStatusOK {
		t.Fatalf("Run failed: %v", status)
	}
	if got, status := vm.PopFloat32(); status != VMStatusOK || got != 11 {
		t.Fatalf("float32 math built-ins returned %v, want 11 (status %v)", got, status)
	}
}

func TestBuiltInAbsPreservesIntegerKindAndWrapsMinimum(t *testing.T) {
	var code CodeMemory
	appendOpcodeValue(&code, KindInt32, uint64(uint32(0x80000000)))
	code.AppendInstruction(makeBuiltInInstruction(makeBuiltInFunction(BuiltInAbs, KindInt32)))
	if got := runOpcodeResult(t, code, KindInt32); got != 0x80000000 {
		t.Fatalf("math::abs(int32 minimum) bits = %#x, want %#x", got, uint64(0x80000000))
	}

	var floatCode CodeMemory
	appendOpcodeValue(&floatCode, KindFloat64, math.Float64bits(-3.5))
	floatCode.AppendInstruction(makeBuiltInInstruction(makeBuiltInFunction(BuiltInAbs, KindFloat64)))
	if got := math.Float64frombits(runOpcodeResult(t, floatCode, KindFloat64)); got != 3.5 {
		t.Fatalf("math::abs(-3.5) = %v, want 3.5", got)
	}
}

func TestMathBuiltInsSupportUnsignedAbsAndTypedMinMax(t *testing.T) {
	for _, test := range []struct {
		name   string
		script string
		kind   ValueKind
		want   uint64
	}{
		{"unsigned_abs", `uint32 script_main() { uint32 value = 4000000000; return math::abs(value); }`, KindUint32, 4000000000},
		{"narrow_min", `int16 script_main() { int16 left = -7; int16 right = 3; return math::min(left, right); }`, KindInt16, uint64(uint16(0xfff9))},
		{"promoted_max", `float64 script_main() { int32 left = 7; float64 right = 9.5; return math::max(left, right); }`, KindFloat64, math.Float64bits(9.5)},
	} {
		t.Run(test.name, func(t *testing.T) {
			linked := mustLinkProgram(t, test.script, 0, 0)
			vm := NewVM(testFrameCapacityBytes)
			if status := vm.Run(linked); status != VMStatusOK {
				t.Fatalf("Run failed: %v", status)
			}
			if got, status := vm.PopBits(test.kind); status != VMStatusOK || got != test.want {
				t.Fatalf("result bits = %#x, want %#x (status %v)", got, test.want, status)
			}
		})
	}
}

func TestBuiltInMinMaxValueKinds(t *testing.T) {
	tests := []struct {
		name    string
		kind    ValueKind
		left    uint64
		right   uint64
		wantMin uint64
		wantMax uint64
	}{
		{"uint32", KindUint32, 4000000000, 17, 17, 4000000000},
		{"int32", KindInt32, uint64(uint32(0xfffffff9)), 3, uint64(uint32(0xfffffff9)), 3},
		{"uint64", KindUint64, 1 << 63, 17, 17, 1 << 63},
		{"int64", KindInt64, uint64(0xfffffffffffffff9), 3, uint64(0xfffffffffffffff9), 3},
		{"float32", KindFloat32, uint64(math.Float32bits(-2.5)), uint64(math.Float32bits(9.5)), uint64(math.Float32bits(-2.5)), uint64(math.Float32bits(9.5))},
		{"float64", KindFloat64, math.Float64bits(-2.5), math.Float64bits(9.5), math.Float64bits(-2.5), math.Float64bits(9.5)},
	}
	for _, test := range tests {
		for _, operation := range []struct {
			name string
			op   BuiltInOperation
			want uint64
		}{
			{"min", BuiltInMin, test.wantMin},
			{"max", BuiltInMax, test.wantMax},
		} {
			t.Run(test.name+"_"+operation.name, func(t *testing.T) {
				vm := NewVM(testFrameCapacityBytes)
				if status := vm.PushBits(test.kind, test.left); status != VMStatusOK {
					t.Fatalf("push left failed: %v", status)
				}
				if status := vm.PushBits(test.kind, test.right); status != VMStatusOK {
					t.Fatalf("push right failed: %v", status)
				}
				if status := vm.executeBuiltIn(makeBuiltInFunction(operation.op, test.kind)); status != VMStatusOK {
					t.Fatalf("execute failed: %v", status)
				}
				if got, status := vm.PopBits(test.kind); status != VMStatusOK || got != operation.want {
					t.Fatalf("result bits = %#x, want %#x (status %v)", got, operation.want, status)
				}
			})
		}
	}
}

func TestMathBuiltInDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		script    string
		wantError string
	}{
		{"arity", `float64 script_main() { return math::pow(2); }`, "expects 2 arguments"},
		{"bool", `float64 script_main() { return math::sin(true); }`, "requires numeric arguments"},
		{"unknown", `float64 script_main() { return other::sin(0); }`, "unknown function"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := NewContext()
			tokens := mustTokenize(t, ctx, test.script)
			program := mustParseTokens(t, ctx, tokens)

			_, ok := NewCompiler(ctx).Compile(program)
			if ok || !strings.Contains(ctx.String(), test.wantError) {
				t.Fatalf("Compile error = %v, want substring %q", ctx.String(), test.wantError)
			}
		})
	}
}

func TestCompileAndRunRemainingMathBuiltIns(t *testing.T) {
	script := `
float64 script_main() {
	return math::map(5, 0, 10, 0, 100)
		+ math::clamp(12, 0, 10)
		+ math::smoothStep(0, 10, 5)
		+ math::lerp(10, 20, 0.5)
		+ math::slerp(1, 1, 0.5);
}
`
	linked := mustLinkProgram(t, script, 0, 0)
	vm := NewVM(testFrameCapacityBytes)
	if status := vm.Run(linked); status != VMStatusOK {
		t.Fatalf("Run failed: %v", status)
	}
	if got, status := vm.PopFloat64(); status != VMStatusOK || math.Abs(got-76.5) > 0.000001 {
		t.Fatalf("remaining math built-ins returned %v, want 76.5 (status %v)", got, status)
	}
}

func TestCompileAndRunFixedPointBuiltIns(t *testing.T) {
	script := `
int32 script_main() {
	return math::lerp(0, 100, 128, 8) + math::smoothStep(0, 100, 128, 8);
}
`
	linked := mustLinkProgram(t, script, 0, 0)
	vm := NewVM(testFrameCapacityBytes)
	if status := vm.Run(linked); status != VMStatusOK {
		t.Fatalf("Run failed: %v", status)
	}
	if got, status := vm.PopBits(KindInt32); status != VMStatusOK || int32(got) != 100 {
		t.Fatalf("fixed-point built-ins returned %d, want 100 (status %v)", int32(got), status)
	}
}

func TestBuiltInRandomIsDeterministicAndSeedable(t *testing.T) {
	vm := NewVM(testFrameCapacityBytes)
	vm.SetRandomSeed(12345)
	sequence := [3]uint64{}
	for index := range sequence {
		if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInRandom, KindInt32)); status != VMStatusOK {
			t.Fatalf("random execute failed: %v", status)
		}
		value, status := vm.PopBits(KindInt32)
		if status != VMStatusOK {
			t.Fatalf("random pop failed: %v", status)
		}
		if value > math.MaxInt32 {
			t.Fatalf("random value %d is outside nonnegative int32 range", value)
		}
		sequence[index] = value
	}
	vm.SetRandomSeed(12345)
	for index, want := range sequence {
		if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInRandom, KindInt32)); status != VMStatusOK {
			t.Fatalf("random replay %d failed: %v", index, status)
		}
		got, status := vm.PopBits(KindInt32)
		if status != VMStatusOK || got != want {
			t.Fatalf("random replay %d = %d, want %d (status %v)", index, got, want, status)
		}
	}
}

func TestBuiltInUnsignedAbsPreservesHighBit(t *testing.T) {
	vm := NewVM(testFrameCapacityBytes)
	if status := vm.PushBits(KindUint8, 0x80); status != VMStatusOK {
		t.Fatalf("push failed: %v", status)
	}
	if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInAbs, KindUint8)); status != VMStatusOK {
		t.Fatalf("execute failed: %v", status)
	}
	if got, status := vm.PopBits(KindUint8); status != VMStatusOK || got != 0x80 {
		t.Fatalf("math::abs(uint8(0x80)) = %#x, want 0x80 (status %v)", got, status)
	}
}

func TestBuiltInRangePreconditions(t *testing.T) {
	vm := NewVM(testFrameCapacityBytes)
	for _, value := range []uint64{5, 1, 1, 0, 10} {
		if status := vm.PushBits(KindInt32, value); status != VMStatusOK {
			t.Fatalf("map push failed: %v", status)
		}
	}
	if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInMap, KindInt32)); status != VMStatusDivisionByZero {
		t.Fatalf("map with equal input bounds status = %v, want division by zero", status)
	}

	for _, value := range []uint64{5, 10, 0} {
		if status := vm.PushBits(KindInt32, value); status != VMStatusOK {
			t.Fatalf("clamp push failed: %v", status)
		}
	}
	if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInClamp, KindInt32)); status != VMStatusInvalidParameter {
		t.Fatalf("clamp with reversed bounds status = %v, want invalid parameter", status)
	}
}

func TestBuiltInMapNormalizesAndClamps(t *testing.T) {
	tests := []struct {
		name    string
		value   int32
		inLow   int32
		inHigh  int32
		outLow  int32
		outHigh int32
		want    int32
	}{
		{"below_input", -5, 0, 10, 0, 100, 0},
		{"above_input", 15, 0, 10, 0, 100, 100},
		{"reversed_input", 2, 10, 0, 0, 100, 20},
		{"reversed_output", 2, 0, 10, 100, 0, 20},
		{"both_reversed", 2, 10, 0, 100, 0, 20},
		{"constant_output", 5, 0, 10, 42, 42, 42},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vm := NewVM(testFrameCapacityBytes)
			for _, value := range []int32{test.value, test.inLow, test.inHigh, test.outLow, test.outHigh} {
				if status := vm.PushBits(KindInt32, uint64(uint32(value))); status != VMStatusOK {
					t.Fatalf("push failed: %v", status)
				}
			}
			if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInMap, KindInt32)); status != VMStatusOK {
				t.Fatalf("map failed: %v", status)
			}
			got, status := vm.PopBits(KindInt32)
			if status != VMStatusOK || int32(got) != test.want {
				t.Fatalf("map result = %d, want %d (status %v)", int32(got), test.want, status)
			}
		})
	}
}

func TestBuiltInMapUsesUnsignedAndFloatOrdering(t *testing.T) {
	vm := NewVM(testFrameCapacityBytes)
	for _, value := range []uint64{4200000000, 4000000000, 4100000000, 0, 1} {
		if status := vm.PushBits(KindUint32, value); status != VMStatusOK {
			t.Fatalf("uint32 push failed: %v", status)
		}
	}
	if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInMap, KindUint32)); status != VMStatusOK {
		t.Fatalf("uint32 map failed: %v", status)
	}
	if got, status := vm.PopBits(KindUint32); status != VMStatusOK || got != 1 {
		t.Fatalf("uint32 map result = %d, want 1 (status %v)", got, status)
	}

	for _, value := range []float32{-5, 0, 10, 20, 40} {
		if status := vm.PushFloat32(value); status != VMStatusOK {
			t.Fatalf("float32 push failed: %v", status)
		}
	}
	if status := vm.executeBuiltIn(makeBuiltInFunction(BuiltInMap, KindFloat32)); status != VMStatusOK {
		t.Fatalf("float32 map failed: %v", status)
	}
	if got, status := vm.PopFloat32(); status != VMStatusOK || got != 20 {
		t.Fatalf("float32 map result = %v, want 20 (status %v)", got, status)
	}
}
