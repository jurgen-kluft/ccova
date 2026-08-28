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
        KindNone    = 0,  // size = 0
        KindVoid    = 1,  // size = 0
        KindBool    = 2,  // size = 1
        KindByte    = 3,  // size = 1
        KindInt8    = 4,  // size = 1
        KindUint8   = 5,  // size = 1
        KindInt16   = 6,  // size = 2
        KindUint16  = 7,  // size = 2
        KindInt32   = 8,  // size = 4
        KindUint32  = 9,  // size = 4
        KindFloat32 = 10, // size = 4
        KindAddress = 11, // size = 4
        KindInt64   = 12, // size = 8
        KindUint64  = 13, // size = 8
        KindFloat64 = 14, // size = 8
        KindCount   = 15,
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

    inline u32 value_kind_size(evaluekind_t kind)
    {
        ASSERT((u32)kind < (u32)KindCount);
        switch (kind)
        {
            case KindNone: 
            case KindVoid: return 0;
            case KindBool: 
            case KindByte: 
            case KindInt8: 
            case KindUint8: return 1;
            case KindInt16:
            case KindUint16: return 2;
            case KindInt32: 
            case KindUint32: 
            case KindFloat32: 
            case KindAddress: return 4;
            case KindInt64: 
            case KindUint64: 
            case KindFloat64: return 8;
            default: CC_ASSUME(0); 
        }
        return 0;
    }
    inline bool value_kind_is_32_bit(evaluekind_t kind) { return kind < KindInt64; }
    inline bool value_kind_is_64_bit(evaluekind_t kind) { return kind >= KindInt64; }

    instruction_t make_instruction(eopcode_t opcode, evaluekind_t kind);
    instruction_t make_arithmetic_instruction(evaluekind_t kind, earithmeticop_t operation);
    instruction_t make_address_instruction(ememorysegment_t segment);
    instruction_t make_compare_instruction(evaluekind_t kind, ecompareop_t operation);
    instruction_t make_convert_instruction(evaluekind_t from, evaluekind_t to);

    inline eopcode_t        instruction_opcode(instruction_t instruction) { return (eopcode_t)(instruction & 0x1fU); }
    inline evaluekind_t     instruction_kind(instruction_t instruction) { return (evaluekind_t)((instruction >> 6) & 0x0fU); }
    inline earithmeticop_t  instruction_arithmetic_op(instruction_t instruction) { return (earithmeticop_t)((instruction >> 10) & 0x3fU); }
    inline ememorysegment_t instruction_address_segment(instruction_t instruction) { return (ememorysegment_t)((instruction >> 6) & 0x03ffU); }
    inline ecompareop_t     instruction_compare_op(instruction_t instruction) { return (ecompareop_t)((instruction >> 10) & 0x3fU); }
    inline evaluekind_t     instruction_convert_from_kind(instruction_t instruction) { return (evaluekind_t)((instruction >> 10) & 0x0fU); }

    address_t               make_address(ememorysegment_t segment, u32 index);
    inline ememorysegment_t address_segment(address_t address) { return (ememorysegment_t)((address >> 24) & 0xffU); }
    inline u32              address_index(address_t address) { return address & AddressIndexMask; }

    ASSERTCTS(sizeof(instruction_t) == 2, "instruction ABI must be 16-bit");
    ASSERTCTS(sizeof(address_t) == 4, "address ABI must be 32-bit");
    ASSERTCTS(OpcodeCount < 32, "opcode count must fit the instruction encoding");
    ASSERTCTS(KindCount <= 16, "value kind must fit four bits");
} // namespace ncore

#endif
