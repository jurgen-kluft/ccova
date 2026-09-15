#ifndef __CCOVA_SEGMENT_MEMORY_H__
#define __CCOVA_SEGMENT_MEMORY_H__

#include "ccova/types.h"
#include "ccova/byte_order.h"

namespace ncore
{
    struct segment_memory_t
    {
        byte* m_data;
        u32   m_size;
        u32   m_capacity;
    };

    // Validation
    bool is_segment_valid(const segment_memory_t* memory);
    bool is_read_valid(const segment_memory_t* memory, u32 offset, u32 size);

    inline u8 read_u8(const segment_memory_t* memory, u32 offset)
    {
        ASSERT(is_read_valid(memory, offset, 1));
        return memory->m_data[offset];
    }
    inline u16 read_u16(const segment_memory_t* memory, u32 offset)
    {
        ASSERT(is_read_valid(memory, offset, 2));
        return read_le_u16(memory->m_data + offset);
    }
    inline u32 read_u32(const segment_memory_t* memory, u32 offset)
    {
        ASSERT(is_read_valid(memory, offset, 4));
        return read_le_u32(memory->m_data + offset);
    }
    inline u64 read_u64(const segment_memory_t* memory, u32 offset)
    {
        ASSERT(is_read_valid(memory, offset, 8));
        return read_le_u64(memory->m_data + offset);
    }

    inline u32 shrink_offset(segment_memory_t* memory, u32 size)
    {
        ASSERT(is_segment_valid(memory));
        ASSERT(size <= memory->m_size);
        return memory->m_size - size;
    }

    inline void write_u8(segment_memory_t* memory, u32 offset, u8 value)
    {
        ASSERT(is_read_valid(memory, offset, 1));
        memory->m_data[offset] = value;
    }
    inline void write_u16(segment_memory_t* memory, u32 offset, u16 value)
    {
        ASSERT(is_read_valid(memory, offset, 2));
        write_le_u16(memory->m_data + offset, value);
    }
    inline void write_u32(segment_memory_t* memory, u32 offset, u32 value)
    {
        ASSERT(is_read_valid(memory, offset, 4));
        write_le_u32(memory->m_data + offset, value);
    }
    inline void write_u64(segment_memory_t* memory, u32 offset, u64 value)
    {
        ASSERT(is_read_valid(memory, offset, 8));
        write_le_u64(memory->m_data + offset, value);
    }

    void append_u8(segment_memory_t* memory, u8 value);
    void append_u16(segment_memory_t* memory, u16 value);
    void append_u32(segment_memory_t* memory, u32 value);
    void append_u64(segment_memory_t* memory, u64 value);
    void append_bits32(segment_memory_t* memory, evaluekind_t kind, u32 bits);
    void append_bits64(segment_memory_t* memory, evaluekind_t kind, u64 bits);
    void append_from(segment_memory_t* memory, const segment_memory_t* source, u32 offset, u32 size);

    u8   truncate_u8(segment_memory_t* memory);
    u16  truncate_u16(segment_memory_t* memory);
    u32  truncate_u32(segment_memory_t* memory);
    u64  truncate_u64(segment_memory_t* memory);
    u32  truncate_bits32(segment_memory_t* memory, evaluekind_t kind);
    u64  truncate_bits64(segment_memory_t* memory, evaluekind_t kind);
    void truncate_to(segment_memory_t* memory, segment_memory_t* destination, u32 offset, u32 size);
} // namespace ncore

#endif
