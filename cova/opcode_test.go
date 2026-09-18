package cova

import (
	"math"
	"testing"
)

func TestOpcodeCoverageCompleteness(t *testing.T) {
	cases := [OpcodeCount]func(*testing.T){
		OpPush: func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindInt32, 1)
			_ = runOpcodeResult(t, code, KindInt32)
		},
		OpArithmetic: func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindInt32, 2)
			appendOpcodeValue(&code, KindInt32, 3)
			code.AppendInstruction(makeArithmeticInstruction(KindInt32, ArithmeticAdd))
			_ = runOpcodeResult(t, code, KindInt32)
		},
		OpConvert: func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindInt32, 1)
			code.AppendInstruction(makeConvertInstruction(KindInt32, KindUint32))
			_ = runOpcodeResult(t, code, KindUint32)
		},
		OpAddr: func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindAddress, uint64(makeAddress(segmentData, 1)))
			_ = runOpcodeResult(t, code, KindAddress)
		},
		OpOffset: func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindAddress, uint64(makeAddress(segmentData, 1)))
			appendOpcodeValue(&code, KindInt32, 1)
			code.AppendInstruction(makeInstruction(OpOffset, KindNone, ModeNone, FlagNone))
			_ = runOpcodeResult(t, code, KindAddress)
		},
		OpDereference: func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindAddress, uint64(makeAddress(segmentBSS, 0)))
			code.AppendInstruction(makeInstruction(OpDereference, KindInt32, ModeNone, FlagNone))
			_ = runOpcodeResult(t, code, KindInt32)
		},
		OpAssign: func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindInt32, 1)
			appendOpcodeValue(&code, KindAddress, uint64(makeAddress(segmentBSS, 0)))
			code.AppendInstruction(makeInstruction(OpAssign, KindInt32, ModeNone, FlagNone))
			code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
			if _, status := runOpcodeProgram(t, code, KindVoid); status != VMStatusOK {
				t.Fatalf("Run status = %s, want ok", status)
			}
		},
		OpCompare: func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindInt32, 1)
			appendOpcodeValue(&code, KindInt32, 1)
			code.AppendInstruction(makeCompareInstruction(KindInt32, CompareEqual))
			_ = runOpcodeResult(t, code, KindBool)
		},
		OpJumpIfFalse: testOpcodeCoverageJumpIfFalse,
		OpJump:        testOpcodeCoverageJump,
		OpCall:        testOpcodeCoverageCall,
		OpCallExtern:  testOpcodeCoverageCallExtern,
		OpRet: func(t *testing.T) {
			var code CodeMemory
			code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
			if _, status := runOpcodeProgram(t, code, KindVoid); status != VMStatusOK {
				t.Fatalf("Run status = %s, want ok", status)
			}
		},
		OpBuiltIn: func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindFloat64, math.Float64bits(9))
			code.AppendInstruction(makeBuiltInInstruction(makeBuiltInFunction(BuiltInSqrt, KindFloat64)))
			_ = runOpcodeResult(t, code, KindFloat64)
		},
	}

	for opcode := OpPush; opcode < OpcodeCount; opcode++ {
		run := cases[opcode]
		if run == nil {
			t.Fatalf("opcode %d has no direct execution test", opcode)
		}
		t.Run("opcode_"+itoaByte(byte(opcode)), run)
	}
}

func testOpcodeCoverageJumpIfFalse(t *testing.T) {
	var code CodeMemory
	appendOpcodeValue(&code, KindBool, 0)
	code.AppendInstruction(makeInstruction(OpJumpIfFalse, KindNone, ModeNone, FlagNone))
	operand := len(code)
	code.AppendUint32(0)
	appendOpcodeValue(&code, KindInt32, 1)
	target := uint32(len(code))
	appendOpcodeValue(&code, KindInt32, 2)
	code.PatchUint32(operand, target)
	if got := runOpcodeResult(t, code, KindInt32); got != 2 {
		t.Fatalf("result = %d, want 2", got)
	}
}

func testOpcodeCoverageJump(t *testing.T) {
	var code CodeMemory
	code.AppendInstruction(makeInstruction(OpJump, KindNone, ModeNone, FlagNone))
	operand := len(code)
	code.AppendUint32(0)
	appendOpcodeValue(&code, KindInt32, 1)
	target := uint32(len(code))
	appendOpcodeValue(&code, KindInt32, 2)
	code.PatchUint32(operand, target)
	if got := runOpcodeResult(t, code, KindInt32); got != 2 {
		t.Fatalf("result = %d, want 2", got)
	}
}

func testOpcodeCoverageCall(t *testing.T) {
	var code CodeMemory
	code.AppendInstruction(makeInstruction(OpCall, KindNone, ModeNone, FlagNone))
	code.AppendUint32(1)
	code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
	helperAddress := uint32(len(code))
	appendOpcodeValue(&code, KindInt32, 42)
	code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
	program := &LinkedProgram{
		Text:       code,
		EntryPoint: 0,
		Functions: []ScriptFunctionDescriptor{
			{BodyAddress: 0, ReturnKind: KindInt32},
			{BodyAddress: helperAddress, ReturnKind: KindInt32},
		},
	}
	vm := NewVMWithConfig(VMConfig{StackCapacity: 16, CallFrameCapacity: 2})
	if status := vm.Run(program, 0); status != VMStatusOK {
		t.Fatalf("Run status = %s, want ok", status)
	}
	if result, status := vm.PopInt32(); status != VMStatusOK || result != 42 {
		t.Fatalf("result = %d, status=%s, want 42", result, status)
	}
}

func testOpcodeCoverageCallExtern(t *testing.T) {
	var code CodeMemory
	code.AppendInstruction(makeInstruction(OpCallExtern, KindNone, ModeNone, FlagNone))
	code.AppendUint32(7)
	code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
	program := &LinkedProgram{Text: code, EntryPoint: 0, Functions: []ScriptFunctionDescriptor{{BodyAddress: 0, ReturnKind: KindVoid}}}
	vm := NewVM(0)
	called := false
	vm.RegisterExternDispatcher(0, func(_ uintptr, _ *VM, importID uint32) VMStatus {
		called = importID == 7
		return VMStatusOK
	})
	if status := vm.Run(program, 0); status != VMStatusOK || !called {
		t.Fatalf("Run status = %s, called=%t, want ok and called", status, called)
	}
}

func TestOpcodeAddressAllSegments(t *testing.T) {
	segments := []memorySegment{segmentFrame, segmentBSS, segmentExtern, segmentConst, segmentData, segmentStack}
	for _, segment := range segments {
		t.Run(segment.String(), func(t *testing.T) {
			address := makeAddress(segment, 7)
			var code CodeMemory
			appendOpcodeValue(&code, KindAddress, uint64(address))
			if got := Address(uint32(runOpcodeResult(t, code, KindAddress))); got != address {
				t.Fatalf("address = %#x, want %#x", got, address)
			}
		})
	}
}

func TestOpcodeAddressIndexBoundary(t *testing.T) {
	for _, test := range []struct {
		name   string
		index  uint32
		status VMStatus
	}{
		{"maximum", addressIndexMask, VMStatusOK},
		{"overflow", addressIndexMask + 1, VMStatusInvalidAddress},
	} {
		t.Run(test.name, func(t *testing.T) {
			var code CodeMemory
			code.AppendInstruction(makeAddrInstruction(segmentData))
			code.AppendUint32(test.index)
			if test.status == VMStatusOK {
				if got := Address(uint32(runOpcodeResult(t, code, KindAddress))); got != makeAddress(segmentData, test.index) {
					t.Fatalf("address = %#x, want maximum data address", got)
				}
				return
			}
			if _, status := runOpcodeProgram(t, code, KindVoid); status != test.status {
				t.Fatalf("Run status = %s, want %s", status, test.status)
			}
		})
	}
}

func TestOpcodeFrameAddressUsesCurrentFrameBase(t *testing.T) {
	var code CodeMemory
	code.AppendInstruction(makeInstruction(OpCall, KindNone, ModeNone, FlagNone))
	code.AppendUint32(1)
	code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
	helperAddress := uint32(len(code))
	code.AppendInstruction(makeAddrInstruction(segmentFrame))
	code.AppendUint32(3)
	code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
	program := &LinkedProgram{
		Text:          code,
		EntryPoint:    0,
		FrameByteSize: 12,
		Functions: []ScriptFunctionDescriptor{
			{BodyAddress: 0, ReturnKind: KindAddress, FrameByteSize: 4},
			{BodyAddress: helperAddress, ReturnKind: KindAddress, FrameByteSize: 8},
		},
	}
	vm := NewVMWithConfig(VMConfig{FrameCapacity: 12, StackCapacity: 4, CallFrameCapacity: 2})
	if status := vm.Run(program, 0); status != VMStatusOK {
		t.Fatalf("Run status = %s, want ok", status)
	}
	bits, status := vm.PopBits(KindAddress)
	if status != VMStatusOK || Address(uint32(bits)) != makeAddress(segmentFrame, 7) {
		t.Fatalf("frame address = %#x, status=%s, want index 7", bits, status)
	}
}

func TestOpcodeOffsetPositiveNegativeAndZero(t *testing.T) {
	tests := []struct {
		name   string
		base   uint32
		offset int32
		want   uint32
	}{
		{"positive", 7, 3, 10},
		{"negative", 7, -3, 4},
		{"zero", 7, 0, 7},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var code CodeMemory
			appendOpcodeValue(&code, KindAddress, uint64(makeAddress(segmentData, test.base)))
			appendOpcodeValue(&code, KindInt32, uint64(uint32(test.offset)))
			code.AppendInstruction(makeInstruction(OpOffset, KindNone, ModeNone, FlagNone))
			got := Address(uint32(runOpcodeResult(t, code, KindAddress)))
			want := makeAddress(segmentData, test.want)
			if got != want {
				t.Fatalf("address = %#x, want %#x", got, want)
			}
		})
	}
}

func TestOpcodeAssignDereferenceAllStorableKinds(t *testing.T) {
	kinds := append(append([]ValueKind(nil), opcodeNumericKinds...), KindAddress)
	for _, kind := range kinds {
		t.Run(valueKindTestName(kind), func(t *testing.T) {
			bits := opcodeValueBits(kind, 7)
			if kind == KindAddress {
				bits = uint64(makeAddress(segmentData, 7))
			}
			var code CodeMemory
			appendOpcodeValue(&code, kind, bits)
			appendOpcodeValue(&code, KindAddress, uint64(makeAddress(segmentBSS, 0)))
			code.AppendInstruction(makeInstruction(OpAssign, kind, ModeNone, FlagNone))
			appendOpcodeValue(&code, KindAddress, uint64(makeAddress(segmentBSS, 0)))
			code.AppendInstruction(makeInstruction(OpDereference, kind, ModeNone, FlagNone))
			if got := runOpcodeResult(t, code, kind); got != bits {
				t.Fatalf("round-trip bits = %#x, want %#x", got, bits)
			}
		})
	}
}

func TestOpcodeAssignRejectsConstMemory(t *testing.T) {
	var code CodeMemory
	appendOpcodeValue(&code, KindInt32, 1)
	appendOpcodeValue(&code, KindAddress, uint64(makeAddress(segmentConst, 0)))
	code.AppendInstruction(makeInstruction(OpAssign, KindInt32, ModeNone, FlagNone))
	if _, status := runOpcodeProgram(t, code, KindVoid); status != VMStatusReadOnlyMemory {
		t.Fatalf("Run status = %s, want read-only memory", status)
	}
}

func TestOpcodeMemoryOperationsRejectInvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		code func(*CodeMemory)
		want VMStatus
	}{
		{"dereference_kind", func(code *CodeMemory) {
			appendOpcodeValue(code, KindAddress, uint64(makeAddress(segmentBSS, 0)))
			code.AppendInstruction(makeInstruction(OpDereference, KindNone, ModeNone, FlagNone))
		}, VMStatusInvalidValueKind},
		{"assign_kind", func(code *CodeMemory) {
			appendOpcodeValue(code, KindAddress, uint64(makeAddress(segmentBSS, 0)))
			code.AppendInstruction(makeInstruction(OpAssign, KindNone, ModeNone, FlagNone))
		}, VMStatusInvalidValueKind},
		{"dereference_segment", func(code *CodeMemory) {
			appendOpcodeValue(code, KindAddress, uint64(makeAddress(segmentInvalid, 0)))
			code.AppendInstruction(makeInstruction(OpDereference, KindInt32, ModeNone, FlagNone))
		}, VMStatusInvalidAddressSegment},
		{"dereference_bounds", func(code *CodeMemory) {
			appendOpcodeValue(code, KindAddress, uint64(makeAddress(segmentBSS, 31)))
			code.AppendInstruction(makeInstruction(OpDereference, KindInt32, ModeNone, FlagNone))
		}, VMStatusInvalidAddress},
		{"dereference_underflow", func(code *CodeMemory) {
			code.AppendInstruction(makeInstruction(OpDereference, KindInt32, ModeNone, FlagNone))
		}, VMStatusStackUnderflow},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var code CodeMemory
			test.code(&code)
			if _, status := runOpcodeProgram(t, code, KindVoid); status != test.want {
				t.Fatalf("Run status = %s, want %s", status, test.want)
			}
		})
	}
}

func TestOpcodeJumpsBothPathsAndRejectInvalidTargets(t *testing.T) {
	t.Run("conditional_not_taken", func(t *testing.T) {
		var code CodeMemory
		appendOpcodeValue(&code, KindBool, 2)
		code.AppendInstruction(makeInstruction(OpJumpIfFalse, KindNone, ModeNone, FlagNone))
		code.AppendUint32(0xffffffff)
		appendOpcodeValue(&code, KindInt32, 9)
		if got := runOpcodeResult(t, code, KindInt32); got != 9 {
			t.Fatalf("result = %d, want 9", got)
		}
	})
	for _, opcode := range []Opcode{OpJumpIfFalse, OpJump} {
		t.Run("invalid_"+itoaByte(byte(opcode)), func(t *testing.T) {
			var code CodeMemory
			if opcode == OpJumpIfFalse {
				appendOpcodeValue(&code, KindBool, 0)
			}
			code.AppendInstruction(makeInstruction(opcode, KindNone, ModeNone, FlagNone))
			code.AppendUint32(0xffffffff)
			if _, status := runOpcodeProgram(t, code, KindVoid); status != VMStatusInvalidTarget {
				t.Fatalf("Run status = %s, want invalid target", status)
			}
		})
	}
}

func TestOpcodeRejectsUnknownOpcode(t *testing.T) {
	for _, opcode := range []Opcode{0, Opcode(0x3f)} {
		t.Run("opcode_"+itoaByte(byte(opcode)), func(t *testing.T) {
			var code CodeMemory
			code.AppendInstruction(makeInstruction(opcode, KindNone, ModeNone, FlagNone))
			if _, status := runOpcodeProgram(t, code, KindVoid); status != VMStatusInvalidOpcode {
				t.Fatalf("Run status = %s, want invalid opcode", status)
			}
		})
	}
}

func TestOpcodeCallFrameOverflow(t *testing.T) {
	var code CodeMemory
	code.AppendInstruction(makeInstruction(OpCall, KindNone, ModeNone, FlagNone))
	code.AppendUint32(0)
	program := &LinkedProgram{Text: code, EntryPoint: 0, Functions: []ScriptFunctionDescriptor{{BodyAddress: 0, ReturnKind: KindVoid}}}
	vm := NewVMWithConfig(VMConfig{CallFrameCapacity: 1})
	if status := vm.Run(program, 0); status != VMStatusCallFrameOverflow {
		t.Fatalf("Run status = %s, want call-frame overflow", status)
	}
}

func TestOpcodeCallAndReturnAllStorableKinds(t *testing.T) {
	kinds := append(append([]ValueKind(nil), opcodeNumericKinds...), KindAddress)
	for _, kind := range kinds {
		t.Run(valueKindTestName(kind), func(t *testing.T) {
			bits := opcodeValueBits(kind, 7)
			if kind == KindAddress {
				bits = uint64(makeAddress(segmentData, 7))
			}
			var code CodeMemory
			code.AppendInstruction(makeInstruction(OpCall, KindNone, ModeNone, FlagNone))
			code.AppendUint32(1)
			code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
			helperAddress := uint32(len(code))
			appendOpcodeValue(&code, kind, bits)
			code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
			program := &LinkedProgram{
				Text:       code,
				EntryPoint: 0,
				Functions: []ScriptFunctionDescriptor{
					{BodyAddress: 0, ReturnKind: kind},
					{BodyAddress: helperAddress, ReturnKind: kind},
				},
			}
			vm := NewVMWithConfig(VMConfig{StackCapacity: 16, CallFrameCapacity: 2})
			if status := vm.Run(program, 0); status != VMStatusOK {
				t.Fatalf("Run status = %s, want ok", status)
			}
			got, status := vm.PopBits(kind)
			if status != VMStatusOK || got != bits {
				t.Fatalf("return bits = %#x, status=%s, want %#x", got, status, bits)
			}
		})
	}
}

func TestOpcodeReturnValueUnderflow(t *testing.T) {
	var code CodeMemory
	code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
	if _, status := runOpcodeProgram(t, code, KindInt64); status != VMStatusStackUnderflow {
		t.Fatalf("Run status = %s, want stack underflow", status)
	}
}

func TestOpcodeCallExternMissingDispatcher(t *testing.T) {
	var code CodeMemory
	code.AppendInstruction(makeInstruction(OpCallExtern, KindNone, ModeNone, FlagNone))
	code.AppendUint32(3)
	if _, status := runOpcodeProgram(t, code, KindVoid); status != VMStatusMissingExtern {
		t.Fatalf("Run status = %s, want missing extern", status)
	}
}
