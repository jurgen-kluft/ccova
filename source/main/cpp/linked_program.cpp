#include "ccova/image.h"

namespace ncore
{
    static void static_validate_linked_program()
    {
        ASSERTCTS(sizeof(relative_ptr_t<byte>) == 4, "relative pointer ABI must be 32-bit");
        ASSERTCTS(sizeof(relative_array_t<byte>) == 8, "relative array ABI must be 8 bytes");
        ASSERTCTS(sizeof(linked_program_t) == 76, "linked program must match the image root ABI");
        ASSERTCTS(__builtin_offsetof(relative_array_t<byte>, m_data) == 0, "relative array data ABI mismatch");
        ASSERTCTS(__builtin_offsetof(relative_array_t<byte>, m_size) == 4, "relative array size ABI mismatch");
        ASSERTCTS(__builtin_offsetof(linked_program_t, m_functions) == 28, "program functions ABI mismatch");
        ASSERTCTS(__builtin_offsetof(linked_program_t, m_param_kinds) == 36, "program parameter kinds ABI mismatch");
        ASSERTCTS(__builtin_offsetof(linked_program_t, m_param_offsets) == 44, "program parameter offsets ABI mismatch");
        ASSERTCTS(__builtin_offsetof(linked_program_t, m_text) == 52, "program text ABI mismatch");
        ASSERTCTS(__builtin_offsetof(linked_program_t, m_const_data) == 60, "program const data ABI mismatch");
        ASSERTCTS(__builtin_offsetof(linked_program_t, m_data_data) == 68, "program data ABI mismatch");
    }

    template <typename T> static void validate_array(const relative_array_t<T>& array)
    {
        ASSERT(array.m_data.get() != nullptr || array.m_size == 0);
        ASSERT(array.m_data.get() == nullptr || array.m_size != 0);
    }

    template <typename T> static void validate_array(const linked_program_t* program, u32 block_size, const relative_array_t<T>& array)
    {
        ASSERT(program != nullptr);
        if (array.m_size == 0)
        {
            ASSERT(array.m_data.m_offset == 0);
            return;
        }

        ASSERT(array.m_data.m_offset != 0);
        const byte* block        = (const byte*)program;
        const s64   field_offset = (const byte*)&array.m_data - block;
        const s64   data_offset  = field_offset + array.m_data.m_offset;
        ASSERT(data_offset >= 0);
        ASSERT(data_offset <= block_size);
        ASSERT((data_offset & (alignof(T) - 1)) == 0);
        ASSERT(array.m_size <= (block_size - (u32)data_offset) / sizeof(T));
        ASSERT(array.m_data.get() == (const T*)(block + data_offset));
        (void)data_offset;
    }

    static void validate_program_contents(const linked_program_t* program)
    {
        validate_array(program->m_functions);
        validate_array(program->m_param_kinds);
        validate_array(program->m_param_offsets);
        validate_array(program->m_text);
        validate_array(program->m_const_data);
        validate_array(program->m_data_data);
        ASSERT(program->m_param_kinds.m_size == program->m_param_offsets.m_size);
        ASSERT(program->m_functions.m_size != 0);
        ASSERT(program->m_entry_point < program->m_functions.m_size);

        for (u32 index = 0; index < program->m_param_kinds.m_size; ++index)
        {
            const evaluekind_t kind = program->m_param_kinds[index];
            ASSERT(kind > KindVoid && kind < KindCount);
            (void)kind;
        }

        for (u32 index = 0; index < program->m_functions.m_size; ++index)
        {
            const script_function_t& function = program->m_functions[index];
            ASSERT(function.m_body_address < program->m_text.m_size);
            ASSERT(function.m_return_kind > KindNone && function.m_return_kind < KindCount);
            ASSERT(function.m_param_start <= program->m_param_kinds.m_size);
            ASSERT(function.m_param_count <= program->m_param_kinds.m_size - function.m_param_start);
            ASSERT(function.m_param_start + function.m_param_count <= program->m_frame_size);
            ASSERT(function.m_frame_byte_size <= program->m_frame_byte_size);
            (void)function;
        }
    }

    void validate_linked_program(const linked_program_t* program)
    {
        ASSERT(program != nullptr);
        ASSERT(program->m_magic == ProgramImageMagic);
        ASSERT(program->m_version == ProgramImageVersion);
        ASSERT(program->m_endian == ProgramImageEndianLittle);
        ASSERT(program->m_abi == ProgramImageABI);
        static_validate_linked_program();
        validate_program_contents(program);
    }

    void validate_linked_program(const linked_program_t* program, u32 block_size)
    {
        ASSERT(program != nullptr);
        ASSERT(block_size >= sizeof(linked_program_t));
        validate_array(program, block_size, program->m_functions);
        validate_array(program, block_size, program->m_param_kinds);
        validate_array(program, block_size, program->m_param_offsets);
        validate_array(program, block_size, program->m_text);
        validate_array(program, block_size, program->m_const_data);
        validate_array(program, block_size, program->m_data_data);
        validate_linked_program(program);
    }
} // namespace ncore
