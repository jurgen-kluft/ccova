#include "ccore/c_target.h"

#include "ccova/byte_order.h"
#include "ccova/image.h"
#include "ccova/vm.h"

#include "cunittest/cunittest.h"

using namespace ncore;

namespace
{
    static void write_array_header(byte* block, u32 header_offset, u32 count, u32 data_offset)
    {
        const s32 relative_offset = count == 0 ? 0 : (s32)data_offset - (s32)header_offset;
        write_le_u32(block + header_offset, (u32)relative_offset);
        write_le_u32(block + header_offset + 4, count);
    }

    static void write_function(byte* block, u32 offset, u32 body_address, u32 param_start, u32 param_count, u32 frame_byte_size, evaluekind_t return_kind)
    {
        write_le_u32(block + offset, body_address);
        write_le_u32(block + offset + 4, param_start);
        write_le_u32(block + offset + 8, param_count);
        write_le_u32(block + offset + 12, frame_byte_size);
        block[offset + 16] = (byte)return_kind;
        block[offset + 17] = 0;
        block[offset + 18] = 0;
        block[offset + 19] = 0;
    }

    static void write_image_header(byte* block, u32 entry_point, u32 bss_byte_size, u32 frame_size, u32 frame_byte_size)
    {
        write_le_u32(block, ProgramImageMagic);
        write_le_u16(block + 4, ProgramImageVersion);
        block[6] = ProgramImageEndianLittle;
        block[7] = ProgramImageABI;
        write_le_u32(block + 8, entry_point);
        write_le_u32(block + 12, bss_byte_size);
        write_le_u32(block + 16, frame_size);
        write_le_u32(block + 20, frame_byte_size);
    }
}

UNITTEST_SUITE_BEGIN(cova_image)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        UNITTEST_TEST(open_zero_copy_views)
        {
            u32  aligned_storage[40] = {};
            byte* block = (byte*)aligned_storage;
            write_image_header(block, 0, 12, 1, 4);

            const u32 functions_offset     = 72;
            const u32 param_kinds_offset   = 112;
            const u32 param_offsets_offset = 116;
            const u32 text_offset          = 120;
            write_array_header(block, 24, 2, functions_offset);
            write_array_header(block, 32, 1, param_kinds_offset);
            write_array_header(block, 40, 1, param_offsets_offset);
            write_array_header(block, 48, 4, text_offset);
            write_array_header(block, 56, 0, 0);
            write_array_header(block, 64, 0, 0);
            write_function(block, functions_offset, 0, 0, 0, 0, KindVoid);
            write_function(block, functions_offset + ProgramImageFunctionSize, 2, 0, 1, 4, KindInt32);
            block[param_kinds_offset] = KindInt32;
            write_le_u32(block + param_offsets_offset, 0);
            write_le_u16(block + text_offset, make_instruction(OpRet, KindNone));
            write_le_u16(block + text_offset + 2, make_instruction(OpRet, KindNone));

            const linked_program_t* program = open_program_image(block, text_offset + 4);

            CHECK_TRUE(program == (const linked_program_t*)block);
            CHECK_TRUE(program->m_text.data() == block + text_offset);
            CHECK_TRUE((const byte*)program->m_functions.data() == block + functions_offset);
            CHECK_TRUE((const byte*)program->m_param_kinds.data() == block + param_kinds_offset);
            CHECK_TRUE((const byte*)program->m_param_offsets.data() == block + param_offsets_offset);
            CHECK_EQUAL((u32)2, program->m_functions.m_size);
            CHECK_EQUAL((u32)1, program->m_frame_size);
            CHECK_EQUAL((u32)4, program->m_frame_byte_size);
            CHECK_EQUAL((u32)12, program->m_bss_byte_size);
            CHECK_EQUAL((u32)KindInt32, (u32)program->m_functions[1].m_return_kind);
        }

        UNITTEST_TEST(run_memory_block)
        {
            u32  aligned_storage[32] = {};
            byte* block = (byte*)aligned_storage;
            write_image_header(block, 0, 4, 0, 0);

            const u32 functions_offset = 72;
            const u32 text_offset      = 92;
            const u32 const_offset     = 100;
            const u32 data_offset      = 104;
            write_array_header(block, 24, 1, functions_offset);
            write_array_header(block, 32, 0, 0);
            write_array_header(block, 40, 0, 0);
            write_array_header(block, 48, 8, text_offset);
            write_array_header(block, 56, 3, const_offset);
            write_array_header(block, 64, 4, data_offset);
            write_function(block, functions_offset, 0, 0, 0, 0, KindInt32);
            write_le_u16(block + text_offset, make_instruction(OpPush, KindInt32));
            write_le_u32(block + text_offset + 2, 42);
            write_le_u16(block + text_offset + 6, make_instruction(OpRet, KindNone));
            block[const_offset]     = 1;
            block[const_offset + 1] = 2;
            block[const_offset + 2] = 3;
            write_le_u32(block + data_offset, 0x12345678U);

            byte frame[16] = {}, bss[8] = {}, external[8] = {}, data[8] = {}, stack[16] = {};
            call_frame_t call_frames[2] = {};
            vm_t vm;
            initialize_vm(&vm, call_frames, 2, {frame, 16, 16}, {bss, 0, 8}, {external, 8, 8}, {data, 0, 8}, {stack, 0, 16});
            run_vm_image(&vm, block, data_offset + 4);

            CHECK_TRUE(vm.m_program == (const linked_program_t*)block);
            CHECK_EQUAL((u32)42, pop_bits32(&vm, KindInt32));
            CHECK_EQUAL((u32)4, vm.m_memory.m_segments[SegmentBSS].m_size);
            CHECK_EQUAL((u32)0x12345678U, read_u32(&vm.m_memory, make_address(SegmentData, 0)));
            CHECK_EQUAL((u8)2, read_u8(&vm.m_memory, make_address(SegmentConst, 1)));
        }
    }
}
UNITTEST_SUITE_END