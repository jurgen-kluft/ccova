#include "ccova/program_memory.h"

namespace ncore
{
    void initialize_program_memory(program_memory_t* memory, const segment_memory_t& frame, const segment_memory_t& bss, const segment_memory_t& external, const segment_memory_t& constant, const segment_memory_t& data, const segment_memory_t& stack)
    {
        ASSERT(memory != nullptr);
        const segment_memory_t empty = {nullptr, 0, 0};
        for (u32 index = 0; index < (u32)SegmentCount; ++index)
            memory->m_segments[index] = empty;
        memory->m_segments[SegmentFrame]  = frame;
        memory->m_segments[SegmentBSS]    = bss;
        memory->m_segments[SegmentExtern] = external;
        memory->m_segments[SegmentConst]  = constant;
        memory->m_segments[SegmentData]   = data;
        memory->m_segments[SegmentStack]  = stack;
    }
} // namespace ncore