#include "ccova/code_memory.h"
#include "ccova/byte_order.h"

namespace ncore
{
    static inline bool s_valid_code_range(const code_memory_t* memory, u32* offset, u32 size)
    {
        if (offset == nullptr || memory == nullptr)
            return false;
        if (memory->m_code == nullptr && memory->m_size != 0)
            return false;
        if (*offset > memory->m_size)
            return false;
        if (size > memory->m_size - *offset)
            return false;
        return true;
    }

    instruction_t read_instruction(const code_memory_t* memory, u32* offset)
    {
        ASSERT(s_valid_code_range(memory, offset, 2));
        const instruction_t instruction = read_le_u16(memory->m_code + *offset);
        *offset += 2;
        return instruction;
    }

    u32 read_immediate32(const code_memory_t* memory, u32* offset, evaluekind_t kind)
    {
        ASSERT(value_kind_is_32_bit(kind));
        const u32 size = value_kind_size(kind);
        ASSERT(s_valid_code_range(memory, offset, size));
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

    u64 read_immediate64(const code_memory_t* memory, u32* offset, evaluekind_t kind)
    {
        ASSERT(s_valid_code_range(memory, offset, 8));
        ASSERT(value_kind_is_64_bit(kind));
        const u64 value = read_le_u64(memory->m_code + *offset);
        *offset += 8;
        return value;
    }

    u32 read_u32(const code_memory_t* memory, u32* offset)
    {
        ASSERT(s_valid_code_range(memory, offset, 4));
        const u32 value = read_le_u32(memory->m_code + *offset);
        *offset += 4;
        return value;
    }
} // namespace ncore