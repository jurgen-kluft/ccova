#ifndef __CCOVA_IMAGE_H__
#define __CCOVA_IMAGE_H__

#include "ccova/linked_program.h"

namespace ncore
{
    static const u32 ProgramImageMagic        = 0x41564f43U;
    static const u32 ProgramImageVersion      = 0x00010004U;
    static const u8  ProgramImageEndianLittle = 1;
    static const u8  ProgramImageABI          = 1;
    static const u32 ProgramImageHeaderSize   = 76;
    static const u32 ProgramImageFunctionSize = 20;

    const linked_program_t* open_program_image(const byte* block, u32 block_size);
} // namespace ncore

#endif