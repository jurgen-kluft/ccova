#ifndef __CCOVA_TYPES_H__
#define __CCOVA_TYPES_H__

#include "ccore/c_debug.h"

namespace ncore
{
    enum eopcode_t : u8
    {
        OpPush = 1,
        OpArithmetic,
        OpConvert,
        OpAddr,
        OpOffset,
        OpDereference,
        OpAssign,
        OpCompare,
        OpJumpIfFalse,
        OpJump,
        OpCall,
        OpCallExtern,
        OpRet,
        OpBuiltIn,
        OpcodeCount,
    };

    typedef u16 builtin_function_t;

    enum ebuiltinoperation_t : u8
    {
        BuiltInOperationInvalid = 0,
        BuiltInAbs,         // math::abs(value)
        BuiltInSin,         // math::sin(value)
        BuiltInCos,         // math::cos(value)
        BuiltInTan,         // math::tan(value)
        BuiltInAsin,        // math::asin(value)
        BuiltInAcos,        // math::acos(value)
        BuiltInAtan,        // math::atan(value)
        BuiltInPow,         // math::pow(base, exponent)
        BuiltInSqrt,        // math::sqrt(value)
        BuiltInMin,         // math::min(a, b)
        BuiltInMax,         // math::max(a, b)
        BuiltInMap,         // math::map(value, inMin, inMax, outMin, outMax)
        BuiltInRandom,      // math::random() -> int32
        BuiltInClamp,       // math::clamp(value, min, max)
        BuiltInSmoothStep,  // math::smoothStep(edge0, edge1, x) or (start, end, t, shift)
        BuiltInLerp,        // math::lerp(a, b, t) or (start, end, t, shift)
        BuiltInSlerp,       // math::slerp(a, b, t)
    };

    enum earithmeticop_t : u8
    {
        ArithmeticInvalid = 0,
        ArithmeticAdd,
        ArithmeticSub,
        ArithmeticMul,
        ArithmeticDiv,
        ArithmeticModulo,
        ArithmeticBitwiseAnd,
        ArithmeticBitwiseOr,
        ArithmeticBitwiseXor,
        ArithmeticShiftLeft,
        ArithmeticShiftRight,
    };

    enum ecompareop_t : u8
    {
        CompareInvalid = 0,
        CompareEqual,
        CompareNotEqual,
        CompareLess,
        CompareLessEqual,
        CompareGreater,
        CompareGreaterEqual,
    };

    enum evaluekind_t : u8
    {
        KindNone = 0,
        KindVoid,
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
        KindAddress,
        KindCount,
    };

    enum ememorysegment_t : u8
    {
        SegmentInvalid = 0,
        SegmentFrame,
        SegmentBSS,
        SegmentExtern,
        SegmentConst,
        SegmentData,
        SegmentStack,
        SegmentReserved0,
        SegmentReserved1,
        SegmentCount,
    };

    typedef u16 instruction_t;
    typedef u32 address_t;

    static const u32 AddressIndexMask = 0x00ffffffU;

    u32 value_kind_size(evaluekind_t kind);
    bool value_kind_is_32_bit(evaluekind_t kind);
    bool value_kind_is_64_bit(evaluekind_t kind);

    instruction_t make_instruction(eopcode_t opcode, evaluekind_t kind);
    instruction_t make_arithmetic_instruction(evaluekind_t kind, earithmeticop_t operation);
    instruction_t make_address_instruction(ememorysegment_t segment);
    instruction_t make_compare_instruction(evaluekind_t kind, ecompareop_t operation);
    instruction_t make_convert_instruction(evaluekind_t from, evaluekind_t to);
    instruction_t make_builtin_instruction(builtin_function_t function);

    builtin_function_t  make_builtin_function(ebuiltinoperation_t operation, evaluekind_t kind);
    ebuiltinoperation_t builtin_function_operation(builtin_function_t function);
    evaluekind_t        builtin_function_kind(builtin_function_t function);

    eopcode_t          instruction_opcode(instruction_t instruction);
    evaluekind_t       instruction_kind(instruction_t instruction);
    earithmeticop_t    instruction_arithmetic_op(instruction_t instruction);
    ememorysegment_t   instruction_address_segment(instruction_t instruction);
    ecompareop_t       instruction_compare_op(instruction_t instruction);
    evaluekind_t       instruction_convert_from_kind(instruction_t instruction);
    builtin_function_t instruction_builtin_function(instruction_t instruction);

    address_t        make_address(ememorysegment_t segment, u32 index);
    ememorysegment_t address_segment(address_t address);
    u32              address_index(address_t address);

    ASSERTCTS(sizeof(instruction_t) == 2, "instruction ABI must be 16-bit");
    ASSERTCTS(sizeof(address_t) == 4, "address ABI must be 32-bit");
    ASSERTCTS(OpcodeCount < 32, "opcode count must fit the instruction encoding");
    ASSERTCTS(KindCount <= 16, "value kind must fit four bits");
} // namespace ncore

#endif
