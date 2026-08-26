#ifndef __CCOVA_BUILTINS_H__
#define __CCOVA_BUILTINS_H__

#include "ccova/types.h"

namespace ncore
{
    struct vm_t;

    void initialize_builtins(vm_t* vm);
    void reset_builtins(vm_t* vm);
    void set_random_seed(vm_t* vm, u64 seed);
    void execute_builtin(vm_t* vm, builtin_function_t function);
} // namespace ncore

#endif
