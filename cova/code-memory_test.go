package cova

import "testing"

func TestInstructionOpcodeRoundTrips(t *testing.T) {
	for opcode := OpPush; opcode < OpcodeCount; opcode++ {
		instruction := makeInstruction(opcode, KindUint32, ModeNone, FlagNone)
		var code CodeMemory
		code.AppendInstruction(instruction)
		var ip uint32
		decoded, status := code.ReadInstructionChecked(&ip)
		if status != VMStatusOK || decoded.Opcode() != opcode || decoded.Kind() != KindUint32 || ip != 2 {
			t.Fatalf("opcode %d round trip: decoded=%#x ip=%d status=%s", opcode, decoded, ip, status)
		}
	}
}

func TestSpecializedInstructionPayloadsRoundTrip(t *testing.T) {
	for function := BuiltInFunction(0); function <= 0x07ff; function++ {
		instruction := makeBuiltInInstruction(function)
		if instruction.Opcode() != OpBuiltIn || instruction.BuiltInFunction() != function {
			t.Fatalf("built-in function %d did not round trip", function)
		}
	}
	for _, operation := range []ArithmeticOp{ArithmeticAdd, ArithmeticSub, ArithmeticMul, ArithmeticDiv} {
		instruction := makeArithmeticInstruction(KindInt16, operation)
		if instruction.Opcode() != OpArithmetic || instruction.Kind() != KindInt16 || instruction.ArithmeticOp() != operation {
			t.Fatalf("arithmetic operation %d did not round trip", operation)
		}
	}
	for _, operation := range []CompareOp{CompareEqual, CompareNotEqual, CompareLess, CompareLessEqual, CompareGreater, CompareGreaterEqual} {
		instruction := makeCompareInstruction(KindFloat32, operation)
		if instruction.Opcode() != OpCompare || instruction.Kind() != KindFloat32 || instruction.CompareOp() != operation {
			t.Fatalf("compare operation %d did not round trip", operation)
		}
	}
	for from := KindNone; from < KindCount; from++ {
		for to := KindNone; to < KindCount; to++ {
			instruction := makeConvertInstruction(from, to)
			if instruction.Opcode() != OpConvert || instruction.ConvertFromKind() != from || instruction.Kind() != to {
				t.Fatalf("conversion %d -> %d did not round trip", from, to)
			}
		}
	}
	for segment := segmentInvalid; segment < segmentCount; segment++ {
		instruction := makeAddrInstruction(segment)
		if instruction.Opcode() != OpAddr || instruction.AddressSegment() != segment {
			t.Fatalf("address segment %d did not round trip", segment)
		}
	}
}

func TestCodeImmediateWidthsRoundTrip(t *testing.T) {
	tests := []struct {
		kind ValueKind
		bits uint64
		size uint32
	}{
		{KindUint8, 0xa5, 1},
		{KindUint16, 0xa5b6, 2},
		{KindUint32, 0xa5b6c7d8, 4},
		{KindUint64, 0xa5b6c7d8e9fa1023, 8},
	}
	for _, test := range tests {
		var code CodeMemory
		code.AppendImmediate(test.kind, test.bits)
		var ip uint32
		bits, status := code.ReadImmediateChecked(&ip, test.kind)
		if status != VMStatusOK || bits != test.bits || ip != test.size {
			t.Fatalf("kind %d immediate = %#x ip=%d status=%s, want %#x ip=%d", test.kind, bits, ip, status, test.bits, test.size)
		}
	}

	var code CodeMemory
	code.AppendUint32(0x89abcdef)
	var ip uint32
	value, status := code.ReadUint32Checked(&ip)
	if status != VMStatusOK || value != 0x89abcdef || ip != 4 {
		t.Fatalf("uint32 operand = %#x ip=%d status=%s", value, ip, status)
	}
}

func TestValueKindSizeRejectsInvalidKind(t *testing.T) {
	if got := ValueKind(255).Size(); got != 0 {
		t.Fatalf("expected invalid kind size 0, got %d", got)
	}
}

func TestCheckedCodeReadersRejectTruncatedInput(t *testing.T) {
	tests := []struct {
		name string
		read func(CodeMemory) VMStatus
	}{
		{
			name: "instruction",
			read: func(code CodeMemory) VMStatus {
				var ip uint32
				_, status := code.ReadInstructionChecked(&ip)
				return status
			},
		},
		{
			name: "immediate",
			read: func(code CodeMemory) VMStatus {
				var ip uint32
				_, status := code.ReadImmediateChecked(&ip, KindUint32)
				return status
			},
		},
		{
			name: "int operand",
			read: func(code CodeMemory) VMStatus {
				var ip uint32
				_, status := code.ReadUint32Checked(&ip)
				return status
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if status := test.read(CodeMemory{0}); status != VMStatusMalformedBytecode {
				t.Fatal("expected truncated input error")
			}
		})
	}
}

func TestRunRejectsTruncatedInstructionWithoutPanic(t *testing.T) {
	code := CodeMemory{byte(OpRet)}
	program := &LinkedProgram{
		Text:       code,
		EntryPoint: 0,
		Functions:  []ScriptFunctionDescriptor{{BodyAddress: 0, ReturnKind: KindVoid}},
	}

	if status := NewVM(0).Run(program, 0); status != VMStatusMalformedBytecode {
		t.Fatalf("expected malformed bytecode status, got %s", status)
	}
}

func TestRunRejectsTruncatedOperandWithoutPanic(t *testing.T) {
	code := CodeMemory{}
	code.AppendInstruction(makeInstruction(OpJump, KindNone, ModeNone, FlagNone))
	code = append(code, 0)
	program := &LinkedProgram{
		Text:       code,
		EntryPoint: 0,
		Functions:  []ScriptFunctionDescriptor{{BodyAddress: 0, ReturnKind: KindVoid}},
	}

	if status := NewVM(0).Run(program, 0); status != VMStatusMalformedBytecode {
		t.Fatalf("expected malformed bytecode status, got %s", status)
	}
}
