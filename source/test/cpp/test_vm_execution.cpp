#include "ccore/c_target.h"

#include "ccova/float_bits.h"
#include "ccova/builtins.h"
#include "ccova/program_image.h"
#include "ccova/vm.h"

#include "cunittest/cunittest.h"

using namespace ncore;

namespace
{
    static const i32 VMDefaultTimerCapacity = 8;

    static void emit_instruction(segment_memory_t* text, instruction_t instruction) { append_u16(text, instruction); }
    static void emit_u32(segment_memory_t* text, u32 value) { append_u32(text, value); }

    static void emit_push_u32(segment_memory_t* text, evaluekind_t kind, u32 value)
    {
        emit_instruction(text, make_instruction(OpPush, kind));
        append_u32(text, value);
    }

    static void initialize_test_vm(vm_t* vm, call_frame_t* call_frames, u64* timer_end_time_ms, u32* timer_duration_ms, byte* frame, byte* bss, byte* external, byte* data, byte* stack)
    {
        vm_segments_t segments = {{frame, 64, 64}, {bss, 0, 16}, {external, 16, 16}, {data, 0, 16}, {stack, 0, 64}};
        initialize_vm(vm, {call_frames, 0, 8}, {timer_end_time_ms, timer_duration_ms, VMDefaultTimerCapacity}, segments);
    }

    static void initialize_program(linked_program_t* program, const byte* text, u32 text_size, const script_function_t* functions, u32 function_count, const evaluekind_t* param_kinds = nullptr, const u32* param_offsets = nullptr, u32 param_count = 0)
    {
        program->m_magic           = ProgramImageMagic;
        program->m_version         = ProgramImageVersion;
        program->m_endian          = ProgramImageEndianLittle;
        program->m_abi             = ProgramImageABI;
        program->m_entry_point     = 0;
        program->m_bss_byte_size   = 0;
        program->m_frame_size      = param_count;
        program->m_frame_byte_size = 64;
        program->m_functions.set(functions, function_count);
        program->m_param_kinds.set(param_kinds, param_count);
        program->m_param_offsets.set(param_offsets, param_count);
        program->m_text.set(text, text_size);
        program->m_const_data.set(nullptr, 0);
        program->m_data_data.set(nullptr, 0);
    }

    static void extern_add_one(void* host_context, vm_t* vm, u32 import_id)
    {
        *(u32*)host_context = import_id;
        const u32 value = pop_bits32(vm, KindUint32);
        push_bits32(vm, KindUint32, value + 1);
    }
}

UNITTEST_SUITE_BEGIN(cova_vm_execution)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        UNITTEST_TEST(arithmetic_conversion_and_comparison)
        {
            byte text_storage[128] = {};
            segment_memory_t text = {text_storage, 0, 128};
            emit_push_u32(&text, KindInt32, 7);
            emit_push_u32(&text, KindInt32, 5);
            emit_instruction(&text, make_arithmetic_instruction(KindInt32, ArithmeticMul));
            emit_instruction(&text, make_convert_instruction(KindInt32, KindFloat64));
            emit_instruction(&text, make_instruction(OpPush, KindFloat64));
            append_u64(&text, f64_to_bits(35.0));
            emit_instruction(&text, make_compare_instruction(KindFloat64, CompareEqual));
            emit_instruction(&text, make_instruction(OpRet, KindNone));

            const script_function_t functions[] = {{0, 0, 0, 0, KindBool}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);
            load_program(&vm, &program);
            run_vm(&vm, 0);

            CHECK_EQUAL((u32)1, pop_bits32(&vm, KindBool));
            CHECK_EQUAL((u32)0, vm.m_call_frames.m_count);
        }

        UNITTEST_TEST(bitwise_modulo_and_masked_shifts)
        {
            byte text_storage[128] = {};
            segment_memory_t text = {text_storage, 0, 128};
            emit_push_u32(&text, KindInt32, 29);
            emit_push_u32(&text, KindInt32, 6);
            emit_instruction(&text, make_arithmetic_instruction(KindInt32, ArithmeticModulo));
            emit_push_u32(&text, KindInt32, 3);
            emit_instruction(&text, make_arithmetic_instruction(KindInt32, ArithmeticBitwiseAnd));
            emit_push_u32(&text, KindInt32, 8);
            emit_instruction(&text, make_arithmetic_instruction(KindInt32, ArithmeticBitwiseOr));
            emit_push_u32(&text, KindInt32, 1);
            emit_instruction(&text, make_arithmetic_instruction(KindInt32, ArithmeticBitwiseXor));
            emit_push_u32(&text, KindInt32, 35);
            emit_instruction(&text, make_arithmetic_instruction(KindInt32, ArithmeticShiftLeft));
            emit_push_u32(&text, KindInt32, (u32)-1);
            emit_instruction(&text, make_arithmetic_instruction(KindInt32, ArithmeticMul));
            emit_push_u32(&text, KindInt32, 34);
            emit_instruction(&text, make_arithmetic_instruction(KindInt32, ArithmeticShiftRight));
            emit_instruction(&text, make_instruction(OpRet, KindNone));

            const script_function_t functions[] = {{0, 0, 0, 0, KindInt32}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);
            load_program(&vm, &program);
            run_vm(&vm, 0);

            CHECK_EQUAL((u32)-16, pop_bits32(&vm, KindInt32));
        }

        UNITTEST_TEST(math_builtins)
        {
            byte text_storage[128] = {};
            segment_memory_t text = {text_storage, 0, 128};
            emit_push_u32(&text, KindInt32, (u32)-7);
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInAbs, KindInt32)));
            emit_instruction(&text, make_instruction(OpPush, KindFloat32));
            append_u32(&text, f32_to_bits(9.0f));
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInSqrt, KindFloat32)));
            emit_instruction(&text, make_instruction(OpPush, KindFloat64));
            append_u64(&text, f64_to_bits(2.0));
            emit_instruction(&text, make_instruction(OpPush, KindFloat64));
            append_u64(&text, f64_to_bits(3.0));
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInPow, KindFloat64)));
            emit_instruction(&text, make_instruction(OpRet, KindNone));

            const script_function_t functions[] = {{0, 0, 0, 0, KindVoid}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);
            load_program(&vm, &program);
            run_vm(&vm, 0);

            CHECK_EQUAL((f64)8.0, bits_to_f64(pop_bits64(&vm, KindFloat64)));
            CHECK_EQUAL((f32)3.0f, bits_to_f32(pop_bits32(&vm, KindFloat32)));
            CHECK_EQUAL((u32)7, pop_bits32(&vm, KindInt32));
        }

        UNITTEST_TEST(time_tick_builtin)
        {
            byte text_storage[32] = {};
            segment_memory_t text = {text_storage, 0, 32};
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInTickTime, KindFloat32)));
            emit_instruction(&text, make_instruction(OpRet, KindNone));

            const script_function_t functions[] = {{0, 0, 0, 0, KindFloat32}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);

            load_program(&vm, &program);

            run_vm(&vm, 1000);
            CHECK_EQUAL((f32)0.0f, bits_to_f32(pop_bits32(&vm, KindFloat32)));
            run_vm(&vm, 1250);
            CHECK_EQUAL((f32)0.25f, bits_to_f32(pop_bits32(&vm, KindFloat32)));
            run_vm(&vm, 1250);
            CHECK_EQUAL((f32)0.0f, bits_to_f32(pop_bits32(&vm, KindFloat32)));
        }

        UNITTEST_TEST(random_state_persists_across_ticks)
        {
            byte text_storage[16] = {};
            segment_memory_t text = {text_storage, 0, 16};
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInRandom, KindInt32)));
            emit_instruction(&text, make_instruction(OpRet, KindNone));

            const script_function_t functions[] = {{0, 0, 0, 0, KindInt32}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);

            load_program(&vm, &program);
            run_vm(&vm, 0);
            const u32 first = pop_bits32(&vm, KindInt32);
            run_vm(&vm, 0);
            const u32 second = pop_bits32(&vm, KindInt32);
            CHECK_TRUE(first != second);

            load_program(&vm, &program);
            run_vm(&vm, 0);
            CHECK_EQUAL(first, pop_bits32(&vm, KindInt32));
        }

        UNITTEST_TEST(time_timer_builtins)
        {
            byte text_storage[64] = {};
            segment_memory_t text = {text_storage, 0, 64};
            emit_instruction(&text, make_address_instruction(SegmentExtern));
            emit_u32(&text, 0);
            emit_instruction(&text, make_instruction(OpDereference, KindBool));
            emit_instruction(&text, make_instruction(OpJumpIfFalse, KindNone));
            const u32 start_target_operand = text.m_size;
            emit_u32(&text, 0);
            emit_instruction(&text, make_instruction(OpPush, KindUint8));
            append_u8(&text, 0);
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInTimerDone, KindBool)));
            emit_instruction(&text, make_instruction(OpRet, KindNone));
            const u32 start_target = text.m_size;
            emit_instruction(&text, make_instruction(OpPush, KindUint8));
            append_u8(&text, 0);
            emit_push_u32(&text, KindUint32, 100);
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInTimerStart, KindVoid)));
            emit_instruction(&text, make_instruction(OpPush, KindBool));
            append_u8(&text, 0);
            emit_instruction(&text, make_instruction(OpRet, KindNone));
            write_u32(&text, start_target_operand, start_target);

            const script_function_t functions[] = {{0, 0, 0, 0, KindBool}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);

            load_program(&vm, &program);

            run_vm(&vm, 1000);
            CHECK_EQUAL((u32)0, pop_bits32(&vm, KindBool));
            external[0] = 1;
            run_vm(&vm, 1100);
            CHECK_EQUAL((u32)0, pop_bits32(&vm, KindBool));
            run_vm(&vm, 1101);
            CHECK_EQUAL((u32)1, pop_bits32(&vm, KindBool));
        }

        UNITTEST_TEST(typed_min_max_builtins)
        {
            byte text_storage[128] = {};
            segment_memory_t text = {text_storage, 0, 128};
            emit_instruction(&text, make_instruction(OpPush, KindInt16));
            append_u16(&text, (u16)(s16)-7);
            emit_instruction(&text, make_instruction(OpPush, KindInt16));
            append_u16(&text, 3);
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInMin, KindInt16)));
            emit_push_u32(&text, KindUint32, 4000000000U);
            emit_push_u32(&text, KindUint32, 17U);
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInMax, KindUint32)));
            emit_instruction(&text, make_instruction(OpPush, KindFloat64));
            append_u64(&text, f64_to_bits(-2.5));
            emit_instruction(&text, make_instruction(OpPush, KindFloat64));
            append_u64(&text, f64_to_bits(9.5));
            emit_instruction(&text, make_builtin_instruction(make_builtin_function(BuiltInMax, KindFloat64)));
            emit_instruction(&text, make_instruction(OpRet, KindNone));

            const script_function_t functions[] = {{0, 0, 0, 0, KindVoid}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);

            load_program(&vm, &program);
            run_vm(&vm, 0);

            CHECK_EQUAL((f64)9.5, bits_to_f64(pop_bits64(&vm, KindFloat64)));
            CHECK_EQUAL((u32)4000000000U, pop_bits32(&vm, KindUint32));
            CHECK_EQUAL((s16)-7, (s16)(u16)pop_bits32(&vm, KindInt16));
        }

        UNITTEST_TEST(address_offset_assign_and_dereference)
        {
            byte text_storage[128] = {};
            segment_memory_t text = {text_storage, 0, 128};
            emit_push_u32(&text, KindUint32, 0x12345678U);
            emit_instruction(&text, make_address_instruction(SegmentExtern));
            emit_u32(&text, 0);
            emit_push_u32(&text, KindInt32, 4);
            emit_instruction(&text, make_instruction(OpOffset, KindNone));
            emit_instruction(&text, make_instruction(OpAssign, KindUint32));
            emit_instruction(&text, make_address_instruction(SegmentExtern));
            emit_u32(&text, 4);
            emit_instruction(&text, make_instruction(OpDereference, KindUint32));
            emit_instruction(&text, make_instruction(OpRet, KindNone));

            const script_function_t functions[] = {{0, 0, 0, 0, KindUint32}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);
            load_program(&vm, &program);
            run_vm(&vm, 0);

            CHECK_EQUAL((u32)0x12345678U, pop_bits32(&vm, KindUint32));
            CHECK_EQUAL((u32)0x12345678U, read_u32(&vm.m_memory, make_address(SegmentExtern, 4)));
        }

        UNITTEST_TEST(conditional_and_unconditional_jump)
        {
            byte text_storage[128] = {};
            segment_memory_t text = {text_storage, 0, 128};
            emit_instruction(&text, make_instruction(OpPush, KindBool));
            append_u8(&text, 0);
            emit_instruction(&text, make_instruction(OpJumpIfFalse, KindNone));
            const u32 false_target_operand = text.m_size;
            emit_u32(&text, 0);
            emit_push_u32(&text, KindUint32, 1);
            emit_instruction(&text, make_instruction(OpJump, KindNone));
            const u32 end_target_operand = text.m_size;
            emit_u32(&text, 0);
            const u32 false_target = text.m_size;
            emit_push_u32(&text, KindUint32, 42);
            const u32 end_target = text.m_size;
            emit_instruction(&text, make_instruction(OpRet, KindNone));
            write_u32(&text, false_target_operand, false_target);
            write_u32(&text, end_target_operand, end_target);

            const script_function_t functions[] = {{0, 0, 0, 0, KindUint32}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);
            load_program(&vm, &program);
            run_vm(&vm, 0);

            CHECK_EQUAL((u32)42, pop_bits32(&vm, KindUint32));
        }

        UNITTEST_TEST(script_call_with_parameter)
        {
            byte text_storage[128] = {};
            segment_memory_t text = {text_storage, 0, 128};
            emit_push_u32(&text, KindInt32, 7);
            emit_instruction(&text, make_instruction(OpCall, KindNone));
            emit_u32(&text, 1);
            emit_instruction(&text, make_instruction(OpRet, KindNone));
            const u32 function_body = text.m_size;
            emit_instruction(&text, make_address_instruction(SegmentFrame));
            emit_u32(&text, 0);
            emit_instruction(&text, make_instruction(OpDereference, KindInt32));
            emit_push_u32(&text, KindInt32, 1);
            emit_instruction(&text, make_arithmetic_instruction(KindInt32, ArithmeticAdd));
            emit_instruction(&text, make_instruction(OpRet, KindNone));

            const script_function_t functions[] = {
                {0, 0, 0, 0, KindInt32},
                {function_body, 0, 1, 4, KindInt32},
            };
            const evaluekind_t param_kinds[] = {KindInt32};
            const u32 param_offsets[] = {0};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 2, param_kinds, param_offsets, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);
            load_program(&vm, &program);
            run_vm(&vm, 0);

            CHECK_EQUAL((u32)8, pop_bits32(&vm, KindInt32));
            CHECK_EQUAL((u32)0, vm.m_frame_top);
        }

        UNITTEST_TEST(extern_call)
        {
            byte text_storage[64] = {};
            segment_memory_t text = {text_storage, 0, 64};
            emit_push_u32(&text, KindUint32, 9);
            emit_instruction(&text, make_instruction(OpCallExtern, KindNone));
            emit_u32(&text, 7);
            emit_instruction(&text, make_instruction(OpRet, KindNone));

            const script_function_t functions[] = {{0, 0, 0, 0, KindUint32}};
            linked_program_t program;
            initialize_program(&program, text_storage, text.m_size, functions, 1);

            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[64] = {};
            call_frame_t call_frames[8] = {};
            u64 timer_end_time_ms[VMDefaultTimerCapacity] = {};
            u32 timer_duration_ms[VMDefaultTimerCapacity] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, timer_end_time_ms, timer_duration_ms, frame, bss, external, data, stack);
            u32 import_id = 0;
            register_extern_dispatcher(&vm, &import_id, extern_add_one);
            load_program(&vm, &program);
            run_vm(&vm, 0);

            CHECK_EQUAL((u32)7, import_id);
            CHECK_EQUAL((u32)10, pop_bits32(&vm, KindUint32));
        }
    }
}
UNITTEST_SUITE_END
