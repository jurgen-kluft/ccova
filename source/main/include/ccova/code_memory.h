#ifndef __CCOVA_CODE_MEMORY_H__
#define __CCOVA_CODE_MEMORY_H__

#include "ccova/types.h"
#include "ccova/byte_order.h"

namespace ncore
{
    struct code_memory_t
    {
        const byte* m_code;
        u32         m_size;
    };

    bool is_code_range_valid(const code_memory_t* memory, u32* offset, u32 size);

    inline instruction_t read_instruction(const code_memory_t* memory, u32* offset)
    {
        ASSERT(is_code_range_valid(memory, offset, 2));
        const instruction_t instruction = read_le_u16(memory->m_code + *offset);
        *offset += 2;
        return instruction;
    }

    inline u32 read_immediate32(const code_memory_t* memory, u32* offset, evaluekind_t kind)
    {
        ASSERT(value_kind_is_32_bit(kind));
        const u32 size = value_kind_size(kind);
        ASSERT(is_code_range_valid(memory, offset, size));
        const byte* data = memory->m_code + *offset;
        *offset += size;
        switch (size)
        {
            case 1: return data[0];
            case 2: return read_le_u16(data);
            case 4: return read_le_u32(data);
            default: ASSERT(false); return 0;
        }
    }

    inline u64 read_immediate64(const code_memory_t* memory, u32* offset, evaluekind_t kind)
    {
        ASSERT(is_code_range_valid(memory, offset, 8));
        ASSERT(value_kind_is_64_bit(kind));
        const u64 value = read_le_u64(memory->m_code + *offset);
        *offset += 8;
        return value;
    }

    inline u32 read_u32(const code_memory_t* memory, u32* offset)
    {
        ASSERT(is_code_range_valid(memory, offset, 4));
        const u32 value = read_le_u32(memory->m_code + *offset);
        *offset += 4;
        return value;
    }
} // namespace ncore

#endif
