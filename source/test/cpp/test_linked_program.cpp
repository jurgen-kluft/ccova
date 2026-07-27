#include "ccore/c_target.h"

#include "ccova/image.h"

#include "cunittest/cunittest.h"

using namespace ncore;

UNITTEST_SUITE_BEGIN(cova_linked_program)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        UNITTEST_TEST(validate)
        {
            const byte              text[]       = {OpRet, 0};
            const script_function_t functions[] = {{0, 0, 0, 0, KindVoid}};
            linked_program_t program = {};
            program.m_magic           = ProgramImageMagic;
            program.m_version         = ProgramImageVersion;
            program.m_endian          = ProgramImageEndianLittle;
            program.m_abi             = ProgramImageABI;
            program.m_functions.set(functions, 1);
            program.m_param_kinds.set(nullptr, 0);
            program.m_param_offsets.set(nullptr, 0);
            program.m_text.set(text, 2);
            program.m_const_data.set(nullptr, 0);
            program.m_data_data.set(nullptr, 0);

            validate_linked_program(&program);
            CHECK_EQUAL((u32)1, program.m_functions.m_size);
            CHECK_EQUAL((u32)0, program.m_entry_point);
        }

        UNITTEST_TEST(resolve_negative_relative_offset)
        {
            struct fixture_t
            {
                byte                   m_data[4];
                relative_array_t<byte> m_array;
            } fixture = {{1, 2, 3, 4}, {}};

            fixture.m_array.set(fixture.m_data, 4);

            CHECK_TRUE(fixture.m_array.m_data.m_offset < 0);
            CHECK_TRUE(fixture.m_array.data() == fixture.m_data);
            CHECK_EQUAL((u8)3, (u8)fixture.m_array[2]);
        }

        UNITTEST_TEST(relative_array_layout_and_empty_value)
        {
            struct fixture_t
            {
                relative_array_t<byte> m_array;
                byte                   m_data[4];
            } fixture = {{}, {1, 2, 3, 4}};

            CHECK_EQUAL((uptr_t)0, (uptr_t)&fixture.m_array.m_data - (uptr_t)&fixture.m_array);
            CHECK_EQUAL((uptr_t)4, (uptr_t)&fixture.m_array.m_size - (uptr_t)&fixture.m_array);

            fixture.m_array.set(fixture.m_data, 4);
            CHECK_TRUE(fixture.m_array.m_data.m_offset > 0);
            CHECK_TRUE(fixture.m_array.data() == fixture.m_data);
            CHECK_EQUAL((u32)4, fixture.m_array.m_size);

            fixture.m_array.set(nullptr, 0);
            CHECK_EQUAL((s32)0, fixture.m_array.m_data.m_offset);
            CHECK_EQUAL((u32)0, fixture.m_array.m_size);
            CHECK_TRUE(fixture.m_array.data() == nullptr);
        }
    }
}
UNITTEST_SUITE_END
