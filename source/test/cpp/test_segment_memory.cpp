#include "ccore/c_target.h"

#include "ccova/segment_memory.h"

#include "cunittest/cunittest.h"

using namespace ncore;

UNITTEST_SUITE_BEGIN(cova_segment_memory)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        UNITTEST_TEST(append_and_truncate)
        {
            byte             storage[32] = {};
            segment_memory_t segment     = {storage, 0, 32};

            append_u8(&segment, 0x12U);
            append_u16(&segment, 0x3456U);
            append_u32(&segment, 0x789abcdeU);
            append_u64(&segment, 0x0123456789abcdefULL);

            CHECK_EQUAL((u32)15, segment.m_size);
            CHECK_EQUAL((u64)0x0123456789abcdefULL, truncate_u64(&segment));
            CHECK_EQUAL((u32)0x789abcdeU, truncate_u32(&segment));
            CHECK_EQUAL((u16)0x3456U, truncate_u16(&segment));
            CHECK_EQUAL((u8)0x12U, truncate_u8(&segment));
            CHECK_EQUAL((u32)0, segment.m_size);
        }

        UNITTEST_TEST(bit_transport_preserves_compact_sizes)
        {
            byte             storage[32] = {};
            segment_memory_t segment     = {storage, 0, 32};

            append_bits32(&segment, KindInt8, 0x81U);
            CHECK_EQUAL((u32)1, segment.m_size);
            append_bits32(&segment, KindUint16, 0x2345U);
            CHECK_EQUAL((u32)3, segment.m_size);
            append_bits32(&segment, KindFloat32, 0x3f800000U);
            CHECK_EQUAL((u32)7, segment.m_size);
            append_bits64(&segment, KindUint64, 0x0123456789abcdefULL);
            CHECK_EQUAL((u32)15, segment.m_size);

            CHECK_EQUAL((u64)0x0123456789abcdefULL, truncate_bits64(&segment, KindUint64));
            CHECK_EQUAL((u32)0x3f800000U, truncate_bits32(&segment, KindFloat32));
            CHECK_EQUAL((u32)0x2345U, truncate_bits32(&segment, KindUint16));
            CHECK_EQUAL((u32)0x81U, truncate_bits32(&segment, KindInt8));
            CHECK_EQUAL((u32)0, segment.m_size);
        }

        UNITTEST_TEST(read_write_and_copy)
        {
            byte             storage[32] = {};
            segment_memory_t segment     = {storage, 16, 32};

            write_u8(&segment, 0, 0xffU);
            write_u32(&segment, 4, 0x12345678U);
            write_u64(&segment, 8, 0x0102030405060708ULL);

            CHECK_EQUAL((u8)0xffU, read_u8(&segment, 0));
            CHECK_EQUAL((u32)0x12345678U, read_u32(&segment, 4));
            CHECK_EQUAL((u64)0x0102030405060708ULL, read_u64(&segment, 8));

            byte             destination_storage[8] = {};
            segment_memory_t destination            = {destination_storage, 8, 8};

            truncate_to(&segment, &destination, 0, 4);
            CHECK_EQUAL((u32)12, segment.m_size);
            CHECK_EQUAL((u32)0x01020304U, read_u32(&destination, 0));

            append_from(&segment, &destination, 0, 4);
            CHECK_EQUAL((u32)16, segment.m_size);
            CHECK_EQUAL((u32)0x01020304U, read_u32(&segment, 12));
        }
    }
}
UNITTEST_SUITE_END
