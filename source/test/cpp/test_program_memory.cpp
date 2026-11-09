#include "ccore/c_target.h"

#include "ccova/program_memory.h"

#include "cunittest/cunittest.h"

using namespace ncore;

UNITTEST_SUITE_BEGIN(cova_program_memory)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        UNITTEST_TEST(routes_by_address_segment)
        {
            byte frame_storage[8]  = {};
            byte bss_storage[8]    = {};
            byte extern_storage[8] = {};
            byte const_storage[8]  = {};
            byte data_storage[8]   = {};
            byte stack_storage[8]  = {};

            program_memory_t memory;
            const segment_memory_t externals[ExternMemoryBankCount] = {{extern_storage, 8, 8}, {nullptr, 0, 0}, {nullptr, 0, 0}, {nullptr, 0, 0}};
            initialize_program_memory(&memory, {frame_storage, 8, 8}, {bss_storage, 8, 8}, externals, {const_storage, 8, 8}, {data_storage, 8, 8}, {stack_storage, 0, 8});

            write_u8(&memory, make_address(SegmentExtern0, 0), 11);
            write_u8(&memory, make_address(SegmentBSS, 1), 12);
            write_u8(&memory, make_address(SegmentFrame, 2), 13);

            CHECK_EQUAL((u8)11, extern_storage[0]);
            CHECK_EQUAL((u8)12, bss_storage[1]);
            CHECK_EQUAL((u8)13, frame_storage[2]);
            CHECK_EQUAL((u8)11, read_u8(&memory, make_address(SegmentExtern0, 0)));
            CHECK_EQUAL((u8)12, read_u8(&memory, make_address(SegmentBSS, 1)));
            CHECK_EQUAL((u8)13, read_u8(&memory, make_address(SegmentFrame, 2)));
        }

        UNITTEST_TEST(typed_read_write)
        {
            byte                   storage[64]      = {};
            byte                   const_storage[8] = {0x78, 0x56, 0x34, 0x12};
            program_memory_t       memory;
            const segment_memory_t empty                              = {nullptr, 0, 0};
            const segment_memory_t externals[ExternMemoryBankCount] = {{storage, 64, 64}, empty, empty, empty};
            initialize_program_memory(&memory, empty, empty, externals, {const_storage, 8, 8}, empty, empty);

            const address_t external = make_address(SegmentExtern0, 0);

            write_bool(&memory, external, true);
            CHECK_TRUE(read_bool(&memory, external));
            CHECK_EQUAL((u8)1, read_u8(&memory, external));
            write_u8(&memory, external + 1, 2);
            CHECK_TRUE(read_bool(&memory, external + 1));

            write_u8(&memory, external + 2, 0xa5U);
            write_s8(&memory, external + 3, (s8)-12);
            write_u8(&memory, external + 4, 241U);
            CHECK_EQUAL((u8)0xa5U, read_u8(&memory, external + 2));
            CHECK_EQUAL((s8)-12, read_s8(&memory, external + 3));
            CHECK_EQUAL((u8)241U, read_u8(&memory, external + 4));

            write_s16(&memory, external + 6, (s16)-12345);
            write_u16(&memory, external + 8, 54321U);
            CHECK_EQUAL((s16)-12345, read_s16(&memory, external + 6));
            CHECK_EQUAL((u16)54321U, read_u16(&memory, external + 8));

            write_s32(&memory, external + 12, (s32)-123456789);
            write_u32(&memory, external + 16, 0xfedcba98U);
            write_f32(&memory, external + 20, (f32)-3.75);
            CHECK_EQUAL((s32)-123456789, read_s32(&memory, external + 12));
            CHECK_EQUAL((u32)0xfedcba98U, read_u32(&memory, external + 16));
            CHECK_EQUAL((f32)-3.75, read_f32(&memory, external + 20));

            const address_t stored_address = make_address(SegmentData, 0x1234U);
            write_address(&memory, external + 24, stored_address);
            CHECK_EQUAL(stored_address, read_address(&memory, external + 24));

            write_s64(&memory, external + 28, (s64)-0x0102030405060708LL);
            write_u64(&memory, external + 36, 0xfedcba9876543210ULL);
            write_f64(&memory, external + 44, (f64)(1.0 / 3.0));
            CHECK_EQUAL((s64)-0x0102030405060708LL, read_s64(&memory, external + 28));
            CHECK_EQUAL((u64)0xfedcba9876543210ULL, read_u64(&memory, external + 36));
            CHECK_EQUAL((f64)(1.0 / 3.0), read_f64(&memory, external + 44));

            CHECK_EQUAL((u32)0x12345678U, read_u32(&memory, make_address(SegmentConst, 0)));
        }
    }
}
UNITTEST_SUITE_END
