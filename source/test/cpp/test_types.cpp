#include "ccore/c_target.h"

#include "ccova/types.h"

#include "cunittest/cunittest.h"

using namespace ncore;

UNITTEST_SUITE_BEGIN(cova_types)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        UNITTEST_TEST(instructions)
        {
            for (u32 opcode = OpPush; opcode < OpcodeCount; ++opcode)
            {
                const instruction_t instruction = make_instruction((eopcode_t)opcode, KindUint32);
                CHECK_EQUAL(opcode, (u32)instruction_opcode(instruction));
                CHECK_EQUAL((u32)KindUint32, (u32)instruction_kind(instruction));
            }

            for (u32 operation = ArithmeticAdd; operation <= ArithmeticShiftRight; ++operation)
            {
                const instruction_t instruction = make_arithmetic_instruction(KindInt16, (earithmeticop_t)operation);
                CHECK_EQUAL(operation, (u32)instruction_arithmetic_op(instruction));
            }

            for (u32 operation = CompareEqual; operation <= CompareGreaterEqual; ++operation)
            {
                const instruction_t instruction = make_compare_instruction(KindFloat32, (ecompareop_t)operation);
                CHECK_EQUAL(operation, (u32)instruction_compare_op(instruction));
            }

            const instruction_t conversion = make_convert_instruction(KindInt16, KindFloat64);
            CHECK_EQUAL((u32)KindInt16, (u32)instruction_convert_from_kind(conversion));
            CHECK_EQUAL((u32)KindFloat64, (u32)instruction_kind(conversion));

            const instruction_t address_instruction = make_address_instruction(SegmentExtern);
            CHECK_EQUAL((u32)SegmentExtern, (u32)instruction_address_segment(address_instruction));

            for (u32 function = 0; function <= 0x07ffU; ++function)
            {
                const instruction_t instruction = make_builtin_instruction((builtin_function_t)function);
                CHECK_EQUAL((u32)OpBuiltIn, (u32)instruction_opcode(instruction));
                CHECK_EQUAL(function, (u32)instruction_builtin_function(instruction));
            }

            const builtin_function_t sine = make_builtin_function(BuiltInSin, KindFloat64);
            CHECK_EQUAL((u32)BuiltInSin, (u32)builtin_function_operation(sine));
            CHECK_EQUAL((u32)KindFloat64, (u32)builtin_function_kind(sine));

            const address_t address = make_address(SegmentData, 0x00abcdefU);
            CHECK_EQUAL((u32)SegmentData, (u32)address_segment(address));
            CHECK_EQUAL((u32)0x00abcdefU, address_index(address));
        }

        UNITTEST_TEST(value_kind_transport_width)
        {
            for (u32 kind = KindBool; kind < KindCount; ++kind)
            {
                const evaluekind_t value_kind = (evaluekind_t)kind;
                if (value_kind_size(value_kind) <= 4)
                {
                    CHECK_TRUE(value_kind_is_32_bit(value_kind));
                    CHECK_FALSE(value_kind_is_64_bit(value_kind));
                }
                else
                {
                    CHECK_FALSE(value_kind_is_32_bit(value_kind));
                    CHECK_TRUE(value_kind_is_64_bit(value_kind));
                }
            }
        }
    }
}
UNITTEST_SUITE_END
