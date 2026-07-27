#ifndef __CCOVA_LINKED_PROGRAM_H__
#define __CCOVA_LINKED_PROGRAM_H__

#include "ccore/c_debug.h"
#include "ccova/types.h"

namespace ncore
{
    template <typename T> struct relative_ptr_t
    {
        s32 m_offset;

        inline const T* get() const { return m_offset == 0 ? nullptr : (const T*)((const byte*)this + m_offset); }
        inline const T& operator[](u32 index) const { return get()[index]; }

        void set(const T* data)
        {
            if (data == nullptr)
            {
                m_offset = 0;
                return;
            }
            const s64 offset = (s64)data - (s64)this;
            ASSERT(offset >= (-2147483647LL - 1LL) && offset <= 2147483647LL);
            ASSERT(offset != 0);
            m_offset = (s32)offset;
        }
    };

    template <typename T> struct relative_array_t
    {
        relative_ptr_t<T> m_data;
        u32               m_size;

        inline const T* data() const { return m_data.get(); }
        inline const T& operator[](u32 index) const
        {
            ASSERT(index < m_size);
            return m_data[index];
        }

        void set(const T* data, u32 size)
        {
            ASSERT(data != nullptr || size == 0);
            m_size = size;
            m_data.set(data);
        }
    };

    struct script_function_t
    {
        u32          m_body_address;
        u32          m_param_start;
        u32          m_param_count;
        u32          m_frame_byte_size;
        evaluekind_t m_return_kind;
    };

    struct linked_program_t
    {
        u32                                 m_magic;
        u16                                 m_version;
        u8                                  m_endian;
        u8                                  m_abi;
        u32                                 m_entry_point;
        u32                                 m_bss_byte_size;
        u32                                 m_frame_size;
        u32                                 m_frame_byte_size;
        relative_array_t<script_function_t> m_functions;
        relative_array_t<evaluekind_t>      m_param_kinds;
        relative_array_t<u32>               m_param_offsets;
        relative_array_t<byte>              m_text;
        relative_array_t<byte>              m_const_data;
        relative_array_t<byte>              m_data_data;
    };

    void validate_linked_program(const linked_program_t* program);
    void validate_linked_program(const linked_program_t* program, u32 block_size);

    ASSERTCTS(sizeof(relative_ptr_t<byte>) == 4, "relative pointer ABI must be 32-bit");
    ASSERTCTS(sizeof(relative_array_t<byte>) == 8, "relative array ABI must be 8 bytes");
    ASSERTCTS(sizeof(linked_program_t) == 72, "linked program must match the image root ABI");
    static_assert(__builtin_offsetof(relative_array_t<byte>, m_data) == 0, "relative array data ABI mismatch");
    static_assert(__builtin_offsetof(relative_array_t<byte>, m_size) == 4, "relative array size ABI mismatch");
    static_assert(__builtin_offsetof(linked_program_t, m_functions) == 24, "program functions ABI mismatch");
    static_assert(__builtin_offsetof(linked_program_t, m_param_kinds) == 32, "program parameter kinds ABI mismatch");
    static_assert(__builtin_offsetof(linked_program_t, m_param_offsets) == 40, "program parameter offsets ABI mismatch");
    static_assert(__builtin_offsetof(linked_program_t, m_text) == 48, "program text ABI mismatch");
    static_assert(__builtin_offsetof(linked_program_t, m_const_data) == 56, "program const data ABI mismatch");
    static_assert(__builtin_offsetof(linked_program_t, m_data_data) == 64, "program data ABI mismatch");
} // namespace ncore

#endif