#ifndef __CCOVA_VM_H__
#define __CCOVA_VM_H__

#include "ccova/linked_program.h"
#include "ccova/program_memory.h"

namespace ncore
{
    struct vm_t;

    typedef void (*extern_dispatcher_fn)(void* host_context, vm_t* vm, u32 import_id);

    struct call_frame_t
    {
        u32          m_return_pc;
        u32          m_local_base;
        evaluekind_t m_return_kind;
    };

    struct vm_t
    {
        program_memory_t        m_memory;
        u32                     m_pc;
        const linked_program_t* m_program;
        void*                   m_host_context;
        extern_dispatcher_fn    m_extern_dispatcher;
        call_frame_t*           m_call_frames;
        u32                     m_call_frame_count;
        u32                     m_call_frame_capacity;
        u32                     m_frame_top;
        u64                     m_random_seed;
        u64                     m_random_s0;
        u64                     m_random_s1;
    };

    void initialize_vm(vm_t* vm, call_frame_t* call_frames, u32 call_frame_capacity, const segment_memory_t& frame, const segment_memory_t& bss, const segment_memory_t& external, const segment_memory_t& data, const segment_memory_t& stack);
    void load_program(vm_t* vm, const linked_program_t* program);
    void load_program_image(vm_t* vm, const byte* block, u32 block_size);
    void reset_vm(vm_t* vm);
    void register_extern_dispatcher(vm_t* vm, void* host_context, extern_dispatcher_fn dispatcher);

    void run_vm(vm_t* vm, const linked_program_t* program);
    void run_vm_image(vm_t* vm, const byte* block, u32 block_size);
    void run_loaded_vm(vm_t* vm);

    void push_bits32(vm_t* vm, evaluekind_t kind, u32 bits);
    u32  pop_bits32(vm_t* vm, evaluekind_t kind);
    void push_bits64(vm_t* vm, evaluekind_t kind, u64 bits);
    u64  pop_bits64(vm_t* vm, evaluekind_t kind);
} // namespace ncore

#endif
