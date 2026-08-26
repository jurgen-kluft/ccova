#include "ccore/c_target.h"

#include "ccova/byte_order.h"
#include "ccova/code_memory.h"

#include "cunittest/cunittest.h"

using namespace ncore;

UNITTEST_SUITE_BEGIN(cova_code_memory)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        UNITTEST_TEST(read)
        {
            byte code[16] = {};
            write_le_u16(code, make_instruction(OpPush, KindUint32));
            write_le_u32(code + 2, 0xa5b6c7d8U);
            write_le_u64(code + 6, 0x0123456789abcdefULL);

            const code_memory_t memory = {code, 14};
            u32                 offset = 0;

            CHECK_EQUAL((u32)OpPush, (u32)instruction_opcode(read_instruction(&memory, &offset)));
            CHECK_EQUAL((u32)2, offset);
            CHECK_EQUAL((u32)0xa5b6c7d8U, read_immediate32(&memory, &offset, KindUint32));
            CHECK_EQUAL((u32)6, offset);
            CHECK_EQUAL((u64)0x0123456789abcdefULL, read_immediate64(&memory, &offset, KindUint64));
            CHECK_EQUAL((u32)14, offset);
        }
    }
}
UNITTEST_SUITE_END
