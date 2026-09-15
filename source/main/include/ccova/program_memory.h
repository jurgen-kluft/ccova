#ifndef __CCOVA_PROGRAM_MEMORY_H__
#define __CCOVA_PROGRAM_MEMORY_H__

#include "ccova/segment_memory.h"
#include "ccova/float_bits.h"

namespace ncore
{
    struct program_memory_t
    { segment_memory_t m_segments[SegmentCount]; };

    void initialize_program_memory(program_memory_t* memory, const segment_memory_t& frame, const segment_memory_t& bss, const segment_memory_t& external, const segment_memory_t& constant, const segment_memory_t& data, const segment_memory_t& stack);

    inline bool is_active_segment(ememorysegment_t segment) { return segment >= SegmentFrame && segment <= SegmentStack; }

    inline const segment_memory_t* segment_for_address(const program_memory_t* memory, address_t address)
    {
        ASSERT(memory != nullptr);
        const ememorysegment_t segment = address_segment(address);
        ASSERT(is_active_segment(segment));
        return &memory->m_segments[(u32)segment];
    }

    inline segment_memory_t* writable_segment_for_address(program_memory_t* memory, address_t address)
    {
        ASSERT(memory != nullptr);
        const ememorysegment_t segment = address_segment(address);
        ASSERT(is_active_segment(segment));
        ASSERT(segment != SegmentConst);
        return &memory->m_segments[(u32)segment];
    }

    inline u8        read_u8(const program_memory_t* memory, address_t address) { return read_u8(segment_for_address(memory, address), address_index(address)); }
    inline u16       read_u16(const program_memory_t* memory, address_t address) { return read_u16(segment_for_address(memory, address), address_index(address)); }
    inline u32       read_u32(const program_memory_t* memory, address_t address) { return read_u32(segment_for_address(memory, address), address_index(address)); }
    inline u64       read_u64(const program_memory_t* memory, address_t address) { return read_u64(segment_for_address(memory, address), address_index(address)); }
    inline bool      read_bool(const program_memory_t* memory, address_t address) { return read_u8(memory, address) != 0; }
    inline s8        read_s8(const program_memory_t* memory, address_t address) { return (s8)read_u8(memory, address); }
    inline s16       read_s16(const program_memory_t* memory, address_t address) { return (s16)read_u16(memory, address); }
    inline s32       read_s32(const program_memory_t* memory, address_t address) { return (s32)read_u32(memory, address); }
    inline s64       read_s64(const program_memory_t* memory, address_t address) { return (s64)read_u64(memory, address); }
    inline f32       read_f32(const program_memory_t* memory, address_t address) { return bits_to_f32(read_u32(memory, address)); }
    inline f64       read_f64(const program_memory_t* memory, address_t address) { return bits_to_f64(read_u64(memory, address)); }
    inline address_t read_address(const program_memory_t* memory, address_t address) { return read_u32(memory, address); }

    inline void write_u8(program_memory_t* memory, address_t address, u8 value) { write_u8(writable_segment_for_address(memory, address), address_index(address), value); }
    inline void write_u16(program_memory_t* memory, address_t address, u16 value) { write_u16(writable_segment_for_address(memory, address), address_index(address), value); }
    inline void write_u32(program_memory_t* memory, address_t address, u32 value) { write_u32(writable_segment_for_address(memory, address), address_index(address), value); }
    inline void write_u64(program_memory_t* memory, address_t address, u64 value) { write_u64(writable_segment_for_address(memory, address), address_index(address), value); }
    inline void write_bool(program_memory_t* memory, address_t address, bool value) { write_u8(memory, address, value ? 1 : 0); }
    inline void write_s8(program_memory_t* memory, address_t address, s8 value) { write_u8(memory, address, (u8)value); }
    inline void write_s16(program_memory_t* memory, address_t address, s16 value) { write_u16(memory, address, (u16)value); }
    inline void write_s32(program_memory_t* memory, address_t address, s32 value) { write_u32(memory, address, (u32)value); }
    inline void write_s64(program_memory_t* memory, address_t address, s64 value) { write_u64(memory, address, (u64)value); }
    inline void write_f32(program_memory_t* memory, address_t address, f32 value) { write_u32(memory, address, f32_to_bits(value)); }
    inline void write_f64(program_memory_t* memory, address_t address, f64 value) { write_u64(memory, address, f64_to_bits(value)); }
    inline void write_address(program_memory_t* memory, address_t destination, address_t value) { write_u32(memory, destination, value); }

} // namespace ncore

#endif