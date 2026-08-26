#include "ccore/c_target.h"

#include "ccova/builtins.h"
#include "ccova/float_bits.h"
#include "ccova/vm.h"

#include "cunittest/cunittest.h"

using namespace ncore;

namespace
{
    static void initialize_test_vm(vm_t* vm, call_frame_t* call_frames, byte* frame, byte* bss, byte* external, byte* data, byte* stack)
    {
        initialize_vm(vm, call_frames, 4, {frame, 64, 64}, {bss, 0, 16}, {external, 16, 16}, {data, 0, 16}, {stack, 0, 128});
    }
}

UNITTEST_SUITE_BEGIN(cova_builtins)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        UNITTEST_TEST(range_and_interpolation)
        {
            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[128] = {};
            call_frame_t call_frames[4] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, frame, bss, external, data, stack);

            push_bits32(&vm, KindInt32, 5);
            push_bits32(&vm, KindInt32, 0);
            push_bits32(&vm, KindInt32, 10);
            push_bits32(&vm, KindInt32, 0);
            push_bits32(&vm, KindInt32, 100);
            execute_builtin(&vm, make_builtin_function(BuiltInMap, KindInt32));
            CHECK_EQUAL((s32)50, (s32)pop_bits32(&vm, KindInt32));

            push_bits32(&vm, KindInt32, 15);
            push_bits32(&vm, KindInt32, 0);
            push_bits32(&vm, KindInt32, 10);
            push_bits32(&vm, KindInt32, 0);
            push_bits32(&vm, KindInt32, 100);
            execute_builtin(&vm, make_builtin_function(BuiltInMap, KindInt32));
            CHECK_EQUAL((s32)100, (s32)pop_bits32(&vm, KindInt32));

            push_bits32(&vm, KindInt32, 2);
            push_bits32(&vm, KindInt32, 10);
            push_bits32(&vm, KindInt32, 0);
            push_bits32(&vm, KindInt32, 100);
            push_bits32(&vm, KindInt32, 0);
            execute_builtin(&vm, make_builtin_function(BuiltInMap, KindInt32));
            CHECK_EQUAL((s32)20, (s32)pop_bits32(&vm, KindInt32));

            push_bits32(&vm, KindInt32, 5);
            push_bits32(&vm, KindInt32, 0);
            push_bits32(&vm, KindInt32, 10);
            push_bits32(&vm, KindInt32, 42);
            push_bits32(&vm, KindInt32, 42);
            execute_builtin(&vm, make_builtin_function(BuiltInMap, KindInt32));
            CHECK_EQUAL((s32)42, (s32)pop_bits32(&vm, KindInt32));

            push_bits32(&vm, KindInt8, (u8)(s8)-5);
            push_bits32(&vm, KindInt8, (u8)(s8)-10);
            push_bits32(&vm, KindInt8, (u8)(s8)10);
            push_bits32(&vm, KindInt8, (u8)(s8)0);
            push_bits32(&vm, KindInt8, (u8)(s8)100);
            execute_builtin(&vm, make_builtin_function(BuiltInMap, KindInt8));
            CHECK_EQUAL((s8)25, (s8)(u8)pop_bits32(&vm, KindInt8));

            push_bits32(&vm, KindUint16, 50000);
            push_bits32(&vm, KindUint16, 40000);
            push_bits32(&vm, KindUint16, 60000);
            push_bits32(&vm, KindUint16, 0);
            push_bits32(&vm, KindUint16, 100);
            execute_builtin(&vm, make_builtin_function(BuiltInMap, KindUint16));
            CHECK_EQUAL((u16)50, (u16)pop_bits32(&vm, KindUint16));

            push_bits32(&vm, KindUint32, 4200000000U);
            push_bits32(&vm, KindUint32, 4000000000U);
            push_bits32(&vm, KindUint32, 4100000000U);
            push_bits32(&vm, KindUint32, 0);
            push_bits32(&vm, KindUint32, 1);
            execute_builtin(&vm, make_builtin_function(BuiltInMap, KindUint32));
            CHECK_EQUAL((u32)1, pop_bits32(&vm, KindUint32));

            push_bits32(&vm, KindFloat32, f32_to_bits(-5.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(0.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(10.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(20.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(40.0f));
            execute_builtin(&vm, make_builtin_function(BuiltInMap, KindFloat32));
            CHECK_EQUAL((f32)20.0f, bits_to_f32(pop_bits32(&vm, KindFloat32)));

            push_bits32(&vm, KindInt16, (u16)(s16)-20);
            push_bits32(&vm, KindInt16, (u16)(s16)-10);
            push_bits32(&vm, KindInt16, (u16)(s16)10);
            execute_builtin(&vm, make_builtin_function(BuiltInClamp, KindInt16));
            CHECK_EQUAL((s16)-10, (s16)(u16)pop_bits32(&vm, KindInt16));

            push_bits64(&vm, KindFloat64, f64_to_bits(10.0));
            push_bits64(&vm, KindFloat64, f64_to_bits(20.0));
            push_bits64(&vm, KindFloat64, f64_to_bits(0.5));
            execute_builtin(&vm, make_builtin_function(BuiltInLerp, KindFloat64));
            CHECK_EQUAL((f64)15.0, bits_to_f64(pop_bits64(&vm, KindFloat64)));

            push_bits32(&vm, KindFloat32, f32_to_bits(9.0f));
            execute_builtin(&vm, make_builtin_function(BuiltInSqrt, KindFloat32));
            CHECK_EQUAL((f32)3.0f, bits_to_f32(pop_bits32(&vm, KindFloat32)));

            const ebuiltinoperation_t unary_operations[] = {BuiltInSin, BuiltInCos, BuiltInTan, BuiltInAsin, BuiltInAcos, BuiltInAtan};
            const f32                  unary_inputs32[]   = {0.0f, 0.0f, 0.0f, 0.0f, 1.0f, 0.0f};
            const f32                  unary_results32[]  = {0.0f, 1.0f, 0.0f, 0.0f, 0.0f, 0.0f};
            const f64                  unary_inputs64[]   = {0.0, 0.0, 0.0, 0.0, 1.0, 0.0};
            const f64                  unary_results64[]  = {0.0, 1.0, 0.0, 0.0, 0.0, 0.0};
            for (u32 i = 0; i < sizeof(unary_operations) / sizeof(unary_operations[0]); ++i)
            {
                push_bits32(&vm, KindFloat32, f32_to_bits(unary_inputs32[i]));
                execute_builtin(&vm, make_builtin_function(unary_operations[i], KindFloat32));
                CHECK_EQUAL(unary_results32[i], bits_to_f32(pop_bits32(&vm, KindFloat32)));

                push_bits64(&vm, KindFloat64, f64_to_bits(unary_inputs64[i]));
                execute_builtin(&vm, make_builtin_function(unary_operations[i], KindFloat64));
                CHECK_EQUAL(unary_results64[i], bits_to_f64(pop_bits64(&vm, KindFloat64)));
            }

            push_bits64(&vm, KindFloat64, f64_to_bits(16.0));
            execute_builtin(&vm, make_builtin_function(BuiltInSqrt, KindFloat64));
            CHECK_EQUAL((f64)4.0, bits_to_f64(pop_bits64(&vm, KindFloat64)));

            push_bits32(&vm, KindFloat32, f32_to_bits(2.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(3.0f));
            execute_builtin(&vm, make_builtin_function(BuiltInPow, KindFloat32));
            CHECK_EQUAL((f32)8.0f, bits_to_f32(pop_bits32(&vm, KindFloat32)));

            push_bits64(&vm, KindFloat64, f64_to_bits(3.0));
            push_bits64(&vm, KindFloat64, f64_to_bits(2.0));
            execute_builtin(&vm, make_builtin_function(BuiltInPow, KindFloat64));
            CHECK_EQUAL((f64)9.0, bits_to_f64(pop_bits64(&vm, KindFloat64)));

            push_bits32(&vm, KindFloat32, f32_to_bits(10.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(20.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(0.5f));
            execute_builtin(&vm, make_builtin_function(BuiltInLerp, KindFloat32));
            CHECK_EQUAL((f32)15.0f, bits_to_f32(pop_bits32(&vm, KindFloat32)));

            push_bits32(&vm, KindFloat32, f32_to_bits(0.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(10.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(5.0f));
            execute_builtin(&vm, make_builtin_function(BuiltInSmoothStep, KindFloat32));
            CHECK_EQUAL((f32)0.5f, bits_to_f32(pop_bits32(&vm, KindFloat32)));

            push_bits32(&vm, KindFloat32, f32_to_bits(1.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(1.0f));
            push_bits32(&vm, KindFloat32, f32_to_bits(0.5f));
            execute_builtin(&vm, make_builtin_function(BuiltInSlerp, KindFloat32));
            CHECK_EQUAL((f32)1.0f, bits_to_f32(pop_bits32(&vm, KindFloat32)));

            push_bits64(&vm, KindFloat64, f64_to_bits(1.0));
            push_bits64(&vm, KindFloat64, f64_to_bits(1.0));
            push_bits64(&vm, KindFloat64, f64_to_bits(0.5));
            execute_builtin(&vm, make_builtin_function(BuiltInSlerp, KindFloat64));
            CHECK_EQUAL((f64)1.0, bits_to_f64(pop_bits64(&vm, KindFloat64)));

            push_bits32(&vm, KindInt32, 7);
            push_bits32(&vm, KindInt32, 3);
            execute_builtin(&vm, make_builtin_function(BuiltInMin, KindInt32));
            CHECK_EQUAL((s32)3, (s32)pop_bits32(&vm, KindInt32));

            push_bits64(&vm, KindFloat64, f64_to_bits(7.0));
            push_bits64(&vm, KindFloat64, f64_to_bits(3.0));
            execute_builtin(&vm, make_builtin_function(BuiltInMax, KindFloat64));
            CHECK_EQUAL((f64)7.0, bits_to_f64(pop_bits64(&vm, KindFloat64)));

            push_bits64(&vm, KindFloat64, f64_to_bits(0.0));
            push_bits64(&vm, KindFloat64, f64_to_bits(10.0));
            push_bits64(&vm, KindFloat64, f64_to_bits(5.0));
            execute_builtin(&vm, make_builtin_function(BuiltInSmoothStep, KindFloat64));
            CHECK_EQUAL((f64)0.5, bits_to_f64(pop_bits64(&vm, KindFloat64)));

            push_bits32(&vm, KindInt32, 0);
            push_bits32(&vm, KindInt32, 100);
            push_bits32(&vm, KindInt32, 64);
            push_bits32(&vm, KindUint8, 8);
            execute_builtin(&vm, make_builtin_function(BuiltInLerp, KindInt32));
            CHECK_EQUAL((s32)25, (s32)pop_bits32(&vm, KindInt32));

            push_bits32(&vm, KindInt32, 0);
            push_bits32(&vm, KindInt32, 100);
            push_bits32(&vm, KindInt32, 128);
            push_bits32(&vm, KindUint8, 8);
            execute_builtin(&vm, make_builtin_function(BuiltInSmoothStep, KindInt32));
            CHECK_EQUAL((s32)50, (s32)pop_bits32(&vm, KindInt32));

            push_bits64(&vm, KindInt64, 1000);
            push_bits64(&vm, KindInt64, 2000);
            push_bits64(&vm, KindInt64, 256);
            push_bits32(&vm, KindUint8, 9);
            execute_builtin(&vm, make_builtin_function(BuiltInLerp, KindInt64));
            CHECK_EQUAL((s64)1500, (s64)pop_bits64(&vm, KindInt64));

            push_bits64(&vm, KindInt64, 0);
            push_bits64(&vm, KindInt64, 100);
            push_bits64(&vm, KindInt64, 16384);
            push_bits32(&vm, KindUint8, 16);
            execute_builtin(&vm, make_builtin_function(BuiltInSmoothStep, KindInt64));
            CHECK_EQUAL((s64)50, (s64)pop_bits64(&vm, KindInt64));
        }

        UNITTEST_TEST(unsigned_abs_and_seeded_random)
        {
            byte frame[64] = {}, bss[16] = {}, external[16] = {}, data[16] = {}, stack[128] = {};
            call_frame_t call_frames[4] = {};
            vm_t vm;
            initialize_test_vm(&vm, call_frames, frame, bss, external, data, stack);

            push_bits32(&vm, KindUint8, 0x80);
            execute_builtin(&vm, make_builtin_function(BuiltInAbs, KindUint8));
            CHECK_EQUAL((u32)0x80, pop_bits32(&vm, KindUint8));

            set_random_seed(&vm, 12345);
            execute_builtin(&vm, make_builtin_function(BuiltInRandom, KindInt32));
            const u32 first = pop_bits32(&vm, KindInt32);
            execute_builtin(&vm, make_builtin_function(BuiltInRandom, KindInt32));
            const u32 second = pop_bits32(&vm, KindInt32);
            CHECK_TRUE(first <= 0x7fffffffU);
            CHECK_TRUE(second <= 0x7fffffffU);
            set_random_seed(&vm, 12345);
            execute_builtin(&vm, make_builtin_function(BuiltInRandom, KindInt32));
            CHECK_EQUAL(first, pop_bits32(&vm, KindInt32));
            execute_builtin(&vm, make_builtin_function(BuiltInRandom, KindInt32));
            CHECK_EQUAL(second, pop_bits32(&vm, KindInt32));
        }
    }
}
UNITTEST_SUITE_END
