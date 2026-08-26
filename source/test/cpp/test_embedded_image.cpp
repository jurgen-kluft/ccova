#include "ccore/c_target.h"

#include "ccova/vm.h"

#include "cunittest/cunittest.h"

extern unsigned char go_program_image[];
extern unsigned int  go_program_image_len;

using namespace ncore;

UNITTEST_SUITE_BEGIN(cova_embedded_image)
{
    UNITTEST_FIXTURE(main)
    {
        UNITTEST_FIXTURE_SETUP() {}
        UNITTEST_FIXTURE_TEARDOWN() {}

        UNITTEST_TEST(run_go_compiled_program)
        {
            CHECK_EQUAL((uptr_t)0, (uptr_t)go_program_image & 3U);

            byte         frame[64]      = {};
            byte         bss[16]        = {};
            byte         external[8]    = {};
            byte         data[16]       = {};
            byte         stack[64]      = {};
            call_frame_t call_frames[8] = {};
            vm_t         vm;
            initialize_vm(&vm, call_frames, 8, {frame, 64, 64}, {bss, 0, 16}, {external, 8, 8}, {data, 0, 16}, {stack, 0, 64});

            load_program_image(&vm, go_program_image, (u32)go_program_image_len);
            run_loaded_vm(&vm);
            CHECK_EQUAL((s32)6, (s32)pop_bits32(&vm, KindInt32));

            run_loaded_vm(&vm);
            CHECK_EQUAL((s32)6, (s32)pop_bits32(&vm, KindInt32));
        }
    }
}
UNITTEST_SUITE_END

static_assert(sizeof(go_program_image_len) == sizeof(u32), "embedded image length must fit u32");