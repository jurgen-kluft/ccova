#include "ccore/c_target.h"

#include "ccova/program_image.h"
#include "ccova/vm.h"

#include "cunittest/cunittest.h"

using namespace ncore;

namespace
{
    struct extern_call_t
    {
        vm_t* m_vm;
        u32   m_import_id;
    };

    static void test_extern_dispatcher(void* host_context, vm_t* vm, u32 import_id)
    {
        extern_call_t* call = (extern_call_t*)host_context;
        call->m_vm           = vm;
        call->m_import_id    = import_id;
    }
}

UNITTEST_SUITE_BEGIN(cova_vm)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        static const i32 VMDefaultTimerCapacity = 8;

        UNITTEST_TEST(initialize)
        {
            byte         frame_storage[32] = {};
            byte         bss_storage[16]   = {};
            byte         extern_storage[8] = {};
            byte         data_storage[16]  = {};
            byte         stack_storage[16] = {};
            call_frame_t call_frames[4]    = {};
            u64          timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32          timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t         vm;

            vm_segments_t segments = {{frame_storage, 32, 32}, {bss_storage, 0, 16}, {extern_storage, 8, 8}, {data_storage, 0, 16}, {stack_storage, 0, 16}};
            initialize_vm(&vm, {call_frames, 0, 4}, {timer_end_time_ms, timer_duration_ms, VMDefaultTimerCapacity}, segments);

            CHECK_EQUAL((u32)0, vm.m_pc);
            CHECK_TRUE(vm.m_program == nullptr);
            CHECK_TRUE(vm.m_call_frames.m_frames == call_frames);
            CHECK_EQUAL((u32)0, vm.m_call_frames.m_count);
            CHECK_EQUAL((u32)4, vm.m_call_frames.m_capacity);
            CHECK_EQUAL((u32)0, vm.m_frame_top);
            CHECK_TRUE(vm.m_extern_dispatcher == nullptr);
            CHECK_TRUE(vm.m_timers.m_end_time_ms == timer_end_time_ms);
            CHECK_TRUE(vm.m_timers.m_duration_ms == timer_duration_ms);
            CHECK_EQUAL((u32)VMDefaultTimerCapacity, vm.m_timers.m_capacity);
            CHECK_EQUAL((u64)0, vm.m_time.m_current_time_ms);
            CHECK_EQUAL((f32)0.0f, vm.m_time.m_tick_time);
            CHECK_FALSE(vm.m_time.m_has_current_time);
            for (u32 index = 0; index < VMDefaultTimerCapacity; ++index)
            {
                CHECK_EQUAL((u64)0, timer_end_time_ms[index]);
                CHECK_EQUAL((u32)0, timer_duration_ms[index]);
            }
        }

        UNITTEST_TEST(load_and_reset)
        {
            byte         frame_storage[32];
            byte         bss_storage[16];
            byte         extern_storage[8] = {};
            byte         data_storage[16];
            byte         stack_storage[16] = {};
            call_frame_t call_frames[4]    = {};
            u64          timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32          timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t         vm;
            for (u32 index = 0; index < 32; ++index)
                frame_storage[index] = 0xffU;
            for (u32 index = 0; index < 16; ++index)
            {
                bss_storage[index]  = 0xffU;
                data_storage[index] = 0xffU;
            }

            vm_segments_t segments = {{frame_storage, 32, 32}, {bss_storage, 0, 16}, {extern_storage, 8, 8}, {data_storage, 0, 16}, {stack_storage, 0, 16}};
            initialize_vm(&vm, {call_frames, 0, 4}, {timer_end_time_ms, timer_duration_ms, VMDefaultTimerCapacity}, segments);

            const byte              text[]       = {OpRet, 0};
            const byte              const_data[] = {1, 2, 3};
            const byte              data_data[]  = {4, 5, 6, 7};
            const script_function_t functions[]  = {{0, 0, 0, 12, KindVoid}};
            linked_program_t program = {};
            program.m_magic           = ProgramImageMagic;
            program.m_version         = ProgramImageVersion;
            program.m_endian          = ProgramImageEndianLittle;
            program.m_abi             = ProgramImageABI;
            program.m_entry_point     = 0;
            program.m_bss_byte_size   = 6;
            program.m_frame_size      = 1;
            program.m_frame_byte_size = 12;
            program.m_functions.set(functions, 1);
            program.m_param_kinds.set(nullptr, 0);
            program.m_param_offsets.set(nullptr, 0);
            program.m_text.set(text, 2);
            program.m_const_data.set(const_data, 3);
            program.m_data_data.set(data_data, 4);

            load_program(&vm, &program);
            CHECK_TRUE(vm.m_program == &program);
            CHECK_EQUAL((u32)6, vm.m_memory.m_segments[SegmentBSS].m_size);
            CHECK_EQUAL((u32)4, vm.m_memory.m_segments[SegmentData].m_size);
            CHECK_EQUAL((u32)3, vm.m_memory.m_segments[SegmentConst].m_size);
            CHECK_EQUAL((u8)2, read_u8(&vm.m_memory, make_address(SegmentConst, 1)));
            for (u32 index = 0; index < 6; ++index)
                CHECK_EQUAL((u8)0, bss_storage[index]);
            for (u32 index = 0; index < 4; ++index)
                CHECK_EQUAL(data_data[index], data_storage[index]);

            bss_storage[0]  = 8;
            data_storage[0] = 9;
            append_u32(&vm.m_memory.m_segments[SegmentStack], 0x12345678U);
            timer_duration_ms[0] = 100;
            timer_end_time_ms[0] = 1100;
            reset_vm(&vm);

            CHECK_EQUAL((u32)0, vm.m_pc);
            CHECK_EQUAL((u32)1, vm.m_call_frames.m_count);
            CHECK_EQUAL((u32)12, vm.m_frame_top);
            CHECK_EQUAL((u32)0, vm.m_call_frames.m_frames[0].m_return_pc);
            CHECK_EQUAL((u32)0, vm.m_call_frames.m_frames[0].m_local_base);
            CHECK_EQUAL((u32)KindVoid, (u32)vm.m_call_frames.m_frames[0].m_return_kind);
            CHECK_EQUAL((u32)0, vm.m_memory.m_segments[SegmentStack].m_size);
            CHECK_EQUAL((u8)8, bss_storage[0]);
            CHECK_EQUAL((u8)9, data_storage[0]);
            for (u32 index = 0; index < 32; ++index)
                CHECK_EQUAL((u8)0, frame_storage[index]);
            CHECK_EQUAL((u32)100, timer_duration_ms[0]);
            CHECK_EQUAL((u64)1100, timer_end_time_ms[0]);

            vm.m_time.m_current_time_ms  = 1200;
            vm.m_time.m_tick_time        = 0.1f;
            vm.m_time.m_has_current_time = true;
            load_program(&vm, &program);
            for (u32 index = 0; index < 6; ++index)
                CHECK_EQUAL((u8)0, bss_storage[index]);
            for (u32 index = 0; index < 4; ++index)
                CHECK_EQUAL(data_data[index], data_storage[index]);
            CHECK_EQUAL((u64)1200, vm.m_time.m_current_time_ms);
            CHECK_EQUAL((f32)0.1f, vm.m_time.m_tick_time);
            CHECK_TRUE(vm.m_time.m_has_current_time);
        }

        UNITTEST_TEST(register_extern_dispatcher)
        {
            byte         frame_storage[8]  = {};
            byte         stack_storage[8]  = {};
            call_frame_t call_frames[1]    = {};
            u64          timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32          timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t         vm;
            const segment_memory_t empty = {nullptr, 0, 0};
            vm_segments_t segments = {{frame_storage, 8, 8}, empty, empty, empty, {stack_storage, 0, 8}};
            initialize_vm(&vm, {call_frames, 0, 1}, {timer_end_time_ms, timer_duration_ms, VMDefaultTimerCapacity}, segments);

            extern_call_t call = {nullptr, 0};
            register_extern_dispatcher(&vm, &call, test_extern_dispatcher);
            CHECK_TRUE(vm.m_host_context == &call);
            CHECK_TRUE(vm.m_extern_dispatcher == test_extern_dispatcher);

            vm.m_extern_dispatcher(vm.m_host_context, &vm, 17);
            CHECK_TRUE(call.m_vm == &vm);
            CHECK_EQUAL((u32)17, call.m_import_id);
        }
    }
}
UNITTEST_SUITE_END
