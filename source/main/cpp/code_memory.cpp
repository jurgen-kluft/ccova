#include "ccova/code_memory.h"
#include "ccova/byte_order.h"

namespace ncore
{
    bool is_code_range_valid(const code_memory_t* memory, u32* offset, u32 size)
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
} // namespace ncore