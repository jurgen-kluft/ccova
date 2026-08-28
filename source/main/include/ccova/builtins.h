#ifndef __CCOVA_BUILTINS_H__
#define __CCOVA_BUILTINS_H__

#include "ccova/types.h"

namespace ncore
{
    struct vm_t;

    enum ebuiltinoperation_t : u8
    {
        BuiltInOperationInvalid = 0,
        BuiltInAbs,        // math::abs(value)
        BuiltInSin,        // math::sin(value)
        BuiltInCos,        // math::cos(value)
        BuiltInTan,        // math::tan(value)
        BuiltInAsin,       // math::asin(value)
        BuiltInAcos,       // math::acos(value)
        BuiltInAtan,       // math::atan(value)
        BuiltInPow,        // math::pow(base, exponent)
        BuiltInSqrt,       // math::sqrt(value)
        BuiltInMin,        // math::min(a, b)
        BuiltInMax,        // math::max(a, b)
        BuiltInMap,        // math::map(value, inMin, inMax, outMin, outMax)
        BuiltInRandom,     // math::random() -> int32
        BuiltInClamp,      // math::clamp(value, min, max)
        BuiltInSmoothStep, // math::smoothStep(edge0, edge1, x) or (start, end, t, shift)
        BuiltInLerp,       // math::lerp(a, b, t) or (start, end, t, shift)
        BuiltInSlerp,      // math::slerp(a, b, t)
    };

    typedef u16 builtin_function_t;

    void initialize_builtins(vm_t* vm);
    void reset_builtins(vm_t* vm);
    void set_random_seed(vm_t* vm, u64 seed);
    void execute_builtin(vm_t* vm, builtin_function_t function);

    inline ebuiltinoperation_t builtin_function_operation(builtin_function_t function) { return (ebuiltinoperation_t)((function >> 4) & 0x7fU); }
    inline evaluekind_t        builtin_function_kind(builtin_function_t function) { return (evaluekind_t)(function & 0x0fU); }
    inline builtin_function_t  instruction_builtin_function(instruction_t instruction) { return (builtin_function_t)((instruction >> 5) & 0x07ffU); }

    instruction_t      make_builtin_instruction(builtin_function_t function);
    builtin_function_t make_builtin_function(ebuiltinoperation_t operation, evaluekind_t kind);
} // namespace ncore

#endif
