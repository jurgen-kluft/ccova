#ifndef __CCOVA_VM_H__
#define __CCOVA_VM_H__

#include "ccova/linked_program.h"
#include "ccova/program_memory.h"

namespace ncore
{
    struct vm_t;

    typedef void (*extern_dispatcher_fn)(void* host_context, vm_t* vm, u32 import_id);

    struct call_frame_t
    {                                   // sizeof(call_frame_t) = 8 bytes
        u32          m_local_base;      // Base index for local variables in the call frame
        u32          m_return_pc : 24;  // Return program counter for the call frame (byte offset)
        evaluekind_t m_return_kind : 8; // Return value kind for the call frame
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

    inline u8  pop_u8(vm_t* vm) { return (u8)pop_bits32(vm, KindUint8); }
    inline u16 pop_u16(vm_t* vm) { return (u16)pop_bits32(vm, KindUint16); }
    inline u32 pop_u32(vm_t* vm) { return (u32)pop_bits32(vm, KindUint32); }
    inline i32 pop_i32(vm_t* vm) { return (i32)pop_bits32(vm, KindInt32); }
    inline u64 pop_u64(vm_t* vm) { return (u64)pop_bits64(vm, KindUint64); }
    inline i64 pop_i64(vm_t* vm) { return (i64)pop_bits64(vm, KindInt64); }

    template <typename T> inline T* pop_data_pointer(vm_t* vm)
    {
        const address_t        address = pop_bits32(vm, KindAddress);
        const ememorysegment_t segment = address_segment(address);
        const u32              index   = address_index(address);
        return reinterpret_cast<T*>(vm->m_memory.m_segments[segment].m_data + index);
    }

} // namespace ncore

#endif
