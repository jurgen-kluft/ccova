#include "ccova/types.h"

namespace ncore
{
    // Layout of the instruction_t type:
    // 15 14 13 12 11 10 9 8 7 6 5 4 3 2 1 0
    // |  opcode  |   kind   |     operation     |
    // opcode: 5 bits (0-31)
    // kind: 4 bits (0-15)
    // operation: 6 bits (0-63)

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

    address_t make_address(ememorysegment_t segment, u32 index)
    {
        ASSERT(segment > SegmentInvalid && segment < SegmentCount);
        ASSERT(index <= AddressIndexMask);
        return ((u32)segment << 24) | index;
    }

} // namespace ncore
