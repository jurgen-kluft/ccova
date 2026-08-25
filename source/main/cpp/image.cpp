#include "ccore/c_debug.h"
#include "ccova/image.h"

namespace ncore
{
    static void static_validate()
    {
        ASSERTS(sizeof(script_function_t) == ProgramImageFunctionSize, "script function must match the image ABI");
        ASSERTS(CC_OFFSETOF(script_function_t, m_body_address) == 0, "script function body address ABI mismatch");
        ASSERTS(CC_OFFSETOF(script_function_t, m_param_start) == 4, "script function parameter start ABI mismatch");
        ASSERTS(CC_OFFSETOF(script_function_t, m_param_count) == 8, "script function parameter count ABI mismatch");
        ASSERTS(CC_OFFSETOF(script_function_t, m_frame_byte_size) == 12, "script function frame size ABI mismatch");
        ASSERTS(CC_OFFSETOF(script_function_t, m_return_kind) == 16, "script function return kind ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_magic) == 0, "program magic ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_entry_point) == 8, "program entry point ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_bss_byte_size) == 12, "program BSS size ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_frame_size) == 16, "program frame size ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_frame_byte_size) == 20, "program frame byte size ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_functions) == 24, "program functions ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_param_kinds) == 32, "program parameter kinds ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_param_offsets) == 40, "program parameter offsets ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_text) == 48, "program text ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_const_data) == 56, "program const data ABI mismatch");
        ASSERTS(CC_OFFSETOF(linked_program_t, m_data_data) == 64, "program data ABI mismatch");
    }

    const linked_program_t* open_program_image(const byte* block, u32 block_size)
    {
        ASSERT(block != nullptr);
        ASSERT(block_size >= ProgramImageHeaderSize);
        ASSERT(((uint_t)block & 3U) == 0);

        static_validate();

        const linked_program_t* program = (const linked_program_t*)block;
        validate_linked_program(program, block_size);
        return program;
    }

} // namespace ncore
