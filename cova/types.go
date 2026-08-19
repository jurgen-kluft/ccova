package cova

import "fmt"

type TypeKind uint8

const (
	TypeInvalid TypeKind = iota
	TypeVoid
	TypeBool
	TypeByte
	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt64
	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint64
	TypeFloat32
	TypeFloat64
	TypePointer
	TypeString
	TypeChar
	TypeArray
	TypeStruct
)

type StructField struct {
	Name       string
	Type       *Type
	ByteOffset int
}

type StructType struct {
	Name         string
	Fields       []StructField
	FieldsByName map[string]int
	Size         int
	Alignment    int
}

type Type struct {
	Kind         TypeKind
	Name         string
	Size         int
	Base         *Type
	ElementCount int
	Struct       *StructType
	IsConst      bool
}

func (typ *Type) String() string {
	if typ == nil {
		return "<nil>"
	}
	prefix := ""
	if typ.IsConst {
		prefix = "const "
	}
	if typ.Kind == TypePointer {
		if typ.Base == nil {
			return prefix + "<invalid>*"
		}
		return typ.Base.String() + "*" + suffixConst(typ.IsConst)
	}
	if typ.Kind == TypeArray {
		return fmt.Sprintf("%s[%d]", typ.Base.String(), typ.ElementCount)
	}
	return prefix + typ.Name
}

func suffixConst(isConst bool) string {
	if isConst {
		return " const"
	}
	return ""
}

func (typ *Type) Alignment() int {
	if typ == nil {
		return 1
	}
	if typ.Kind == TypeVoid {
		return 1
	}
	if typ.Kind == TypePointer || typ.Kind == TypeString {
		return 4
	}
	if typ.Kind == TypeArray && typ.Base != nil {
		return typ.Base.Alignment()
	}
	if typ.Kind == TypeStruct && typ.Struct != nil {
		return typ.Struct.Alignment
	}
	if typ.Size <= 1 {
		return 1
	}
	if typ.Size >= 8 {
		return 8
	}
	return typ.Size
}

var (
	VoidType    = &Type{Kind: TypeVoid, Name: "void", Size: 0}
	BoolType    = &Type{Kind: TypeBool, Name: "bool", Size: 1}
	ByteType    = &Type{Kind: TypeByte, Name: "byte", Size: 1}
	Int8Type    = &Type{Kind: TypeInt8, Name: "int8", Size: 1}
	Int16Type   = &Type{Kind: TypeInt16, Name: "int16", Size: 2}
	Int32Type   = &Type{Kind: TypeInt32, Name: "int32", Size: 4}
	Int64Type   = &Type{Kind: TypeInt64, Name: "int64", Size: 8}
	Uint8Type   = &Type{Kind: TypeUint8, Name: "uint8", Size: 1}
	Uint16Type  = &Type{Kind: TypeUint16, Name: "uint16", Size: 2}
	Uint32Type  = &Type{Kind: TypeUint32, Name: "uint32", Size: 4}
	Uint64Type  = &Type{Kind: TypeUint64, Name: "uint64", Size: 8}
	Float32Type = &Type{Kind: TypeFloat32, Name: "float32", Size: 4}
	Float64Type = &Type{Kind: TypeFloat64, Name: "float64", Size: 8}
	StringType  = &Type{Kind: TypeString, Name: "string", Size: 4, Base: Uint8Type}
	CharType    = Uint8Type
	IntType     = Int32Type
)

var namedTypes = map[string]*Type{
	"void":    VoidType,
	"bool":    BoolType,
	"byte":    ByteType,
	"char":    CharType,
	"int":     IntType,
	"int8":    Int8Type,
	"int16":   Int16Type,
	"int32":   Int32Type,
	"int64":   Int64Type,
	"uint8":   Uint8Type,
	"uint16":  Uint16Type,
	"uint32":  Uint32Type,
	"uint64":  Uint64Type,
	"float":   Float32Type,
	"float32": Float32Type,
	"float64": Float64Type,
	"double":  Float64Type,
	"i8":      Int8Type,
	"i16":     Int16Type,
	"i32":     Int32Type,
	"i64":     Int64Type,
	"u8":      Uint8Type,
	"u16":     Uint16Type,
	"u32":     Uint32Type,
	"u64":     Uint64Type,
	"f32":     Float32Type,
	"f64":     Float64Type,
}

func LookupNamedType(name string) *Type {
	return namedTypes[name]
}

func QualifiedType(base *Type, isConst bool) *Type {
	if base == nil {
		return nil
	}
	if !isConst {
		return base
	}
	clone := *base
	clone.IsConst = true
	return &clone
}

func (typ *Type) IsSignedInteger() bool {
	if typ == nil {
		return false
	}
	switch typ.Kind {
	case TypeInt8, TypeInt16, TypeInt32, TypeInt64:
		return true
	default:
		return false
	}
}

func (typ *Type) IsUnsignedInteger() bool {
	if typ == nil {
		return false
	}
	switch typ.Kind {
	case TypeByte, TypeUint8, TypeUint16, TypeUint32, TypeUint64:
		return true
	default:
		return false
	}
}

func (typ *Type) IsFloat() bool {
	if typ == nil {
		return false
	}
	return typ.Kind == TypeFloat32 || typ.Kind == TypeFloat64
}

func (typ *Type) IsNumeric() bool {
	if typ == nil {
		return false
	}
	return typ.IsSignedInteger() || typ.IsUnsignedInteger() || typ.IsFloat()
}

func PointerTo(base *Type) *Type {
	return PointerToQualified(base, false)
}

func PointerToQualified(base *Type, isConst bool) *Type {
	if base == nil {
		return nil
	}
	return &Type{Kind: TypePointer, Name: base.Name + "*", Size: 4, Base: base, IsConst: isConst}
}

func ArrayOf(elementType *Type, count int) (*Type, error) {
	if elementType == nil || elementType.Kind == TypeVoid {
		return nil, fmt.Errorf("array element type must be complete")
	}
	if count <= 0 {
		return nil, fmt.Errorf("array element count must be positive")
	}
	if elementType.Size > int(^uint(0)>>1)/count {
		return nil, fmt.Errorf("array byte size overflows int")
	}
	byteSize := elementType.Size * count
	if uint64(byteSize) > uint64(addressIndexMask)+1 {
		return nil, fmt.Errorf("array byte size exceeds the VM segment address space")
	}
	return &Type{
		Kind:         TypeArray,
		Name:         fmt.Sprintf("%s[%d]", elementType.Name, count),
		Size:         byteSize,
		Base:         elementType,
		ElementCount: count,
	}, nil
}

func NewStructType(name string, fields []StructField) (*Type, error) {
	if name == "" {
		return nil, fmt.Errorf("struct name cannot be empty")
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("struct %q must declare at least one field", name)
	}
	descriptor := &StructType{
		Name:         name,
		Fields:       make([]StructField, 0, len(fields)),
		FieldsByName: make(map[string]int, len(fields)),
		Alignment:    1,
	}
	offset := 0
	for _, field := range fields {
		if field.Name == "" {
			return nil, fmt.Errorf("struct %q has a field with no name", name)
		}
		if _, exists := descriptor.FieldsByName[field.Name]; exists {
			return nil, fmt.Errorf("struct %q has duplicate field %q", name, field.Name)
		}
		if field.Type == nil || field.Type.Kind == TypeVoid || field.Type.Size <= 0 {
			return nil, fmt.Errorf("struct %q field %q must have a complete type", name, field.Name)
		}
		alignment := field.Type.Alignment()
		if alignment > descriptor.Alignment {
			descriptor.Alignment = alignment
		}
		alignedOffset, ok := alignUpInt(offset, alignment)
		if !ok || field.Type.Size > int(^uint(0)>>1)-alignedOffset {
			return nil, fmt.Errorf("struct %q layout overflows int", name)
		}
		field.ByteOffset = alignedOffset
		descriptor.FieldsByName[field.Name] = len(descriptor.Fields)
		descriptor.Fields = append(descriptor.Fields, field)
		offset = alignedOffset + field.Type.Size
	}
	size, ok := alignUpInt(offset, descriptor.Alignment)
	if !ok {
		return nil, fmt.Errorf("struct %q layout overflows int", name)
	}
	descriptor.Size = size
	if uint64(size) > uint64(addressIndexMask)+1 {
		return nil, fmt.Errorf("struct %q byte size exceeds the VM segment address space", name)
	}
	return &Type{Kind: TypeStruct, Name: name, Size: size, Struct: descriptor}, nil
}

func alignUpInt(value int, alignment int) (int, bool) {
	if alignment <= 1 {
		return value, value >= 0
	}
	if value < 0 || value > int(^uint(0)>>1)-(alignment-1) {
		return 0, false
	}
	return (value + alignment - 1) / alignment * alignment, true
}

func IsSameType(left *Type, right *Type) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.Kind != right.Kind || left.Name != right.Name || left.Size != right.Size || left.ElementCount != right.ElementCount || left.IsConst != right.IsConst {
		return false
	}
	if left.Kind == TypeStruct {
		return left.Struct == right.Struct
	}
	return IsSameType(left.Base, right.Base)
}

func IsTopLevelConst(typ *Type) bool {
	return typ != nil && typ.IsConst
}

func isAggregateType(typ *Type) bool {
	return typ != nil && (typ.Kind == TypeArray || typ.Kind == TypeStruct)
}

var typeToValueKind = map[TypeKind]ValueKind{
	TypeVoid: KindVoid,
	TypeBool: KindBool,
	TypeByte: KindByte,
	TypeChar: KindByte,
	TypeInt8: KindInt8, TypeInt16: KindInt16, TypeInt32: KindInt32, TypeInt64: KindInt64,
	TypeUint8: KindUint8, TypeUint16: KindUint16, TypeUint32: KindUint32, TypeUint64: KindUint64,
	TypeFloat32: KindFloat32, TypeFloat64: KindFloat64,
	TypePointer: KindAddress,
	TypeString:  KindAddress,
}

func valueKindFromType(typ *Type) ValueKind {
	if typ == nil {
		return KindNone
	}
	if kind, ok := typeToValueKind[typ.Kind]; ok {
		return kind
	}
	return KindNone
}

type Opcode byte

// Note: Keep the number of opcodes below 32 to fit in 5 bits of the instruction encoding.
const (
	OpPush Opcode = iota + 1
	OpArithmetic
	OpConvert
	OpAddr
	OpOffset
	OpDereference
	OpAssign
	OpCompare
	OpJumpIfFalse
	OpJump
	OpCall
	OpCallExtern
	OpRet
	OpBuiltIn
	OpcodeCount
)

type BuiltInFunction uint16

const (
	BuiltInInvalid BuiltInFunction = iota
)

type BuiltInOperation byte

const (
	BuiltInOperationInvalid BuiltInOperation = iota
	BuiltInAbs                               // math.abs(a)
	BuiltInSin                               // math.sin(a)
	BuiltInCos                               // math.cos(a)
	BuiltInTan                               // math.tan(a)
	BuiltInAsin                              // math.asin(a)
	BuiltInAcos                              // math.acos(a)
	BuiltInAtan                              // math.atan(a)
	BuiltInPow                               // math.pow(a, b)
	BuiltInSqrt                              // math.sqrt(a)
	BuiltInMin                               // math.min(a,b)
	BuiltInMax                               // math.max(a,b)
	BuiltInMap                               // math.map(value, inMin, inMax, outMin, outMax)
	BuiltInRandom                            // math.random() -> int32
	BuiltInClamp                             // math.clamp(value, min, max)
	BuiltInSmoothStep                        // math.smoothstep(edge0, edge1, x, resolution)
	BuiltInInterpolate                       // math.interpolate(a, b, t)
	BuiltInLerp                              // math.lerp(a, b, t)
	BuiltInSlerp                             // math.slerp(a, b, t)
)

func builtInNumArgs(operation BuiltInOperation) int {
	switch operation {
	case BuiltInAbs, BuiltInSin, BuiltInCos, BuiltInTan, BuiltInAsin, BuiltInAcos, BuiltInAtan, BuiltInSqrt:
		return 1
	case BuiltInPow, BuiltInMin, BuiltInMax:
		return 2
	case BuiltInClamp, BuiltInInterpolate, BuiltInLerp, BuiltInSlerp:
		return 3
	case BuiltInSmoothStep:
		return 4
	case BuiltInMap:
		return 5
	case BuiltInRandom:
		return 0
	default:
		return 0
	}
}

func makeBuiltInFunction(operation BuiltInOperation, kind ValueKind) BuiltInFunction {
	return BuiltInFunction(uint16(operation&0x7f)<<4 | uint16(kind&0x0f))
}

func (function BuiltInFunction) Operation() BuiltInOperation {
	return BuiltInOperation((function >> 4) & 0x7f)
}

func (function BuiltInFunction) Kind() ValueKind {
	return ValueKind(function & 0x0f)
}

type ArithmeticOp byte

// Note: Keep arithmetic operations below 64 to fit in the 6-bit instruction field.
const (
	ArithmeticInvalid ArithmeticOp = iota
	ArithmeticAdd
	ArithmeticSub
	ArithmeticMul
	ArithmeticDiv
	ArithmeticModulo
	ArithmeticBitwiseAnd
	ArithmeticBitwiseOr
	ArithmeticBitwiseXor
	ArithmeticShiftLeft
	ArithmeticShiftRight
)

type CompareOp byte

// Note: Keep the number of compare operations below 8 to fit in 3 bits of the instruction encoding.
const (
	CompareInvalid CompareOp = iota
	CompareEqual
	CompareNotEqual
	CompareLess
	CompareLessEqual
	CompareGreater
	CompareGreaterEqual
)

type ValueKind byte

const (
	KindNone ValueKind = iota
	KindVoid
	KindBool
	KindByte
	KindInt8
	KindInt16
	KindInt32
	KindInt64
	KindUint8
	KindUint16
	KindUint32
	KindUint64
	KindFloat32
	KindFloat64
	KindAddress
	KindCount
)

var valueKindSize = [KindCount]uint32{
	KindNone: 0, KindVoid: 0,
	KindBool: 1, KindByte: 1,
	KindInt8: 1, KindInt16: 2, KindInt32: 4, KindInt64: 8,
	KindUint8: 1, KindUint16: 2, KindUint32: 4, KindUint64: 8,
	KindFloat32: 4, KindFloat64: 8,
	KindAddress: 4, // Assuming a 32-bit address space
}

func (kind ValueKind) Size() uint32 {
	if kind >= KindCount {
		return 0
	}
	size := valueKindSize[kind]
	return size
}

type ScopeKind int

const (
	ScopeInvalid ScopeKind = iota
	ScopeFrame
	ScopeBSS
	ScopeConst
	ScopeData
	ScopeExtern
)

type ExternDispatcher func(hostContext uintptr, vm *VM, importID uint32) VMStatus

type ExternDispatcherBinding struct {
	HostContext uintptr
	Dispatcher  ExternDispatcher
}

type DeclKind int

const (
	DeclVariable DeclKind = iota + 1
	DeclFunction
)

type SymbolBinding struct {
	Name           string
	Kind           DeclKind
	Scope          ScopeKind
	Type           *Type
	SlotIndex      uint32
	ByteOffset     uint32
	ByteSize       uint32
	ByteAlignment  uint32
	ParamCount     uint32
	ParamTypes     []*Type
	ParamOffsets   []uint32
	FrameSlotCount uint32
	FrameByteSize  uint32
	TempFuncID     uint32
	ScriptAddress  uint32
}

type CallPatch struct {
	OperandPos int
	TempFuncID uint32
	Line       int
}

type ScriptFunctionDescriptor struct {
	BodyAddress   uint32
	ParamStart    uint32
	ParamCount    uint32
	FrameByteSize uint32
	ReturnKind    ValueKind
}

type ProgramSymbols struct {
	Symbols       map[string]SymbolBinding
	ExternSymbols []SymbolBinding
	BSSSymbols    []SymbolBinding
	DataSymbols   []SymbolBinding
	ConstSymbols  []SymbolBinding
}

func NewProgramSymbols() *ProgramSymbols {
	return &ProgramSymbols{
		Symbols:       make(map[string]SymbolBinding),
		ExternSymbols: make([]SymbolBinding, 0),
		BSSSymbols:    make([]SymbolBinding, 0),
		DataSymbols:   make([]SymbolBinding, 0),
		ConstSymbols:  make([]SymbolBinding, 0),
	}
}

func CopyProgramSymbols(src *ProgramSymbols) *ProgramSymbols {
	if src == nil {
		return nil
	}
	dst := &ProgramSymbols{
		Symbols:       cloneBindingsMap(src.Symbols),
		ExternSymbols: append([]SymbolBinding(nil), src.ExternSymbols...),
		BSSSymbols:    append([]SymbolBinding(nil), src.BSSSymbols...),
		DataSymbols:   append([]SymbolBinding(nil), src.DataSymbols...),
		ConstSymbols:  append([]SymbolBinding(nil), src.ConstSymbols...),
	}
	return dst
}

type LinkedProgram struct {
	Text          CodeMemory
	EntryPoint    uint32
	Functions     []ScriptFunctionDescriptor
	ParamKinds    []ValueKind
	ParamOffsets  []uint32
	FrameSize     uint32
	FrameByteSize uint32
	ConstByteSize uint32
	ConstData     []byte
	DataByteSize  uint32
	DataData      []byte
	BSSSize       uint32
	BSSByteSize   uint32
	DebugSymbols  *ProgramSymbols
}

type RelocatableProgram struct {
	Text                    CodeMemory
	ProgramSymbols          *ProgramSymbols
	Functions               []SymbolBinding
	CallPatches             []CallPatch
	UsedExternalFunctionIDs []uint32
	EntryFunction           uint32
	FrameSize               uint32
	FrameByteSize           uint32
	ConstByteSize           uint32
	ConstData               []byte
	DataByteSize            uint32
	DataData                []byte
	BSSSize                 uint32
	BSSByteSize             uint32
}

func alignUpU32(offset uint32, alignment uint32) uint32 {
	if alignment <= 1 {
		return offset
	}
	mask := alignment - 1
	return (offset + mask) &^ mask
}

func checkedAlignUpU32(offset uint32, alignment uint32) (uint32, bool) {
	if alignment <= 1 {
		return offset, true
	}
	mask := uint64(alignment - 1)
	aligned := (uint64(offset) + mask) &^ mask
	return uint32(aligned), aligned <= uint64(^uint32(0))
}

func lenU32[S ~[]E, E any](values S) uint32 {
	return uint32(len(values))
}
