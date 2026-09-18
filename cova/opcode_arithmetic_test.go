package cova

import (
	"math"
	"testing"
)

var opcodeNumericKinds = []ValueKind{
	KindBool,
	KindByte,
	KindInt8,
	KindInt16,
	KindInt32,
	KindInt64,
	KindUint8,
	KindUint16,
	KindUint32,
	KindUint64,
	KindFloat32,
	KindFloat64,
}

func opcodeValueBits(kind ValueKind, value int64) uint64 {
	switch kind {
	case KindFloat32:
		return uint64(math.Float32bits(float32(value)))
	case KindFloat64:
		return math.Float64bits(float64(value))
	case KindBool, KindByte, KindInt8, KindUint8:
		return uint64(uint8(value))
	case KindInt16, KindUint16:
		return uint64(uint16(value))
	case KindInt32, KindUint32:
		return uint64(uint32(value))
	default:
		return uint64(value)
	}
}

func appendOpcodeValue(code *CodeMemory, kind ValueKind, bits uint64) {
	if kind == KindAddress {
		address := Address(uint32(bits))
		code.AppendInstruction(makeAddrInstruction(address.Segment()))
		code.AppendUint32(address.Index())
		return
	}
	code.AppendInstruction(makeInstruction(OpPush, kind, ModeNone, FlagNone))
	code.AppendImmediate(kind, bits)
}

func runOpcodeProgram(t *testing.T, code CodeMemory, returnKind ValueKind) (*VM, VMStatus) {
	t.Helper()
	program := &LinkedProgram{
		Text:          code,
		EntryPoint:    0,
		BSSByteSize:   32,
		DataByteSize:  32,
		DataData:      make([]byte, 32),
		Functions:     []ScriptFunctionDescriptor{{BodyAddress: 0, ReturnKind: returnKind, FrameByteSize: 16}},
		FrameByteSize: 16,
	}
	vm := NewVMWithConfig(VMConfig{FrameCapacity: 32, StackCapacity: 128, CallFrameCapacity: 8})
	return vm, vm.Run(program, 0)
}

func runOpcodeResult(t *testing.T, code CodeMemory, returnKind ValueKind) uint64 {
	t.Helper()
	code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
	vm, status := runOpcodeProgram(t, code, returnKind)
	if status != VMStatusOK {
		t.Fatalf("Run status = %s, want ok", status)
	}
	bits, status := vm.PopBits(returnKind)
	if status != VMStatusOK {
		t.Fatalf("PopBits status = %s, want ok", status)
	}
	if got := len(vm.memory.segment[segmentStack]); got != 0 {
		t.Fatalf("stack retains %d bytes", got)
	}
	return bits
}

func TestOpcodePushAllImmediateKinds(t *testing.T) {
	if len(opcodeNumericKinds) != 12 {
		t.Fatalf("numeric kind matrix has %d entries, want 12", len(opcodeNumericKinds))
	}
	for _, kind := range opcodeNumericKinds {
		t.Run(valueKindTestName(kind), func(t *testing.T) {
			want := opcodeValueBits(kind, 7)
			var code CodeMemory
			appendOpcodeValue(&code, kind, want)
			if got := runOpcodeResult(t, code, kind); got != want {
				t.Fatalf("result bits = %#x, want %#x", got, want)
			}
		})
	}
}

func TestOpcodePushRejectsInvalidKinds(t *testing.T) {
	for _, kind := range []ValueKind{KindNone, KindVoid, KindAddress} {
		t.Run(valueKindTestName(kind), func(t *testing.T) {
			var code CodeMemory
			code.AppendInstruction(makeInstruction(OpPush, kind, ModeNone, FlagNone))
			code.AppendImmediate(kind, 1)
			_, status := runOpcodeProgram(t, code, KindVoid)
			if status != VMStatusInvalidValueKind {
				t.Fatalf("Run status = %s, want invalid value kind", status)
			}
		})
	}
}

func TestOpcodeArithmeticAllOperationsAndKinds(t *testing.T) {
	operations := []struct {
		op   ArithmeticOp
		want int64
	}{
		{ArithmeticAdd, 15},
		{ArithmeticSub, 9},
		{ArithmeticMul, 36},
		{ArithmeticDiv, 4},
	}
	if got := len(operations) * len(opcodeNumericKinds); got != 48 {
		t.Fatalf("arithmetic matrix has %d entries, want 48", got)
	}
	for _, kind := range opcodeNumericKinds {
		for _, operation := range operations {
			name := valueKindTestName(kind) + "/" + arithmeticOpTestName(operation.op)
			t.Run(name, func(t *testing.T) {
				var code CodeMemory
				appendOpcodeValue(&code, kind, opcodeValueBits(kind, 12))
				appendOpcodeValue(&code, kind, opcodeValueBits(kind, 3))
				code.AppendInstruction(makeArithmeticInstruction(kind, operation.op))
				want := opcodeValueBits(kind, operation.want)
				if got := runOpcodeResult(t, code, kind); got != want {
					t.Fatalf("result bits = %#x, want %#x", got, want)
				}
			})
		}
	}
}

func TestOpcodeArithmeticDivisionByZeroAllKinds(t *testing.T) {
	for _, kind := range opcodeNumericKinds {
		t.Run(valueKindTestName(kind), func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, kind, opcodeValueBits(kind, 1))
			appendOpcodeValue(&code, kind, opcodeValueBits(kind, 0))
			code.AppendInstruction(makeArithmeticInstruction(kind, ArithmeticDiv))
			_, status := runOpcodeProgram(t, code, kind)
			if status != VMStatusDivisionByZero {
				t.Fatalf("Run status = %s, want division by zero", status)
			}
		})
	}
}

func TestOpcodeArithmeticRejectsInvalidOperationAndKind(t *testing.T) {
	tests := []struct {
		name string
		kind ValueKind
		op   ArithmeticOp
	}{
		{"operation", KindInt32, ArithmeticInvalid},
		{"kind", KindAddress, ArithmeticAdd},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindInt32, 2)
			appendOpcodeValue(&code, KindInt32, 1)
			code.AppendInstruction(makeArithmeticInstruction(test.kind, test.op))
			_, status := runOpcodeProgram(t, code, test.kind)
			if status != VMStatusInvalidOpcode {
				t.Fatalf("Run status = %s, want invalid opcode", status)
			}
		})
	}
}

func TestOpcodeArithmeticSignedMinimumDividedByNegativeOneWraps(t *testing.T) {
	tests := []struct {
		kind ValueKind
		bits uint64
	}{
		{KindInt8, 0x80},
		{KindInt16, 0x8000},
		{KindInt32, 0x80000000},
		{KindInt64, 0x8000000000000000},
	}
	for _, test := range tests {
		t.Run(valueKindTestName(test.kind), func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, test.kind, test.bits)
			appendOpcodeValue(&code, test.kind, opcodeValueBits(test.kind, -1))
			code.AppendInstruction(makeArithmeticInstruction(test.kind, ArithmeticDiv))
			if got := runOpcodeResult(t, code, test.kind); got != test.bits {
				t.Fatalf("result bits = %#x, want wrapped minimum %#x", got, test.bits)
			}
		})
	}
}

func TestOpcodeCompareAllValidOperationsAndKinds(t *testing.T) {
	operations := []CompareOp{CompareEqual, CompareNotEqual, CompareLess, CompareLessEqual, CompareGreater, CompareGreaterEqual}
	kinds := append(append([]ValueKind(nil), opcodeNumericKinds...), KindAddress)
	caseCount := 0
	for _, kind := range kinds {
		for _, operation := range operations {
			if kind == KindBool && operation != CompareEqual && operation != CompareNotEqual {
				continue
			}
			caseCount++
			name := valueKindTestName(kind) + "/" + compareOpTestName(operation)
			t.Run(name, func(t *testing.T) {
				leftBits := opcodeValueBits(kind, 2)
				rightBits := opcodeValueBits(kind, 3)
				if kind == KindBool {
					leftBits = 0
					rightBits = 1
				} else if kind == KindAddress {
					leftBits = uint64(makeAddress(segmentData, 2))
					rightBits = uint64(makeAddress(segmentData, 3))
				}
				var code CodeMemory
				appendOpcodeValue(&code, kind, leftBits)
				appendOpcodeValue(&code, kind, rightBits)
				code.AppendInstruction(makeCompareInstruction(kind, operation))
				want := operation == CompareNotEqual || operation == CompareLess || operation == CompareLessEqual
				got := runOpcodeResult(t, code, KindBool)
				if (got != 0) != want || got > 1 {
					t.Fatalf("comparison result = %d, want %t canonical bool", got, want)
				}
			})
		}
	}
	if caseCount != 74 {
		t.Fatalf("comparison matrix has %d entries, want 74", caseCount)
	}
}

func TestOpcodeCompareRejectsOrderedBoolAndInvalidInputs(t *testing.T) {
	for _, operation := range []CompareOp{CompareLess, CompareLessEqual, CompareGreater, CompareGreaterEqual} {
		t.Run(compareOpTestName(operation), func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindBool, 0)
			appendOpcodeValue(&code, KindBool, 1)
			code.AppendInstruction(makeCompareInstruction(KindBool, operation))
			_, status := runOpcodeProgram(t, code, KindBool)
			if status != VMStatusInvalidOpcode {
				t.Fatalf("Run status = %s, want invalid opcode", status)
			}
		})
	}

	var code CodeMemory
	appendOpcodeValue(&code, KindInt32, 1)
	appendOpcodeValue(&code, KindInt32, 1)
	code.AppendInstruction(makeCompareInstruction(KindInt32, CompareInvalid))
	if _, status := runOpcodeProgram(t, code, KindBool); status != VMStatusInvalidOpcode {
		t.Fatalf("invalid operation status = %s, want invalid opcode", status)
	}

	code = nil
	code.AppendInstruction(makeCompareInstruction(KindNone, CompareEqual))
	if _, status := runOpcodeProgram(t, code, KindBool); status != VMStatusInvalidValueKind {
		t.Fatalf("invalid kind status = %s, want invalid value kind", status)
	}
}

func TestOpcodeCompareFloatNaN(t *testing.T) {
	for _, kind := range []ValueKind{KindFloat32, KindFloat64} {
		t.Run(valueKindTestName(kind), func(t *testing.T) {
			bits := math.Float64bits(math.NaN())
			if kind == KindFloat32 {
				bits = uint64(math.Float32bits(float32(math.NaN())))
			}
			var code CodeMemory
			appendOpcodeValue(&code, kind, bits)
			appendOpcodeValue(&code, kind, bits)
			code.AppendInstruction(makeCompareInstruction(kind, CompareEqual))
			if got := runOpcodeResult(t, code, KindBool); got != 0 {
				t.Fatalf("NaN equality result = %d, want false", got)
			}
		})
	}
}

func valueKindTestName(kind ValueKind) string {
	return "kind_" + itoaByte(byte(kind))
}

func arithmeticOpTestName(op ArithmeticOp) string {
	return "arithmetic_" + itoaByte(byte(op))
}

func compareOpTestName(op CompareOp) string {
	return "compare_" + itoaByte(byte(op))
}

func itoaByte(value byte) string {
	if value < 10 {
		return string(rune('0' + value))
	}
	return string([]byte{'0' + value/10, '0' + value%10})
}
