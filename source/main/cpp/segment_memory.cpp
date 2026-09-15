#include "ccova/segment_memory.h"
#include "ccova/byte_order.h"

namespace ncore
{
    bool is_segment_valid(const segment_memory_t* memory)
    {
        if (memory == nullptr)
            return false;
        if (memory->m_size > memory->m_capacity)
            return false;
        if (memory->m_data == nullptr && memory->m_capacity != 0)
            return false;
        return true;
    }

    bool is_read_valid(const segment_memory_t* memory, u32 offset, u32 size)
    {
        if (memory == nullptr)
            return false;
        if (offset > memory->m_size)
            return false;
        if (size > memory->m_size - offset)
            return false;
        return true;
    }

    static u32 grow_segment(segment_memory_t* memory, u32 size)
    {
        ASSERT(is_segment_valid(memory));
        ASSERT(size <= memory->m_capacity - memory->m_size);
        const u32 offset = memory->m_size;
        memory->m_size += size;
        return offset;
    }

    static void move_bytes(byte* destination, const byte* source, u32 size)
    {
        ASSERT(destination != nullptr || size == 0);
        ASSERT(source != nullptr || size == 0);
        if (destination == source || size == 0)
            return;
        if (destination < source || destination >= source + size)
        {
            for (u32 index = 0; index < size; ++index)
                destination[index] = source[index];
        }
        else
        {
            for (u32 index = size; index > 0; --index)
                destination[index - 1] = source[index - 1];
        }
    }

    void append_u8(segment_memory_t* memory, u8 value)
    {
        const u32 offset       = grow_segment(memory, sizeof(value));
        memory->m_data[offset] = value;
    }
    
    void append_u16(segment_memory_t* memory, u16 value)
    {
        const u32 offset = grow_segment(memory, sizeof(value));
        write_le_u16(memory->m_data + offset, value);
    }
    
    void append_u32(segment_memory_t* memory, u32 value)
    {
        const u32 offset = grow_segment(memory, sizeof(value));
        write_le_u32(memory->m_data + offset, value);
    }
    
    void append_u64(segment_memory_t* memory, u64 value)
    {
        const u32 offset = grow_segment(memory, sizeof(value));
        write_le_u64(memory->m_data + offset, value);
    }

    void append_bits32(segment_memory_t* memory, evaluekind_t kind, u32 bits)
    {
        ASSERT(value_kind_is_32_bit(kind));
        switch (value_kind_size(kind))
        {
            case 1: append_u8(memory, (u8)bits); break;
            case 2: append_u16(memory, (u16)bits); break;
            case 4: append_u32(memory, bits); break;
            default: ASSERT(false); break;
        }
    }

    void append_bits64(segment_memory_t* memory, evaluekind_t kind, u64 bits)
    {
        ASSERT(value_kind_is_64_bit(kind));
        append_u64(memory, bits);
    }

    void append_from(segment_memory_t* memory, const segment_memory_t* source, u32 offset, u32 size)
    {
        ASSERT(size != 0);
        ASSERT(is_read_valid(source, offset, size));
        const u32 destination_offset = grow_segment(memory, size);
        move_bytes(memory->m_data + destination_offset, source->m_data + offset, size);
    }

    u8 truncate_u8(segment_memory_t* memory)
    {
        const u32 offset = shrink_offset(memory, sizeof(u8));
        const u8  value  = memory->m_data[offset];
        memory->m_size   = offset;
        return value;
    }
    u16 truncate_u16(segment_memory_t* memory)
    {
        const u32 offset = shrink_offset(memory, sizeof(u16));
        const u16 value  = read_le_u16(memory->m_data + offset);
        memory->m_size   = offset;
        return value;
    }
    u32 truncate_u32(segment_memory_t* memory)
    {
        const u32 offset = shrink_offset(memory, sizeof(u32));
        const u32 value  = read_le_u32(memory->m_data + offset);
        memory->m_size   = offset;
        return value;
    }
    u64 truncate_u64(segment_memory_t* memory)
    {
        const u32 offset = shrink_offset(memory, sizeof(u64));
        const u64 value  = read_le_u64(memory->m_data + offset);
        memory->m_size   = offset;
        return value;
    }

    u32 truncate_bits32(segment_memory_t* memory, evaluekind_t kind)
    {
        ASSERT(value_kind_is_32_bit(kind));
        switch (value_kind_size(kind))
        {
            case 1: return truncate_u8(memory);
            case 2: return truncate_u16(memory);
            case 4: return truncate_u32(memory);
            default: ASSERT(false); return 0;
        }
    }

    u64 truncate_bits64(segment_memory_t* memory, evaluekind_t kind)
    {
        ASSERT(value_kind_is_64_bit(kind));
        return truncate_u64(memory);
    }

    void truncate_to(segment_memory_t* memory, segment_memory_t* destination, u32 offset, u32 size)
    {
        ASSERT(size != 0);
        ASSERT(is_read_valid(destination, offset, size));
        const u32 source_offset = shrink_offset(memory, size);
        move_bytes(destination->m_data + offset, memory->m_data + source_offset, size);
        memory->m_size = source_offset;
    }
} // namespace ncore
