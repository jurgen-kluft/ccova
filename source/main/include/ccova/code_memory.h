#ifndef __CCOVA_CODE_MEMORY_H__
#define __CCOVA_CODE_MEMORY_H__

#include "ccova/types.h"

namespace ncore
{
    struct code_memory_t
    {
        const byte* m_code;
        u32         m_size;
    };

    instruction_t read_instruction(const code_memory_t* memory, u32* offset);
    u32           read_immediate32(const code_memory_t* memory, u32* offset, evaluekind_t kind);
    u64           read_immediate64(const code_memory_t* memory, u32* offset, evaluekind_t kind);
    u32           read_u32(const code_memory_t* memory, u32* offset);
} // namespace ncore

#endif
