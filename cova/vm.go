package cova

import (
	"math"
	"unsafe"
)

type callFrame struct {
	returnPC   uint32
	localBase  uint32
	returnKind ValueKind
}

type VM struct {
	memory           ProgramMemory
	pc               uint32
	program          *LinkedProgram
	externDispatcher ExternDispatcherBinding
	callFrames       []callFrame
	callFrameTop     uint32
	frameTop         uint32
	fault            VMFaultInfo
	instructionPC    uint32
	hasInstructionPC bool
	randomSeed       uint64
	randomS0         uint64
	randomS1         uint64
}

const defaultRandomSeed uint64 = 0x1234567890abcdef
const randomStateSeed uint64 = 6364136223846793005

type VMConfig struct {
	FrameCapacity     uint32
	StackCapacity     uint32
	CallFrameCapacity uint32
}

func NewVM(frameCapacity uint32) *VM {
	return NewVMWithConfig(VMConfig{
		FrameCapacity:     frameCapacity,
		StackCapacity:     256,
		CallFrameCapacity: 8,
	})
}

func NewVMWithConfig(config VMConfig) *VM {
	if config.CallFrameCapacity < 1 {
		config.CallFrameCapacity = 1
	}
	vm := &VM{
		memory:     NewProgramMemory(0, 0, 0, 0, config.FrameCapacity, config.StackCapacity),
		callFrames: make([]callFrame, config.CallFrameCapacity),
		fault:      noVMFault(),
	}
	vm.SetRandomSeed(defaultRandomSeed)
	return vm
}

func (vm *VM) nextRandom() uint64 {
	s1 := vm.randomS0
	s0 := vm.randomS1
	result := s0 + s1
	vm.randomS0 = s0
	s1 ^= s1 << 23
	vm.randomS1 = s1 ^ s0 ^ (s1 >> 18) ^ (s0 >> 5)
	return result
}

func (vm *VM) resetRandom() {
	vm.randomS0 = vm.randomSeed + 6364136223846793
	vm.randomS1 = randomStateSeed
	vm.nextRandom()
	vm.nextRandom()
}

func (vm *VM) SetRandomSeed(seed uint64) {
	vm.randomSeed = seed
	vm.resetRandom()
}

func (vm *VM) AllocateExternMemory(size uint32) {
	vm.memory.segment[segmentExtern] = NewMemorySegment(size, size)
}

func (vm *VM) BindExternBlock(block []byte) {
	vm.memory.segment[segmentExtern] = block
}

func (vm *VM) LoadExternInt32(offset uint32) int32 {
	value, status := vm.memory.ReadInt32(makeAddress(segmentExtern, offset))
	if status != VMStatusOK {
		return 0
	}
	return value
}

func (vm *VM) StoreExternInt32(offset uint32, value int32) {
	_ = vm.memory.WriteInt32(makeAddress(segmentExtern, offset), value)
}

func (vm *VM) RegisterExternDispatcher(hostContext uintptr, dispatcher ExternDispatcher) {
	vm.externDispatcher = ExternDispatcherBinding{HostContext: hostContext, Dispatcher: dispatcher}
}

func (vm *VM) FaultInfo() VMFaultInfo {
	return vm.fault
}

func (vm *VM) PushBits(kind ValueKind, bits uint64) VMStatus {
	return vm.pushKind(kind, bits)
}

func (vm *VM) PushBool(value bool) VMStatus {
	if value {
		return vm.pushUint8(1)
	}
	return vm.pushUint8(0)
}

func (vm *VM) PushByte(value byte) VMStatus {
	return vm.pushUint8(value)
}

func (vm *VM) PushInt8(value int8) VMStatus {
	return vm.pushUint8(uint8(value))
}

func (vm *VM) PushInt16(value int16) VMStatus {
	return vm.pushUint16(uint16(value))
}

func (vm *VM) PushInt32(value int32) VMStatus {
	return vm.pushUint32(uint32(value))
}

func (vm *VM) PushInt64(value int64) VMStatus {
	return vm.pushUint64(uint64(value))
}

func (vm *VM) PushUint8(value uint8) VMStatus {
	return vm.pushUint8(value)
}

func (vm *VM) PushUint16(value uint16) VMStatus {
	return vm.pushUint16(value)
}

func (vm *VM) PushUint32(value uint32) VMStatus {
	return vm.pushUint32(value)
}

func (vm *VM) PushUint64(value uint64) VMStatus {
	return vm.pushUint64(value)
}

func (vm *VM) PushFloat32(value float32) VMStatus {
	return vm.pushUint32(math.Float32bits(value))
}

func (vm *VM) PushFloat64(value float64) VMStatus {
	return vm.pushUint64(math.Float64bits(value))
}

func (vm *VM) PopBits(kind ValueKind) (uint64, VMStatus) {
	return vm.popKind(kind)
}

func (vm *VM) PopBool() (bool, VMStatus) {
	value, status := vm.popUint8()
	return value != 0, status
}

func (vm *VM) PopByte() (byte, VMStatus) {
	return vm.popUint8()
}

func (vm *VM) PopInt8() (int8, VMStatus) {
	value, status := vm.popUint8()
	return int8(value), status
}

func (vm *VM) PopInt16() (int16, VMStatus) {
	value, status := vm.popUint16()
	return int16(value), status
}

func (vm *VM) PopInt32() (int32, VMStatus) {
	value, status := vm.popUint32()
	return int32(value), status
}

func (vm *VM) PopInt64() (int64, VMStatus) {
	value, status := vm.popUint64()
	return int64(value), status
}

func (vm *VM) PopUint8() (uint8, VMStatus) {
	return vm.popUint8()
}

func (vm *VM) PopUint16() (uint16, VMStatus) {
	return vm.popUint16()
}

func (vm *VM) PopUint32() (uint32, VMStatus) {
	return vm.popUint32()
}

func (vm *VM) PopUint64() (uint64, VMStatus) {
	return vm.popUint64()
}

func (vm *VM) PopFloat32() (float32, VMStatus) {
	bits, status := vm.popUint32()
	return math.Float32frombits(bits), status
}

func (vm *VM) PopFloat64() (float64, VMStatus) {
	bits, status := vm.popUint64()
	return math.Float64frombits(bits), status
}

func (vm *VM) Run(program *LinkedProgram) VMStatus {
	if status := vm.LoadProgram(program); status != VMStatusOK {
		return status
	}
	return vm.RunLoaded()
}

func (vm *VM) RunImage(blob []byte) VMStatus {
	if status := vm.LoadProgramImage(blob); status != VMStatusOK {
		return status
	}
	return vm.RunLoaded()
}

func (vm *VM) LoadProgram(program *LinkedProgram) VMStatus {
	vm.clearFault()
	if program == nil {
		return vm.setFault(VMStatusInvalidProgram, -1, -1, -1)
	}
	if uint64(program.ConstByteSize) != uint64(len(program.ConstData)) {
		return vm.setFault(VMStatusInvalidImage, -1, int(program.ConstByteSize), len(program.ConstData))
	}
	if uint64(program.DataByteSize) != uint64(len(program.DataData)) {
		return vm.setFault(VMStatusInvalidImage, -1, int(program.DataByteSize), len(program.DataData))
	}
	if uint64(len(program.Text)) > uint64(^uint32(0)) {
		return vm.setFault(VMStatusInvalidImage, -1, int(^uint32(0)), len(program.Text))
	}
	bssByteSize, ok := hostIntFromUint32(program.BSSByteSize)
	if !ok {
		return vm.setFault(VMStatusInvalidImage, -1, int(program.BSSByteSize), -1)
	}
	dataByteSize, ok := hostIntFromUint32(program.DataByteSize)
	if !ok {
		return vm.setFault(VMStatusInvalidImage, -1, int(program.DataByteSize), -1)
	}
	if len(vm.memory.segment[segmentBSS]) != bssByteSize {
		vm.memory.segment[segmentBSS] = NewMemorySegment(program.BSSByteSize, program.BSSByteSize)
	}
	if len(vm.memory.segment[segmentData]) != dataByteSize {
		vm.memory.segment[segmentData] = NewMemorySegment(program.DataByteSize, program.DataByteSize)
	}
	vm.memory.segment[segmentConst] = MemorySegment(program.ConstData)
	vm.program = program
	return VMStatusOK
}

func (vm *VM) LoadProgramImage(blob []byte) VMStatus {
	vm.clearFault()
	image, status := OpenProgramImage(blob)
	if status != VMStatusOK {
		return vm.setFault(status, -1, -1, -1)
	}
	program, status := linkedProgramFromImage(image)
	if status != VMStatusOK {
		return vm.setFault(status, -1, -1, -1)
	}
	return vm.LoadProgram(program)
}

func (vm *VM) Reset() VMStatus {
	vm.clearFault()
	if vm.program == nil {
		return vm.setFault(VMStatusNoProgramLoaded, -1, -1, -1)
	}
	program := vm.program
	for index := range vm.memory.segment[segmentBSS] {
		vm.memory.segment[segmentBSS][index] = 0
	}
	copy(vm.memory.segment[segmentData], program.DataData)
	vm.pc = 0
	vm.memory.segment[segmentStack] = vm.memory.segment[segmentStack][:0]
	for index := range vm.memory.segment[segmentFrame] {
		vm.memory.segment[segmentFrame][index] = 0
	}
	vm.callFrameTop = 0
	vm.frameTop = 0
	vm.resetRandom()

	entryPoint := program.EntryPoint
	if entryPoint >= uint32(len(program.Functions)) {
		return vm.setFault(VMStatusInvalidTarget, int(entryPoint), len(program.Functions), len(program.Functions))
	}
	if program.Functions[int(entryPoint)].ParamCount != 0 {
		return vm.setFault(VMStatusInvalidDescriptor, int(entryPoint), 0, int(program.Functions[int(entryPoint)].ParamCount))
	}
	if status := vm.enterScriptFunction(entryPoint, 0, false); status != VMStatusOK {
		return vm.recordStatus(status)
	}
	return VMStatusOK
}

func (vm *VM) RunLoaded() VMStatus {
	if status := vm.Reset(); status != VMStatusOK {
		return status
	}
	program := vm.program
	for vm.pc < uint32(len(program.Text)) {
		vm.instructionPC = vm.pc
		vm.hasInstructionPC = true
		instruction, status := program.Text.ReadInstructionChecked(&vm.pc)
		if status != VMStatusOK {
			return vm.recordStatus(status)
		}
		op := instruction.Opcode()

		switch op {
		case OpPush:
			kind := instruction.Kind()
			if kind == KindNone || kind == KindAddress {
				return vm.setFault(VMStatusInvalidValueKind, int(kind), -1, -1)
			}
			immediate, status := program.Text.ReadImmediateChecked(&vm.pc, kind)
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if status = vm.pushKind(kind, immediate); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		case OpArithmetic:
			kind := instruction.Kind()
			arithmeticOp := instruction.ArithmeticOp()
			if status = vm.executeArithmetic(kind, arithmeticOp); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		case OpConvert:
			fromKind := instruction.ConvertFromKind()
			kind := instruction.Kind()
			if status = vm.executeConversion(fromKind, kind); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		case OpAddr:
			segment := instruction.AddressSegment()
			offset, status := program.Text.ReadUint32Checked(&vm.pc)
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if segment == segmentFrame {
				frame, frameStatus := vm.currentFrame()
				if frameStatus != VMStatusOK {
					return vm.recordStatus(frameStatus)
				}
				if frame.localBase > addressIndexMask || offset > addressIndexMask-frame.localBase {
					return vm.setFault(VMStatusInvalidAddress, int(offset), int(addressIndexMask), int(uint64(frame.localBase)+uint64(offset)))
				}
				offset += frame.localBase
			} else if offset > addressIndexMask {
				return vm.setFault(VMStatusInvalidAddress, int(offset), int(addressIndexMask), int(offset))
			}
			if status = vm.pushAddress(makeAddress(segment, offset)); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		case OpOffset:
			offset, status := vm.PopInt32()
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			base, status := vm.popAddress()
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			newIndex := int64(base.Index()) + int64(offset)
			if newIndex < 0 || newIndex > int64(addressIndexMask) {
				return vm.setFault(VMStatusInvalidAddress, int(offset), int(addressIndexMask), int(newIndex))
			}
			if status = vm.pushAddress(makeAddress(base.Segment(), uint32(newIndex))); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		case OpDereference:
			kind := instruction.Kind()
			if kind == KindNone {
				return vm.setFault(VMStatusInvalidValueKind, int(kind), -1, -1)
			}
			encodedAddress, status := vm.popAddress()
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if status = vm.pushFromMemory(encodedAddress, kind); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		case OpAssign:
			kind := instruction.Kind()
			if kind == KindNone {
				return vm.setFault(VMStatusInvalidValueKind, int(kind), -1, -1)
			}
			encodedAddress, status := vm.popAddress()
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if status = vm.popToMemory(encodedAddress, kind); status != VMStatusOK {
				return vm.setFault(status, int(encodedAddress), -1, -1)
			}
		case OpCompare:
			kind := instruction.Kind()
			compareOp := instruction.CompareOp()
			result, status := vm.executeTypedComparison(kind, compareOp)
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if status = vm.PushBool(result); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		case OpJumpIfFalse:
			target, status := program.Text.ReadUint32Checked(&vm.pc)
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			condition, status := vm.PopBool()
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if !condition {
				if target >= uint32(len(program.Text)) {
					return vm.setFault(VMStatusInvalidTarget, int(target), len(program.Text), len(program.Text))
				}
				vm.pc = target
			}
		case OpJump:
			target, status := program.Text.ReadUint32Checked(&vm.pc)
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if target >= uint32(len(program.Text)) {
				return vm.setFault(VMStatusInvalidTarget, int(target), len(program.Text), len(program.Text))
			}
			vm.pc = target
		case OpCall:
			target, status := program.Text.ReadUint32Checked(&vm.pc)
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if status = vm.callScriptFunction(target); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		case OpCallExtern:
			importID, status := program.Text.ReadUint32Checked(&vm.pc)
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if status = vm.callExtern(importID); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		case OpRet:
			done, status := vm.returnFromFunction()
			if status != VMStatusOK {
				return vm.recordStatus(status)
			}
			if done {
				return VMStatusOK
			}
		case OpBuiltIn:
			if status = vm.executeBuiltIn(instruction.BuiltInFunction()); status != VMStatusOK {
				return vm.recordStatus(status)
			}
		default:
			return vm.setFault(VMStatusInvalidOpcode, int(op), -1, -1)
		}
	}

	return VMStatusOK
}

func (vm *VM) executeBuiltIn(function BuiltInFunction) VMStatus {
	operation := function.Operation()
	kind := function.Kind()
	if operation == BuiltInAbs {
		return vm.executeBuiltInAbs(kind)
	}
	if operation == BuiltInMin || operation == BuiltInMax {
		return vm.executeBuiltInMinMax(operation, kind)
	}
	if operation == BuiltInMap {
		return vm.executeBuiltInMap(kind)
	}
	if operation == BuiltInClamp {
		return vm.executeBuiltInClamp(kind)
	}
	if operation == BuiltInRandom {
		if kind != KindInt32 {
			return VMStatusInvalidValueKind
		}
		return vm.PushBits(kind, vm.nextRandom()&0x7fffffff)
	}
	if (operation == BuiltInLerp || operation == BuiltInSmoothStep) && (kind == KindInt32 || kind == KindInt64) {
		return vm.executeBuiltInFixedPoint(operation, kind)
	}
	if kind != KindFloat32 && kind != KindFloat64 {
		return VMStatusInvalidValueKind
	}
	if operation == BuiltInPow {
		return vm.executeBuiltInPow(kind)
	}
	var apply func(float64) float64
	switch operation {
	case BuiltInSin:
		apply = math.Sin
	case BuiltInCos:
		apply = math.Cos
	case BuiltInTan:
		apply = math.Tan
	case BuiltInAsin:
		apply = math.Asin
	case BuiltInAcos:
		apply = math.Acos
	case BuiltInAtan:
		apply = math.Atan
	case BuiltInSqrt:
		apply = math.Sqrt
	case BuiltInSmoothStep, BuiltInLerp, BuiltInSlerp:
		return vm.executeBuiltInFloatInterpolation(operation, kind)
	default:
		return VMStatusInvalidOpcode
	}
	if kind == KindFloat32 {
		value, status := vm.PopFloat32()
		if status != VMStatusOK {
			return status
		}
		return vm.PushFloat32(float32(apply(float64(value))))
	}
	value, status := vm.PopFloat64()
	if status != VMStatusOK {
		return status
	}
	return vm.PushFloat64(apply(value))
}

type orderedNumber interface {
	int32 | int64 | uint32 | uint64 | float32 | float64
}

func clampOrdered[T orderedNumber](value, low, high T) (T, bool) {
	if !(low <= high) {
		return 0, false
	}
	if value < low {
		return low, true
	}
	if value > high {
		return high, true
	}
	return value, true
}

func mapOrdered[T orderedNumber](value, inLow, inHigh, outLow, outHigh T) (T, bool) {
	if inLow > inHigh {
		inLow, inHigh = inHigh, inLow
	}
	if outLow > outHigh {
		outLow, outHigh = outHigh, outLow
	}
	if !(inLow < inHigh) {
		return 0, false
	}
	value, _ = clampOrdered(value, inLow, inHigh)
	return (value-inLow)*(outHigh-outLow)/(inHigh-inLow) + outLow, true
}

func (vm *VM) popBuiltInBits(kind ValueKind, count int) ([]uint64, VMStatus) {
	values := make([]uint64, count)
	for index := count - 1; index >= 0; index-- {
		bits, status := vm.PopBits(kind)
		if status != VMStatusOK {
			return nil, status
		}
		values[index] = bits
	}
	return values, VMStatusOK
}

func (vm *VM) executeBuiltInMap(kind ValueKind) VMStatus {
	values, status := vm.popBuiltInBits(kind, 5)
	if status != VMStatusOK {
		return status
	}
	var result uint64
	valid := true
	switch kind {
	case KindByte, KindUint8:
		mapped, ok := mapOrdered(uint32(uint8(values[0])), uint32(uint8(values[1])), uint32(uint8(values[2])), uint32(uint8(values[3])), uint32(uint8(values[4])))
		result, valid = uint64(uint8(mapped)), ok
	case KindInt8:
		mapped, ok := mapOrdered(int32(int8(values[0])), int32(int8(values[1])), int32(int8(values[2])), int32(int8(values[3])), int32(int8(values[4])))
		result, valid = uint64(uint8(int8(mapped))), ok
	case KindInt16:
		mapped, ok := mapOrdered(int32(int16(values[0])), int32(int16(values[1])), int32(int16(values[2])), int32(int16(values[3])), int32(int16(values[4])))
		result, valid = uint64(uint16(int16(mapped))), ok
	case KindInt32:
		mapped, ok := mapOrdered(int32(values[0]), int32(values[1]), int32(values[2]), int32(values[3]), int32(values[4]))
		result, valid = uint64(uint32(mapped)), ok
	case KindInt64:
		mapped, ok := mapOrdered(int64(values[0]), int64(values[1]), int64(values[2]), int64(values[3]), int64(values[4]))
		result, valid = uint64(mapped), ok
	case KindUint16:
		mapped, ok := mapOrdered(uint32(uint16(values[0])), uint32(uint16(values[1])), uint32(uint16(values[2])), uint32(uint16(values[3])), uint32(uint16(values[4])))
		result, valid = uint64(uint16(mapped)), ok
	case KindUint32:
		mapped, ok := mapOrdered(uint32(values[0]), uint32(values[1]), uint32(values[2]), uint32(values[3]), uint32(values[4]))
		result, valid = uint64(mapped), ok
	case KindUint64:
		result, valid = mapOrdered(values[0], values[1], values[2], values[3], values[4])
	case KindFloat32:
		mapped, ok := mapOrdered(math.Float32frombits(uint32(values[0])), math.Float32frombits(uint32(values[1])), math.Float32frombits(uint32(values[2])), math.Float32frombits(uint32(values[3])), math.Float32frombits(uint32(values[4])))
		result, valid = uint64(math.Float32bits(mapped)), ok
	case KindFloat64:
		mapped, ok := mapOrdered(math.Float64frombits(values[0]), math.Float64frombits(values[1]), math.Float64frombits(values[2]), math.Float64frombits(values[3]), math.Float64frombits(values[4]))
		result, valid = math.Float64bits(mapped), ok
	default:
		return VMStatusInvalidValueKind
	}
	if !valid {
		return VMStatusDivisionByZero
	}
	return vm.PushBits(kind, result)
}

func (vm *VM) executeBuiltInClamp(kind ValueKind) VMStatus {
	values, status := vm.popBuiltInBits(kind, 3)
	if status != VMStatusOK {
		return status
	}
	var result uint64
	valid := true
	switch kind {
	case KindByte, KindUint8:
		clamped, ok := clampOrdered(uint32(uint8(values[0])), uint32(uint8(values[1])), uint32(uint8(values[2])))
		result, valid = uint64(uint8(clamped)), ok
	case KindInt8:
		clamped, ok := clampOrdered(int32(int8(values[0])), int32(int8(values[1])), int32(int8(values[2])))
		result, valid = uint64(uint8(int8(clamped))), ok
	case KindInt16:
		clamped, ok := clampOrdered(int32(int16(values[0])), int32(int16(values[1])), int32(int16(values[2])))
		result, valid = uint64(uint16(int16(clamped))), ok
	case KindInt32:
		clamped, ok := clampOrdered(int32(values[0]), int32(values[1]), int32(values[2]))
		result, valid = uint64(uint32(clamped)), ok
	case KindInt64:
		clamped, ok := clampOrdered(int64(values[0]), int64(values[1]), int64(values[2]))
		result, valid = uint64(clamped), ok
	case KindUint16:
		clamped, ok := clampOrdered(uint32(uint16(values[0])), uint32(uint16(values[1])), uint32(uint16(values[2])))
		result, valid = uint64(uint16(clamped)), ok
	case KindUint32:
		clamped, ok := clampOrdered(uint32(values[0]), uint32(values[1]), uint32(values[2]))
		result, valid = uint64(clamped), ok
	case KindUint64:
		result, valid = clampOrdered(values[0], values[1], values[2])
	case KindFloat32:
		clamped, ok := clampOrdered(math.Float32frombits(uint32(values[0])), math.Float32frombits(uint32(values[1])), math.Float32frombits(uint32(values[2])))
		result, valid = uint64(math.Float32bits(clamped)), ok
	case KindFloat64:
		clamped, ok := clampOrdered(math.Float64frombits(values[0]), math.Float64frombits(values[1]), math.Float64frombits(values[2]))
		result, valid = math.Float64bits(clamped), ok
	default:
		return VMStatusInvalidValueKind
	}
	if !valid {
		return VMStatusInvalidParameter
	}
	return vm.PushBits(kind, result)
}

func (vm *VM) executeBuiltInFloatInterpolation(operation BuiltInOperation, kind ValueKind) VMStatus {
	pop := func() (float64, VMStatus) {
		if kind == KindFloat32 {
			value, status := vm.PopFloat32()
			return float64(value), status
		}
		return vm.PopFloat64()
	}
	t, status := pop()
	if status != VMStatusOK {
		return status
	}
	b, status := pop()
	if status != VMStatusOK {
		return status
	}
	a, status := pop()
	if status != VMStatusOK {
		return status
	}
	var result float64
	switch operation {
	case BuiltInSmoothStep:
		if t <= a {
			result = 0
		} else if t >= b {
			result = 1
		} else {
			normalized := (t - a) / (b - a)
			result = normalized * normalized * (3 - 2*normalized)
		}
	case BuiltInLerp:
		result = a + (b-a)*t
	case BuiltInSlerp:
		theta := math.Acos(a * b)
		if math.Abs(theta) < 0.00001 {
			result = a
		} else {
			result = (math.Sin((1-t)*theta)*a + math.Sin(t*theta)*b) / math.Sin(theta)
		}
	default:
		return VMStatusInvalidOpcode
	}
	if kind == KindFloat32 {
		return vm.PushFloat32(float32(result))
	}
	return vm.PushFloat64(result)
}

func (vm *VM) executeBuiltInFixedPoint(operation BuiltInOperation, kind ValueKind) VMStatus {
	shiftBits, status := vm.PopBits(KindUint8)
	if status != VMStatusOK {
		return status
	}
	shift := uint8(shiftBits)
	tBits, status := vm.PopBits(kind)
	if status != VMStatusOK {
		return status
	}
	endBits, status := vm.PopBits(kind)
	if status != VMStatusOK {
		return status
	}
	startBits, status := vm.PopBits(kind)
	if status != VMStatusOK {
		return status
	}
	if kind == KindInt32 {
		if shift >= 31 {
			return VMStatusInvalidParameter
		}
		start, end, t := int32(startBits), int32(endBits), int32(tBits)
		maxT := int32(1) << shift
		if t <= 0 {
			return vm.PushBits(kind, uint64(uint32(start)))
		}
		if t >= maxT {
			return vm.PushBits(kind, uint64(uint32(end)))
		}
		if operation == BuiltInLerp {
			return vm.PushBits(kind, uint64(uint32(start+((end-start)*t>>shift))))
		}
		curveShift := shift
		if curveShift > 15 {
			curveShift = 15
		}
		threeScaled, twoT := int64(3)<<curveShift, int64(t)<<1
		smoothed := int32(int64(t) * int64(t) * (threeScaled - twoT) >> (curveShift * 2))
		return vm.PushBits(kind, uint64(uint32(start+((end-start)*smoothed>>curveShift))))
	}
	if shift >= 63 {
		return VMStatusInvalidParameter
	}
	start, end, t := int64(startBits), int64(endBits), int64(tBits)
	maxT := int64(1) << shift
	if t <= 0 {
		return vm.PushBits(kind, uint64(start))
	}
	if t >= maxT {
		return vm.PushBits(kind, uint64(end))
	}
	if operation == BuiltInLerp {
		return vm.PushBits(kind, uint64(start+((end-start)*t>>shift)))
	}
	curveShift := shift
	if curveShift > 15 {
		curveShift = 15
	}
	threeScaled, twoT := int64(3)<<curveShift, t<<1
	smoothed := t * t * (threeScaled - twoT) >> (curveShift * 2)
	return vm.PushBits(kind, uint64(start+((end-start)*smoothed>>curveShift)))
}

func (vm *VM) executeBuiltInPow(kind ValueKind) VMStatus {
	if kind == KindFloat32 {
		exponent, status := vm.PopFloat32()
		if status != VMStatusOK {
			return status
		}
		base, status := vm.PopFloat32()
		if status != VMStatusOK {
			return status
		}
		return vm.PushFloat32(float32(math.Pow(float64(base), float64(exponent))))
	}
	exponent, status := vm.PopFloat64()
	if status != VMStatusOK {
		return status
	}
	base, status := vm.PopFloat64()
	if status != VMStatusOK {
		return status
	}
	return vm.PushFloat64(math.Pow(base, exponent))
}

func (vm *VM) executeBuiltInAbs(kind ValueKind) VMStatus {
	bits, status := vm.PopBits(kind)
	if status != VMStatusOK {
		return status
	}
	switch kind {
	case KindByte, KindUint8, KindUint16, KindUint32, KindUint64:
		return vm.PushBits(kind, bits)
	case KindInt8:
		if bits&0x80 != 0 {
			bits = uint64(uint8(0 - uint8(bits)))
		}
	case KindInt16:
		if bits&0x8000 != 0 {
			bits = uint64(uint16(0 - uint16(bits)))
		}
	case KindInt32:
		if bits&0x80000000 != 0 {
			bits = uint64(uint32(0 - uint32(bits)))
		}
	case KindInt64:
		if bits&0x8000000000000000 != 0 {
			bits = 0 - bits
		}
	case KindFloat32:
		bits = uint64(math.Float32bits(float32(math.Abs(float64(math.Float32frombits(uint32(bits)))))))
	case KindFloat64:
		bits = math.Float64bits(math.Abs(math.Float64frombits(bits)))
	default:
		return VMStatusInvalidValueKind
	}
	return vm.PushBits(kind, bits)
}

func (vm *VM) executeBuiltInMinMax(operation BuiltInOperation, kind ValueKind) VMStatus {
	rightBits, status := vm.PopBits(kind)
	if status != VMStatusOK {
		return status
	}
	leftBits, status := vm.PopBits(kind)
	if status != VMStatusOK {
		return status
	}
	chooseRight := false
	switch kind {
	case KindByte, KindUint8:
		left, right := uint8(leftBits), uint8(rightBits)
		chooseRight = operation == BuiltInMin && right < left || operation == BuiltInMax && right > left
	case KindInt8:
		left, right := int8(leftBits), int8(rightBits)
		chooseRight = operation == BuiltInMin && right < left || operation == BuiltInMax && right > left
	case KindInt16:
		left, right := int16(leftBits), int16(rightBits)
		chooseRight = operation == BuiltInMin && right < left || operation == BuiltInMax && right > left
	case KindInt32:
		left, right := int32(leftBits), int32(rightBits)
		chooseRight = operation == BuiltInMin && right < left || operation == BuiltInMax && right > left
	case KindInt64:
		left, right := int64(leftBits), int64(rightBits)
		chooseRight = operation == BuiltInMin && right < left || operation == BuiltInMax && right > left
	case KindUint16:
		left, right := uint16(leftBits), uint16(rightBits)
		chooseRight = operation == BuiltInMin && right < left || operation == BuiltInMax && right > left
	case KindUint32:
		left, right := uint32(leftBits), uint32(rightBits)
		chooseRight = operation == BuiltInMin && right < left || operation == BuiltInMax && right > left
	case KindUint64:
		chooseRight = operation == BuiltInMin && rightBits < leftBits || operation == BuiltInMax && rightBits > leftBits
	case KindFloat32:
		left, right := math.Float32frombits(uint32(leftBits)), math.Float32frombits(uint32(rightBits))
		chooseRight = operation == BuiltInMin && right < left || operation == BuiltInMax && right > left
	case KindFloat64:
		left, right := math.Float64frombits(leftBits), math.Float64frombits(rightBits)
		chooseRight = operation == BuiltInMin && right < left || operation == BuiltInMax && right > left
	default:
		return VMStatusInvalidValueKind
	}
	if chooseRight {
		return vm.PushBits(kind, rightBits)
	}
	return vm.PushBits(kind, leftBits)
}

func (vm *VM) clearFault() {
	vm.fault = noVMFault()
	vm.instructionPC = 0
	vm.hasInstructionPC = false
}

func (vm *VM) setFault(status VMStatus, target, required, available int) VMStatus {
	pc := -1
	if vm.hasInstructionPC {
		pc = int(vm.instructionPC)
	}
	vm.fault = VMFaultInfo{
		Status:     status,
		PC:         pc,
		Target:     target,
		Required:   required,
		Available:  available,
		HostStatus: VMStatusOK,
	}
	return status
}

func (vm *VM) recordStatus(status VMStatus) VMStatus {
	if status != VMStatusOK && vm.fault.Status == VMStatusOK {
		return vm.setFault(status, -1, -1, -1)
	}
	return status
}

func (vm *VM) currentFrame() (*callFrame, VMStatus) {
	if vm.callFrameTop == 0 {
		return nil, VMStatusInvalidLifecycle
	}
	return &vm.callFrames[int(vm.callFrameTop-1)], VMStatusOK
}

func (vm *VM) enterScriptFunction(functionIndex uint32, returnPC uint32, popArgs bool) VMStatus {
	if functionIndex >= uint32(len(vm.program.Functions)) {
		return vm.setFault(VMStatusInvalidTarget, int(functionIndex), len(vm.program.Functions), len(vm.program.Functions))
	}
	function := vm.program.Functions[int(functionIndex)]
	bodyAddress := function.BodyAddress
	paramStart := function.ParamStart
	paramCount := function.ParamCount
	frameByteSize := function.FrameByteSize
	if bodyAddress >= uint32(len(vm.program.Text)) {
		return vm.setFault(VMStatusInvalidDescriptor, int(functionIndex), len(vm.program.Text), int(bodyAddress))
	}
	if paramStart > uint32(len(vm.program.ParamKinds)) || paramCount > uint32(len(vm.program.ParamKinds))-paramStart ||
		paramStart > uint32(len(vm.program.ParamOffsets)) || paramCount > uint32(len(vm.program.ParamOffsets))-paramStart {
		return vm.setFault(VMStatusInvalidParameter, int(functionIndex), int(paramCount), len(vm.program.ParamKinds))
	}
	var argumentBytes uint32
	for index := uint32(0); index < paramCount; index++ {
		paramIndex := int(paramStart + index)
		kind := vm.program.ParamKinds[paramIndex]
		size := uint32(kind.Size())
		if size == 0 {
			return vm.setFault(VMStatusInvalidParameter, int(index), -1, int(kind))
		}
		offset := vm.program.ParamOffsets[paramIndex]
		if size > frameByteSize || offset > frameByteSize-size {
			return vm.setFault(VMStatusInvalidParameter, int(index), int(frameByteSize), int(offset+size))
		}
		if argumentBytes > ^uint32(0)-size {
			return vm.setFault(VMStatusInvalidParameter, int(index), -1, -1)
		}
		argumentBytes += size
	}
	if popArgs && argumentBytes > uint32(len(vm.memory.segment[segmentStack])) {
		return vm.setFault(VMStatusStackUnderflow, int(functionIndex), int(argumentBytes), len(vm.memory.segment[segmentStack]))
	}
	localBase := vm.frameTop
	frameCapacity := uint32(len(vm.memory.segment[segmentFrame]))
	if localBase > frameCapacity || frameByteSize > frameCapacity-localBase {
		return vm.setFault(VMStatusFrameOverflow, int(functionIndex), int(uint64(localBase)+uint64(frameByteSize)), len(vm.memory.segment[segmentFrame]))
	}
	if vm.callFrameTop >= uint32(len(vm.callFrames)) {
		return vm.setFault(VMStatusCallFrameOverflow, int(functionIndex), int(vm.callFrameTop+1), len(vm.callFrames))
	}
	frameEnd := localBase + frameByteSize
	clear(vm.memory.segment[segmentFrame][int(localBase):int(frameEnd)])
	if popArgs {
		for index := paramCount; index > 0; index-- {
			paramIndex := int(paramStart + index - 1)
			kind := vm.program.ParamKinds[paramIndex]
			if status := vm.memory.segment[segmentStack].TruncateTo(
				&vm.memory.segment[segmentFrame],
				localBase+vm.program.ParamOffsets[paramIndex],
				kind.Size(),
			); status != VMStatusOK {
				return status
			}
		}
	}
	vm.frameTop = frameEnd
	vm.callFrames[int(vm.callFrameTop)] = callFrame{returnPC: returnPC, localBase: localBase, returnKind: function.ReturnKind}
	vm.callFrameTop++
	vm.pc = bodyAddress
	return VMStatusOK
}

func (vm *VM) callScriptFunction(functionIndex uint32) VMStatus {
	return vm.enterScriptFunction(functionIndex, vm.pc, true)
}

func (vm *VM) callExtern(importID uint32) VMStatus {
	if vm.externDispatcher.Dispatcher == nil {
		return vm.setFault(VMStatusMissingExtern, int(importID), -1, -1)
	}
	hostStatus := vm.externDispatcher.Dispatcher(vm.externDispatcher.HostContext, vm, importID)
	if hostStatus != VMStatusOK {
		vm.fault = VMFaultInfo{
			Status:     VMStatusHostFailure,
			PC:         int(vm.instructionPC),
			Target:     int(importID),
			Required:   -1,
			Available:  -1,
			HostStatus: hostStatus,
		}
		return VMStatusHostFailure
	}
	return VMStatusOK
}

func (vm *VM) returnFromFunction() (bool, VMStatus) {
	if vm.callFrameTop == 0 {
		return false, VMStatusInvalidLifecycle
	}
	vm.callFrameTop--
	frame := vm.callFrames[int(vm.callFrameTop)]
	if hasStackValueKind(frame.returnKind) {
		resultSize := frame.returnKind.Size()
		if resultSize == 0 {
			return false, VMStatusInvalidValueKind
		}
		if uint64(len(vm.memory.segment[segmentStack])) < uint64(resultSize) {
			return false, VMStatusStackUnderflow
		}
	}
	vm.frameTop = frame.localBase
	if vm.callFrameTop == 0 {
		return true, VMStatusOK
	}
	vm.pc = frame.returnPC
	return false, VMStatusOK
}

func hasStackValueKind(kind ValueKind) bool {
	return kind != KindNone && kind != KindVoid
}

func (vm *VM) recordStackPush(status VMStatus, size uint32) VMStatus {
	if status == VMStatusStackOverflow {
		return vm.setFault(status, -1, int(uint64(len(vm.memory.segment[segmentStack]))+uint64(size)), cap(vm.memory.segment[segmentStack]))
	}
	return vm.recordStatus(status)
}

func (vm *VM) pushUint8(value uint8) VMStatus {
	return vm.recordStackPush(vm.memory.segment[segmentStack].AppendUint8(value), 1)
}

func (vm *VM) pushUint16(value uint16) VMStatus {
	return vm.recordStackPush(vm.memory.segment[segmentStack].AppendUint16(value), 2)
}

func (vm *VM) pushUint32(value uint32) VMStatus {
	return vm.recordStackPush(vm.memory.segment[segmentStack].AppendUint32(value), 4)
}

func (vm *VM) pushUint64(value uint64) VMStatus {
	return vm.recordStackPush(vm.memory.segment[segmentStack].AppendUint64(value), 8)
}

func (vm *VM) pushFromMemory(address Address, kind ValueKind) VMStatus {
	source, status := vm.memory.segmentForAddress(address)
	if status != VMStatusOK {
		return vm.setFault(status, int(address), -1, -1)
	}
	offset := address.Index()
	switch kind.Size() {
	case 1:
		value, status := source.ReadUint8(offset)
		if status != VMStatusOK {
			return vm.recordStatus(status)
		}
		return vm.pushUint8(value)
	case 2:
		value, status := source.ReadUint16(offset)
		if status != VMStatusOK {
			return vm.recordStatus(status)
		}
		return vm.pushUint16(value)
	case 4:
		value, status := source.ReadUint32(offset)
		if status != VMStatusOK {
			return vm.recordStatus(status)
		}
		return vm.pushUint32(value)
	case 8:
		value, status := source.ReadUint64(offset)
		if status != VMStatusOK {
			return vm.recordStatus(status)
		}
		return vm.pushUint64(value)
	default:
		return vm.recordStatus(VMStatusInvalidValueKind)
	}
}

func (vm *VM) popToMemory(address Address, kind ValueKind) VMStatus {
	if address.Segment() == segmentConst {
		return VMStatusReadOnlyMemory
	}
	destination, status := vm.memory.segmentForAddress(address)
	if status != VMStatusOK {
		return status
	}
	offset := address.Index()
	size := kind.Size()
	if size == 0 {
		return VMStatusInvalidValueKind
	}
	if uint64(offset)+uint64(size) > uint64(len(*destination)) {
		return VMStatusInvalidAddress
	}
	switch size {
	case 1:
		value, status := vm.popUint8()
		if status != VMStatusOK {
			return status
		}
		return destination.WriteUint8(offset, value)
	case 2:
		value, status := vm.popUint16()
		if status != VMStatusOK {
			return status
		}
		return destination.WriteUint16(offset, value)
	case 4:
		value, status := vm.popUint32()
		if status != VMStatusOK {
			return status
		}
		return destination.WriteUint32(offset, value)
	case 8:
		value, status := vm.popUint64()
		if status != VMStatusOK {
			return status
		}
		return destination.WriteUint64(offset, value)
	default:
		return VMStatusInvalidValueKind
	}
}

func (vm *VM) popUint8() (uint8, VMStatus) {
	value, status := vm.memory.segment[segmentStack].TruncateUint8()
	return value, vm.recordStatus(status)
}

func (vm *VM) popUint16() (uint16, VMStatus) {
	value, status := vm.memory.segment[segmentStack].TruncateUint16()
	return value, vm.recordStatus(status)
}

func (vm *VM) popUint32() (uint32, VMStatus) {
	value, status := vm.memory.segment[segmentStack].TruncateUint32()
	return value, vm.recordStatus(status)
}

func (vm *VM) popUint64() (uint64, VMStatus) {
	value, status := vm.memory.segment[segmentStack].TruncateUint64()
	return value, vm.recordStatus(status)
}

func (vm *VM) pushKind(kind ValueKind, bits uint64) VMStatus {
	status := appendStackBits(&vm.memory.segment[segmentStack], kind, bits)
	if status == VMStatusStackOverflow {
		return vm.setFault(status, -1, int(uint64(len(vm.memory.segment[segmentStack]))+uint64(kind.Size())), cap(vm.memory.segment[segmentStack]))
	}
	return vm.recordStatus(status)
}

func (vm *VM) pushAddress(address Address) VMStatus {
	return vm.pushUint32(uint32(address))
}

func (vm *VM) popAddress() (Address, VMStatus) {
	value, status := vm.popUint32()
	return Address(value), status
}

func (vm *VM) popKind(kind ValueKind) (uint64, VMStatus) {
	bits, status := truncateStackBits(&vm.memory.segment[segmentStack], kind)
	if status != VMStatusOK {
		return 0, vm.recordStatus(status)
	}
	return bits, VMStatusOK
}

func appendStackBits(stack *MemorySegment, kind ValueKind, bits uint64) VMStatus {
	return stack.AppendBits(kind, bits)
}

func truncateStackBits(stack *MemorySegment, kind ValueKind) (uint64, VMStatus) {
	return stack.TruncateBits(kind)
}

type vmInteger interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~int8 | ~int16 | ~int32 | ~int64
}

type vmFloat interface {
	~float32 | ~float64
}

type vmOrdered interface {
	vmInteger | vmFloat
}

type vmNumber interface {
	vmOrdered
}

func executeIntegerArithmetic[T vmInteger](pop func() (T, VMStatus), push func(T) VMStatus, op ArithmeticOp) VMStatus {
	right, status := pop()
	if status != VMStatusOK {
		return status
	}
	left, status := pop()
	if status != VMStatusOK {
		return status
	}
	var result T
	switch op {
	case ArithmeticAdd:
		result = left + right
	case ArithmeticSub:
		result = left - right
	case ArithmeticMul:
		result = left * right
	case ArithmeticDiv:
		if right == 0 {
			return VMStatusDivisionByZero
		}
		result = left / right
	case ArithmeticModulo:
		if right == 0 {
			return VMStatusDivisionByZero
		}
		result = left % right
	case ArithmeticBitwiseAnd:
		result = left & right
	case ArithmeticBitwiseOr:
		result = left | right
	case ArithmeticBitwiseXor:
		result = left ^ right
	case ArithmeticShiftLeft:
		result = left << (uint64(right) & (uint64(unsafe.Sizeof(left))*8 - 1))
	case ArithmeticShiftRight:
		result = left >> (uint64(right) & (uint64(unsafe.Sizeof(left))*8 - 1))
	default:
		return VMStatusInvalidOpcode
	}
	return push(result)
}

func executeFloatArithmetic[T vmFloat](pop func() (T, VMStatus), push func(T) VMStatus, op ArithmeticOp) VMStatus {
	right, status := pop()
	if status != VMStatusOK {
		return status
	}
	left, status := pop()
	if status != VMStatusOK {
		return status
	}
	var result T
	switch op {
	case ArithmeticAdd:
		result = left + right
	case ArithmeticSub:
		result = left - right
	case ArithmeticMul:
		result = left * right
	case ArithmeticDiv:
		if right == 0 {
			return VMStatusDivisionByZero
		}
		result = left / right
	default:
		return VMStatusInvalidOpcode
	}
	return push(result)
}

func (vm *VM) executeArithmetic(kind ValueKind, op ArithmeticOp) VMStatus {
	switch kind {
	case KindBool, KindByte, KindUint8:
		return executeIntegerArithmetic(vm.popUint8, vm.pushUint8, op)
	case KindUint16:
		return executeIntegerArithmetic(vm.PopUint16, vm.PushUint16, op)
	case KindUint32:
		return executeIntegerArithmetic(vm.PopUint32, vm.PushUint32, op)
	case KindUint64:
		return executeIntegerArithmetic(vm.PopUint64, vm.PushUint64, op)
	case KindInt8:
		return executeIntegerArithmetic(vm.PopInt8, vm.PushInt8, op)
	case KindInt16:
		return executeIntegerArithmetic(vm.PopInt16, vm.PushInt16, op)
	case KindInt64:
		return executeIntegerArithmetic(vm.PopInt64, vm.PushInt64, op)
	case KindInt32:
		return executeIntegerArithmetic(vm.PopInt32, vm.PushInt32, op)
	case KindFloat32:
		return executeFloatArithmetic(vm.PopFloat32, vm.PushFloat32, op)
	case KindFloat64:
		return executeFloatArithmetic(vm.PopFloat64, vm.PushFloat64, op)
	}
	return VMStatusInvalidOpcode
}

func compareOrdered[T vmOrdered](left T, right T, op CompareOp) (bool, VMStatus) {
	switch op {
	case CompareEqual:
		return left == right, VMStatusOK
	case CompareNotEqual:
		return left != right, VMStatusOK
	case CompareLess:
		return left < right, VMStatusOK
	case CompareLessEqual:
		return left <= right, VMStatusOK
	case CompareGreater:
		return left > right, VMStatusOK
	case CompareGreaterEqual:
		return left >= right, VMStatusOK
	default:
		return false, VMStatusInvalidOpcode
	}
}

func popAndCompare[T vmOrdered](pop func() (T, VMStatus), op CompareOp) (bool, VMStatus) {
	right, status := pop()
	if status != VMStatusOK {
		return false, status
	}
	left, status := pop()
	if status != VMStatusOK {
		return false, status
	}
	return compareOrdered(left, right, op)
}

func (vm *VM) executeTypedComparison(kind ValueKind, op CompareOp) (bool, VMStatus) {
	switch kind {
	case KindBool:
		right, status := vm.PopBool()
		if status != VMStatusOK {
			return false, status
		}
		left, status := vm.PopBool()
		if status != VMStatusOK {
			return false, status
		}
		switch op {
		case CompareEqual:
			return left == right, VMStatusOK
		case CompareNotEqual:
			return left != right, VMStatusOK
		default:
			return false, VMStatusInvalidOpcode
		}
	case KindByte, KindUint8:
		return popAndCompare(vm.PopUint8, op)
	case KindUint16:
		return popAndCompare(vm.PopUint16, op)
	case KindUint32:
		return popAndCompare(vm.PopUint32, op)
	case KindUint64:
		return popAndCompare(vm.PopUint64, op)
	case KindInt8:
		return popAndCompare(vm.PopInt8, op)
	case KindInt16:
		return popAndCompare(vm.PopInt16, op)
	case KindInt32:
		return popAndCompare(vm.PopInt32, op)
	case KindInt64:
		return popAndCompare(vm.PopInt64, op)
	case KindFloat32:
		return popAndCompare(vm.PopFloat32, op)
	case KindFloat64:
		return popAndCompare(vm.PopFloat64, op)
	case KindAddress:
		right, status := vm.popAddress()
		if status != VMStatusOK {
			return false, status
		}
		left, status := vm.popAddress()
		if status != VMStatusOK {
			return false, status
		}
		return compareOrdered(uint32(left), uint32(right), op)
	default:
		return false, VMStatusInvalidValueKind
	}
}

func convertAndPush[T vmNumber](vm *VM, value T, to ValueKind) VMStatus {
	switch to {
	case KindBool:
		return vm.PushBool(value != 0)
	case KindByte:
		return vm.PushByte(byte(value))
	case KindInt8:
		return vm.PushInt8(int8(value))
	case KindInt16:
		return vm.PushInt16(int16(value))
	case KindInt32:
		return vm.PushInt32(int32(value))
	case KindInt64:
		return vm.PushInt64(int64(value))
	case KindUint8:
		return vm.PushUint8(uint8(value))
	case KindUint16:
		return vm.PushUint16(uint16(value))
	case KindUint32:
		return vm.PushUint32(uint32(value))
	case KindUint64:
		return vm.PushUint64(uint64(value))
	case KindFloat32:
		return vm.PushFloat32(float32(value))
	case KindFloat64:
		return vm.PushFloat64(float64(value))
	case KindAddress:
		return vm.pushAddress(Address(uint32(value)))
	default:
		return VMStatusInvalidValueKind
	}
}

func (vm *VM) executeConversion(from ValueKind, to ValueKind) VMStatus {
	switch from {
	case KindBool:
		value, status := vm.PopBool()
		if status != VMStatusOK {
			return status
		}
		if value {
			return convertAndPush(vm, uint8(1), to)
		}
		return convertAndPush(vm, uint8(0), to)
	case KindByte, KindUint8:
		value, status := vm.PopUint8()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindInt8:
		value, status := vm.PopInt8()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindInt16:
		value, status := vm.PopInt16()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindInt32:
		value, status := vm.PopInt32()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindInt64:
		value, status := vm.PopInt64()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindUint16:
		value, status := vm.PopUint16()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindUint32:
		value, status := vm.PopUint32()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindUint64:
		value, status := vm.PopUint64()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindFloat32:
		value, status := vm.PopFloat32()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindFloat64:
		value, status := vm.PopFloat64()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, value, to)
	case KindAddress:
		value, status := vm.popAddress()
		if status != VMStatusOK {
			return status
		}
		return convertAndPush(vm, uint32(value), to)
	default:
		return VMStatusInvalidValueKind
	}
}
