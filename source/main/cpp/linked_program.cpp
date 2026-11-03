#include "ccova/linked_program.h"

namespace ncore
{

    template <typename T>
    static bool validate_array(const linked_program_t* program, u32 block_size, const relative_array_t<T>& array)
    {
        bool valid = (program != nullptr);
        if (array.m_size == 0)
        {
            valid = valid && (array.m_data.m_offset == 0);
            return valid;
        }

        valid                    = valid && (array.m_data.m_offset != 0);
        const byte* block        = (const byte*)program;
        const s64   field_offset = (const byte*)&array.m_data - block;
        const s64   data_offset  = field_offset + array.m_data.m_offset;
        valid                    = valid && (data_offset >= 0);
        valid                    = valid && (data_offset <= block_size);
        valid                    = valid && ((data_offset & (alignof(T) - 1)) == 0);
        valid                    = valid && (array.m_size <= (block_size - (u32)data_offset) / sizeof(T));
        valid                    = valid && (array.m_data.get() == (const T*)(block + data_offset));
        (void)data_offset;
        return valid;
    }

    static bool validate_program_contents(const linked_program_t* program)
    {
        bool valid = true;
        valid      = valid && validate_array(program, sizeof(*program), program->m_functions);
        valid      = valid && validate_array(program, sizeof(*program), program->m_param_kinds);
        valid      = valid && validate_array(program, sizeof(*program), program->m_param_offsets);
        valid      = valid && validate_array(program, sizeof(*program), program->m_text);
        valid      = valid && validate_array(program, sizeof(*program), program->m_const_data);
        valid      = valid && validate_array(program, sizeof(*program), program->m_data_data);

        valid = valid && (program->m_param_kinds.m_size == program->m_param_offsets.m_size);
        valid = valid && (program->m_functions.m_size != 0);
        valid = valid && (program->m_entry_point < program->m_functions.m_size);

        for (u32 index = 0; valid && index < program->m_param_kinds.m_size; ++index)
        {
            const evaluekind_t kind = program->m_param_kinds[index];
            valid                   = valid && (kind > KindVoid && kind < KindCount);
            (void)kind;
        }

        for (u32 index = 0; valid && index < program->m_functions.m_size; ++index)
        {
            const script_function_t& function = program->m_functions[index];
            valid                             = valid && (function.m_body_address < program->m_text.m_size);
            valid                             = valid && (function.m_return_kind > KindNone && function.m_return_kind < KindCount);
            valid                             = valid && (function.m_param_start <= program->m_param_kinds.m_size);
            valid                             = valid && (function.m_param_count <= program->m_param_kinds.m_size - function.m_param_start);
            valid                             = valid && (function.m_param_start + function.m_param_count <= program->m_frame_size);
            valid                             = valid && (function.m_frame_byte_size <= program->m_frame_byte_size);
            (void)function;
        }
        return valid;
    }

    bool validate_linked_program(const linked_program_t* program, u32 block_size)
    {
        bool valid = (program != nullptr);
        valid      = valid && (program->m_magic == ProgramImageMagic);
        valid      = valid && (program->m_version == ProgramImageVersion);
        valid      = valid && (program->m_endian == ProgramImageEndianLittle);
        valid      = valid && (program->m_abi == ProgramImageABI);

        valid = valid && (block_size >= sizeof(linked_program_t));
        valid = valid && validate_program_contents(program);
        valid = valid && validate_array(program, block_size, program->m_functions);
        valid = valid && validate_array(program, block_size, program->m_param_kinds);
        valid = valid && validate_array(program, block_size, program->m_param_offsets);
        valid = valid && validate_array(program, block_size, program->m_text);
        valid = valid && validate_array(program, block_size, program->m_const_data);
        valid = valid && validate_array(program, block_size, program->m_data_data);
        return valid;

        return valid;
    }

    bool validate_linked_program(const void* ptr, u32 block_size)
    {
        const linked_program_t* program = (const linked_program_t*)ptr;
        return validate_linked_program(program, block_size);
    }

    const linked_program_t* open_program_image(const byte* block, u32 block_size)
    {
        bool valid = (block_size >= ProgramImageHeaderSize);
        valid      = valid && (((uint_t)block & 3U) == 0);
        valid      = valid && validate_linked_program(block, block_size);
        return valid ? (const linked_program_t*)block : nullptr;
    }

}  // namespace ncore
