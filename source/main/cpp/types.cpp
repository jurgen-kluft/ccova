#include "ccova/types.h"

namespace ncore
{
    static const u8 s_value_kind_sizes[KindCount] = {
        0, 0, 1, 1, 1, 2, 4, 8, 1, 2, 4, 8, 4, 8, 4,
    };

    u32 value_kind_size(evaluekind_t kind)
    {
        ASSERT((u32)kind < (u32)KindCount);
        return s_value_kind_sizes[(u32)kind];
    }

    bool value_kind_is_32_bit(evaluekind_t kind)
    {
        const u32 size = value_kind_size(kind);
        return size > 0 && size <= 4;
    }

    bool value_kind_is_64_bit(evaluekind_t kind) { return value_kind_size(kind) == 8; }

    instruction_t make_instruction(eopcode_t opcode, evaluekind_t kind)
    {
        ASSERT((u32)opcode > 0 && (u32)opcode < (u32)OpcodeCount);
        ASSERT((u32)kind < (u32)KindCount);
        return (instruction_t)(((u16)opcode & 0x1fU) | (((u16)kind & 0x0fU) << 6));
    }

    instruction_t make_arithmetic_instruction(evaluekind_t kind, earithmeticop_t operation)
    {
        ASSERT((u32)kind < (u32)KindCount);
        ASSERT(operation > ArithmeticInvalid && operation <= ArithmeticShiftRight);
        return (instruction_t)((u16)OpArithmetic | (((u16)kind & 0x0fU) << 6) | (((u16)operation & 0x3fU) << 10));
    }

    instruction_t make_address_instruction(ememorysegment_t segment)
    {
        ASSERT(segment > SegmentInvalid && segment < SegmentCount);
        return (instruction_t)((u16)OpAddr | ((u16)segment << 6));
    }

    instruction_t make_compare_instruction(evaluekind_t kind, ecompareop_t operation)
    {
        ASSERT((u32)kind < (u32)KindCount);
        ASSERT(operation > CompareInvalid && operation <= CompareGreaterEqual);
        return (instruction_t)((u16)OpCompare | (((u16)kind & 0x0fU) << 6) | (((u16)operation & 0x3fU) << 10));
    }

    instruction_t make_convert_instruction(evaluekind_t from, evaluekind_t to)
    {
        ASSERT((u32)from < (u32)KindCount);
        ASSERT((u32)to < (u32)KindCount);
        return (instruction_t)((u16)OpConvert | (((u16)to & 0x0fU) << 6) | (((u16)from & 0x0fU) << 10));
    }

    instruction_t make_builtin_instruction(builtin_function_t function)
    {
        ASSERT(function <= 0x07ffU);
        return (instruction_t)((u16)OpBuiltIn | ((function & 0x07ffU) << 5));
    }

    builtin_function_t make_builtin_function(ebuiltinoperation_t operation, evaluekind_t kind)
    {
        ASSERT((u32)operation < 128U);
        ASSERT((u32)kind < (u32)KindCount);
        return (builtin_function_t)(((u16)operation << 4) | (u16)kind);
    }

    ebuiltinoperation_t builtin_function_operation(builtin_function_t function) { return (ebuiltinoperation_t)((function >> 4) & 0x7fU); }
    evaluekind_t builtin_function_kind(builtin_function_t function) { return (evaluekind_t)(function & 0x0fU); }

    eopcode_t        instruction_opcode(instruction_t instruction) { return (eopcode_t)(instruction & 0x1fU); }
    evaluekind_t     instruction_kind(instruction_t instruction) { return (evaluekind_t)((instruction >> 6) & 0x0fU); }
    earithmeticop_t  instruction_arithmetic_op(instruction_t instruction) { return (earithmeticop_t)((instruction >> 10) & 0x3fU); }
    ememorysegment_t instruction_address_segment(instruction_t instruction) { return (ememorysegment_t)((instruction >> 6) & 0x03ffU); }
    ecompareop_t     instruction_compare_op(instruction_t instruction) { return (ecompareop_t)((instruction >> 10) & 0x3fU); }
    evaluekind_t     instruction_convert_from_kind(instruction_t instruction) { return (evaluekind_t)((instruction >> 10) & 0x0fU); }
    builtin_function_t instruction_builtin_function(instruction_t instruction) { return (builtin_function_t)((instruction >> 5) & 0x07ffU); }

    address_t make_address(ememorysegment_t segment, u32 index)
    {
        ASSERT(segment > SegmentInvalid && segment < SegmentCount);
        ASSERT(index <= AddressIndexMask);
        return ((u32)segment << 24) | index;
    }

    ememorysegment_t address_segment(address_t address) { return (ememorysegment_t)((address >> 24) & 0xffU); }
    u32              address_index(address_t address) { return address & AddressIndexMask; }
} // namespace ncore
