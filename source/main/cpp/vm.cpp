#include "ccova/vm.h"
#include "ccova/builtins.h"
#include "ccova/code_memory.h"
#include "ccova/float_bits.h"
#include "ccova/image.h"

namespace ncore
{
    static bool s_validate_segment_storage(const segment_memory_t& segment)
    {
        if (segment.m_size > segment.m_capacity)
            return false;
        if (segment.m_data == nullptr && segment.m_capacity != 0)
            return false;
        return true;
    }

    static void s_clear_data(segment_memory_t* segment)
    {
        ASSERT(segment != nullptr);
        ASSERT(segment->m_data != nullptr || segment->m_size == 0);
        for (u32 index = 0; index < segment->m_size; ++index)
            segment->m_data[index] = 0;
    }

    static void s_copy_bytes(segment_memory_t* destination, const relative_array_t<byte>& source)
    {
        ASSERT(destination != nullptr);
        ASSERT(destination->m_size == source.m_size);
        ASSERT(destination->m_data != nullptr || destination->m_size == 0);
        ASSERT(source.data() != nullptr || source.m_size == 0);
        for (u32 index = 0; index < source.m_size; ++index)
            destination->m_data[index] = source[index];
    }

    static call_frame_t* s_current_frame(vm_t* vm)
    {
        ASSERT(vm->m_call_frame_count != 0);
        return &vm->m_call_frames[vm->m_call_frame_count - 1];
    }

    static void s_enter_script_function(vm_t* vm, u32 function_index, u32 return_pc, bool pop_arguments)
    {
        ASSERT(vm->m_program != nullptr);
        ASSERT(function_index < vm->m_program->m_functions.m_size);
        ASSERT(vm->m_call_frame_count < vm->m_call_frame_capacity);

        const script_function_t& function = vm->m_program->m_functions[function_index];
        ASSERT(function.m_param_start <= vm->m_program->m_param_kinds.m_size);
        ASSERT(function.m_param_count <= vm->m_program->m_param_kinds.m_size - function.m_param_start);
        ASSERT(function.m_param_start <= vm->m_program->m_param_offsets.m_size);
        ASSERT(function.m_param_count <= vm->m_program->m_param_offsets.m_size - function.m_param_start);
        ASSERT(vm->m_frame_top <= vm->m_memory.m_segments[SegmentFrame].m_capacity);
        ASSERT(function.m_frame_byte_size <= vm->m_memory.m_segments[SegmentFrame].m_capacity - vm->m_frame_top);

        const u32         local_base = vm->m_frame_top;
        const u32         frame_end  = local_base + function.m_frame_byte_size;
        segment_memory_t* frame      = &vm->m_memory.m_segments[SegmentFrame];
        for (u32 index = local_base; index < frame_end; ++index)
            frame->m_data[index] = 0;

        u32 argument_size = 0;
        for (u32 index = 0; index < function.m_param_count; ++index)
        {
            const u32 parameter_index = function.m_param_start + index;
            const u32 size            = value_kind_size(vm->m_program->m_param_kinds[parameter_index]);
            const u32 offset          = vm->m_program->m_param_offsets[parameter_index];
            ASSERT(size != 0);
            ASSERT(offset <= function.m_frame_byte_size);
            ASSERT(size <= function.m_frame_byte_size - offset);
            ASSERT(size <= 0xffffffffU - argument_size);
            argument_size += size;
            (void)offset;
        }
        (void)argument_size;
        if (pop_arguments)
        {
            segment_memory_t* stack = &vm->m_memory.m_segments[SegmentStack];
            ASSERT(argument_size <= stack->m_size);
            for (u32 index = function.m_param_count; index > 0; --index)
            {
                const u32 parameter_index = function.m_param_start + index - 1;
                truncate_to(stack, frame, local_base + vm->m_program->m_param_offsets[parameter_index], value_kind_size(vm->m_program->m_param_kinds[parameter_index]));
            }
        }

        call_frame_t* call_frame  = &vm->m_call_frames[vm->m_call_frame_count];
        call_frame->m_return_pc   = return_pc;
        call_frame->m_local_base  = local_base;
        call_frame->m_return_kind = function.m_return_kind;
        ++vm->m_call_frame_count;
        vm->m_frame_top = frame_end;
        vm->m_pc        = function.m_body_address;
    }

    void initialize_vm(vm_t* vm, call_frame_t* call_frames, u32 call_frame_capacity, const segment_memory_t& frame, const segment_memory_t& bss, const segment_memory_t& external, const segment_memory_t& data, const segment_memory_t& stack)
    {
        ASSERT(call_frames != nullptr);
        ASSERT(call_frame_capacity != 0);
        ASSERT(s_validate_segment_storage(frame));
        ASSERT(s_validate_segment_storage(bss));
        ASSERT(s_validate_segment_storage(external));
        ASSERT(s_validate_segment_storage(data));
        ASSERT(s_validate_segment_storage(stack));
        ASSERT(frame.m_size == frame.m_capacity);
        ASSERT(stack.m_size == 0);

        const segment_memory_t constant = {nullptr, 0, 0};
        initialize_program_memory(&vm->m_memory, frame, bss, external, constant, data, stack);
        vm->m_pc                  = 0;
        vm->m_program             = nullptr;
        vm->m_host_context        = nullptr;
        vm->m_extern_dispatcher   = nullptr;
        vm->m_call_frames         = call_frames;
        vm->m_call_frame_count    = 0;
        vm->m_call_frame_capacity = call_frame_capacity;
        vm->m_frame_top           = 0;
        initialize_builtins(vm);
    }

    void load_program(vm_t* vm, const linked_program_t* program)
    {
        validate_linked_program(program);

        segment_memory_t* bss  = &vm->m_memory.m_segments[SegmentBSS];
        segment_memory_t* data = &vm->m_memory.m_segments[SegmentData];
        ASSERT(program->m_bss_byte_size <= bss->m_capacity);
        ASSERT(program->m_data_data.m_size <= data->m_capacity);
        ASSERT(program->m_frame_byte_size <= vm->m_memory.m_segments[SegmentFrame].m_capacity);

        bss->m_size                = program->m_bss_byte_size;
        data->m_size               = program->m_data_data.m_size;
        segment_memory_t* constant = &vm->m_memory.m_segments[SegmentConst];
        constant->m_data           = const_cast<byte*>(program->m_const_data.data());
        constant->m_size           = program->m_const_data.m_size;
        constant->m_capacity       = program->m_const_data.m_size;
        vm->m_program              = program;
    }

    void load_program_image(vm_t* vm, const byte* block, u32 block_size)
    {
        load_program(vm, open_program_image(block, block_size));
    }

    void reset_vm(vm_t* vm)
    {
        ASSERT(vm->m_program != nullptr);

        s_clear_data(&vm->m_memory.m_segments[SegmentBSS]);
        s_copy_bytes(&vm->m_memory.m_segments[SegmentData], vm->m_program->m_data_data);
        vm->m_memory.m_segments[SegmentStack].m_size = 0;
        s_clear_data(&vm->m_memory.m_segments[SegmentFrame]);
        vm->m_pc               = 0;
        vm->m_call_frame_count = 0;
        vm->m_frame_top        = 0;
        reset_builtins(vm);

        const u32 entry_point = vm->m_program->m_entry_point;
        ASSERT(entry_point < vm->m_program->m_functions.m_size);
        ASSERT(vm->m_program->m_functions[entry_point].m_param_count == 0);
        s_enter_script_function(vm, entry_point, 0, false);
    }

    void register_extern_dispatcher(vm_t* vm, void* host_context, extern_dispatcher_fn dispatcher)
    {
        vm->m_host_context      = host_context;
        vm->m_extern_dispatcher = dispatcher;
    }

    void push_bits32(vm_t* vm, evaluekind_t kind, u32 bits)
    {
        ASSERT(value_kind_is_32_bit(kind));
        append_bits32(&vm->m_memory.m_segments[SegmentStack], kind, bits);
    }

    u32 pop_bits32(vm_t* vm, evaluekind_t kind)
    {
        ASSERT(value_kind_is_32_bit(kind));
        return truncate_bits32(&vm->m_memory.m_segments[SegmentStack], kind);
    }

    void push_bits64(vm_t* vm, evaluekind_t kind, u64 bits)
    {
        ASSERT(value_kind_is_64_bit(kind));
        append_bits64(&vm->m_memory.m_segments[SegmentStack], kind, bits);
    }

    u64 pop_bits64(vm_t* vm, evaluekind_t kind)
    {
        ASSERT(value_kind_is_64_bit(kind));
        return truncate_bits64(&vm->m_memory.m_segments[SegmentStack], kind);
    }

    template <typename T> static void push_number(vm_t* vm, evaluekind_t kind, T value)
    {
        switch (kind)
        {
            case KindBool: push_bits32(vm, KindBool, value != 0 ? 1 : 0); break;
            case KindByte: push_bits32(vm, KindByte, (u8)value); break;
            case KindInt8: push_bits32(vm, KindInt8, (u8)(s8)value); break;
            case KindInt16: push_bits32(vm, KindInt16, (u16)(s16)value); break;
            case KindInt32: push_bits32(vm, KindInt32, (u32)(s32)value); break;
            case KindInt64: push_bits64(vm, KindInt64, (u64)(s64)value); break;
            case KindUint8: push_bits32(vm, KindUint8, (u8)value); break;
            case KindUint16: push_bits32(vm, KindUint16, (u16)value); break;
            case KindUint32: push_bits32(vm, KindUint32, (u32)value); break;
            case KindUint64: push_bits64(vm, KindUint64, (u64)value); break;
            case KindFloat32: push_bits32(vm, KindFloat32, f32_to_bits((f32)value)); break;
            case KindFloat64: push_bits64(vm, KindFloat64, f64_to_bits((f64)value)); break;
            case KindAddress: push_bits32(vm, KindAddress, (u32)value); break;
            default: ASSERT(false); break;
        }
    }

    static void execute_conversion(vm_t* vm, evaluekind_t from, evaluekind_t to)
    {
        ASSERT(value_kind_size(to) != 0);
        switch (from)
        {
            case KindBool: push_number(vm, to, (u8)(pop_bits32(vm, KindBool) != 0)); break;
            case KindByte:
            case KindUint8: push_number(vm, to, (u8)pop_bits32(vm, from)); break;
            case KindInt8: push_number(vm, to, (s8)(u8)pop_bits32(vm, from)); break;
            case KindInt16: push_number(vm, to, (s16)(u16)pop_bits32(vm, from)); break;
            case KindInt32: push_number(vm, to, (s32)pop_bits32(vm, from)); break;
            case KindInt64: push_number(vm, to, (s64)pop_bits64(vm, from)); break;
            case KindUint16: push_number(vm, to, (u16)pop_bits32(vm, from)); break;
            case KindUint32:
            case KindAddress: push_number(vm, to, pop_bits32(vm, from)); break;
            case KindUint64: push_number(vm, to, pop_bits64(vm, from)); break;
            case KindFloat32: push_number(vm, to, bits_to_f32(pop_bits32(vm, from))); break;
            case KindFloat64: push_number(vm, to, bits_to_f64(pop_bits64(vm, from))); break;
            default: ASSERT(false); break;
        }
    }

    template <typename T> static bool compare_values(T left, T right, ecompareop_t operation)
    {
        switch (operation)
        {
            case CompareEqual: return left == right;
            case CompareNotEqual: return left != right;
            case CompareLess: return left < right;
            case CompareLessEqual: return left <= right;
            case CompareGreater: return left > right;
            case CompareGreaterEqual: return left >= right;
            default: ASSERT(false); return false;
        }
    }

    template <typename T> static bool pop_and_compare32(vm_t* vm, evaluekind_t kind, ecompareop_t operation)
    {
        const T right = (T)pop_bits32(vm, kind);
        const T left  = (T)pop_bits32(vm, kind);
        return compare_values(left, right, operation);
    }

    template <typename T> static bool pop_and_compare64(vm_t* vm, evaluekind_t kind, ecompareop_t operation)
    {
        const T right = (T)pop_bits64(vm, kind);
        const T left  = (T)pop_bits64(vm, kind);
        return compare_values(left, right, operation);
    }

    static bool execute_comparison(vm_t* vm, evaluekind_t kind, ecompareop_t operation)
    {
        switch (kind)
        {
            case KindBool:
            {
                ASSERT(operation == CompareEqual || operation == CompareNotEqual);
                const bool right = pop_bits32(vm, KindBool) != 0;
                const bool left  = pop_bits32(vm, KindBool) != 0;
                return compare_values(left, right, operation);
            }
            case KindByte:
            case KindUint8: return pop_and_compare32<u8>(vm, kind, operation);
            case KindInt8: return pop_and_compare32<s8>(vm, kind, operation);
            case KindInt16: return pop_and_compare32<s16>(vm, kind, operation);
            case KindInt32: return pop_and_compare32<s32>(vm, kind, operation);
            case KindInt64: return pop_and_compare64<s64>(vm, kind, operation);
            case KindUint16: return pop_and_compare32<u16>(vm, kind, operation);
            case KindUint32:
            case KindAddress: return pop_and_compare32<u32>(vm, kind, operation);
            case KindUint64: return pop_and_compare64<u64>(vm, kind, operation);
            case KindFloat32:
            {
                const f32 right = bits_to_f32(pop_bits32(vm, kind));
                const f32 left  = bits_to_f32(pop_bits32(vm, kind));
                return compare_values(left, right, operation);
            }
            case KindFloat64:
            {
                const f64 right = bits_to_f64(pop_bits64(vm, kind));
                const f64 left  = bits_to_f64(pop_bits64(vm, kind));
                return compare_values(left, right, operation);
            }
            default: ASSERT(false); return false;
        }
    }

    template <typename T, typename U> static U integer_arithmetic(T left, T right, earithmeticop_t operation, bool is_signed)
    {
        const U unsigned_left  = (U)left;
        const U unsigned_right = (U)right;
        switch (operation)
        {
            case ArithmeticAdd: return (U)(unsigned_left + unsigned_right);
            case ArithmeticSub: return (U)(unsigned_left - unsigned_right);
            case ArithmeticMul: return (U)(unsigned_left * unsigned_right);
            case ArithmeticDiv:
                ASSERT(right != 0);
                if (is_signed && right == (T)-1 && left == (T)((U)1 << (sizeof(T) * 8 - 1)))
                    return unsigned_left;
                return (U)(left / right);
            case ArithmeticModulo:
                ASSERT(right != 0);
                if (is_signed && right == (T)-1 && left == (T)((U)1 << (sizeof(T) * 8 - 1)))
                    return 0;
                return (U)(left % right);
            case ArithmeticBitwiseAnd: return (U)(unsigned_left & unsigned_right);
            case ArithmeticBitwiseOr: return (U)(unsigned_left | unsigned_right);
            case ArithmeticBitwiseXor: return (U)(unsigned_left ^ unsigned_right);
            case ArithmeticShiftLeft:
            {
                const u32 shift = (u32)(unsigned_right & (sizeof(T) * 8 - 1));
                return (U)(unsigned_left << shift);
            }
            case ArithmeticShiftRight:
            {
                const u32 shift = (u32)(unsigned_right & (sizeof(T) * 8 - 1));
                if (!is_signed || left >= 0 || shift == 0)
                    return (U)(unsigned_left >> shift);
                return (U)((unsigned_left >> shift) | ((U) ~(U)0 << (sizeof(T) * 8 - shift)));
            }
            default: ASSERT(false); return 0;
        }
    }

    template <typename T> static T float_arithmetic(T left, T right, earithmeticop_t operation)
    {
        switch (operation)
        {
            case ArithmeticAdd: return left + right;
            case ArithmeticSub: return left - right;
            case ArithmeticMul: return left * right;
            case ArithmeticDiv: ASSERT(right != 0); return left / right;
            default: ASSERT(false); return 0;
        }
    }

    static void execute_arithmetic(vm_t* vm, evaluekind_t kind, earithmeticop_t operation)
    {
        if (value_kind_is_64_bit(kind))
        {
            const u64 right_bits = pop_bits64(vm, kind);
            const u64 left_bits  = pop_bits64(vm, kind);
            switch (kind)
            {
                case KindInt64: push_bits64(vm, kind, integer_arithmetic<s64, u64>((s64)left_bits, (s64)right_bits, operation, true)); break;
                case KindUint64: push_bits64(vm, kind, integer_arithmetic<u64, u64>(left_bits, right_bits, operation, false)); break;
                case KindFloat64: push_bits64(vm, kind, f64_to_bits(float_arithmetic(bits_to_f64(left_bits), bits_to_f64(right_bits), operation))); break;
                default: ASSERT(false); break;
            }
        }
        else
        {
            const u32 right_bits = pop_bits32(vm, kind);
            const u32 left_bits  = pop_bits32(vm, kind);
            switch (kind)
            {
                case KindBool:
                case KindByte:
                case KindUint8: push_bits32(vm, kind, integer_arithmetic<u8, u8>((u8)left_bits, (u8)right_bits, operation, false)); break;
                case KindInt8: push_bits32(vm, kind, integer_arithmetic<s8, u8>((s8)(u8)left_bits, (s8)(u8)right_bits, operation, true)); break;
                case KindInt16: push_bits32(vm, kind, integer_arithmetic<s16, u16>((s16)(u16)left_bits, (s16)(u16)right_bits, operation, true)); break;
                case KindInt32: push_bits32(vm, kind, integer_arithmetic<s32, u32>((s32)left_bits, (s32)right_bits, operation, true)); break;
                case KindUint16: push_bits32(vm, kind, integer_arithmetic<u16, u16>((u16)left_bits, (u16)right_bits, operation, false)); break;
                case KindUint32: push_bits32(vm, kind, integer_arithmetic<u32, u32>(left_bits, right_bits, operation, false)); break;
                case KindFloat32: push_bits32(vm, kind, f32_to_bits(float_arithmetic(bits_to_f32(left_bits), bits_to_f32(right_bits), operation))); break;
                default: ASSERT(false); break;
            }
        }
    }

    static void push_from_memory(vm_t* vm, address_t address, evaluekind_t kind)
    {
        const ememorysegment_t segment_index = address_segment(address);
        ASSERT(segment_index >= SegmentFrame && segment_index <= SegmentStack);
        const segment_memory_t* segment = &vm->m_memory.m_segments[(u32)segment_index];
        const u32               offset  = address_index(address);
        switch (value_kind_size(kind))
        {
            case 1: push_bits32(vm, kind, read_u8(segment, offset)); break;
            case 2: push_bits32(vm, kind, read_u16(segment, offset)); break;
            case 4: push_bits32(vm, kind, read_u32(segment, offset)); break;
            case 8: push_bits64(vm, kind, read_u64(segment, offset)); break;
            default: ASSERT(false); break;
        }
    }

    static void pop_to_memory(vm_t* vm, address_t address, evaluekind_t kind)
    {
        const ememorysegment_t segment_index = address_segment(address);
        ASSERT(segment_index >= SegmentFrame && segment_index <= SegmentStack);
        ASSERT(segment_index != SegmentConst);
        segment_memory_t* segment = &vm->m_memory.m_segments[(u32)segment_index];
        const u32         offset  = address_index(address);
        if (value_kind_is_64_bit(kind))
        {
            write_u64(segment, offset, pop_bits64(vm, kind));
        }
        else
        {
            const u32 value = pop_bits32(vm, kind);
            switch (value_kind_size(kind))
            {
                case 1: write_u8(segment, offset, (u8)value); break;
                case 2: write_u16(segment, offset, (u16)value); break;
                case 4: write_u32(segment, offset, value); break;
                default: ASSERT(false); break;
            }
        }
    }

    static bool return_from_function(vm_t* vm)
    {
        ASSERT(vm->m_call_frame_count != 0);
        --vm->m_call_frame_count;
        const call_frame_t& frame = vm->m_call_frames[vm->m_call_frame_count];
        if (frame.m_return_kind != KindNone && frame.m_return_kind != KindVoid)
            ASSERT(value_kind_size(frame.m_return_kind) <= vm->m_memory.m_segments[SegmentStack].m_size);
        vm->m_frame_top = frame.m_local_base;
        if (vm->m_call_frame_count == 0)
            return true;
        vm->m_pc = frame.m_return_pc;
        return false;
    }

    void run_vm(vm_t* vm, const linked_program_t* program)
    {
        load_program(vm, program);
        run_loaded_vm(vm);
    }

    void run_vm_image(vm_t* vm, const byte* block, u32 block_size)
    {
        load_program_image(vm, block, block_size);
        run_loaded_vm(vm);
    }

    void run_loaded_vm(vm_t* vm)
    {
        reset_vm(vm);
        const code_memory_t text = {vm->m_program->m_text.data(), vm->m_program->m_text.m_size};
        while (vm->m_pc < text.m_size)
        {
            const instruction_t instruction = read_instruction(&text, &vm->m_pc);
            const eopcode_t     opcode      = instruction_opcode(instruction);
            switch (opcode)
            {
                case OpPush:
                {
                    const evaluekind_t kind = instruction_kind(instruction);
                    ASSERT(kind != KindNone && kind != KindVoid && kind != KindAddress);
                    if (value_kind_is_32_bit(kind))
                        push_bits32(vm, kind, read_immediate32(&text, &vm->m_pc, kind));
                    else
                        push_bits64(vm, kind, read_immediate64(&text, &vm->m_pc, kind));
                    break;
                }
                case OpArithmetic: execute_arithmetic(vm, instruction_kind(instruction), instruction_arithmetic_op(instruction)); break;
                case OpBuiltIn: execute_builtin(vm, instruction_builtin_function(instruction)); break;
                case OpConvert: execute_conversion(vm, instruction_convert_from_kind(instruction), instruction_kind(instruction)); break;
                case OpAddr:
                {
                    const ememorysegment_t segment = instruction_address_segment(instruction);
                    ASSERT(segment >= SegmentFrame && segment <= SegmentStack);
                    u32 offset = read_u32(&text, &vm->m_pc);
                    if (segment == SegmentFrame)
                    {
                        const u32 local_base = s_current_frame(vm)->m_local_base;
                        ASSERT(offset <= AddressIndexMask - local_base);
                        offset += local_base;
                    }
                    ASSERT(offset <= AddressIndexMask);
                    push_bits32(vm, KindAddress, make_address(segment, offset));
                    break;
                }
                case OpOffset:
                {
                    const s32       offset = (s32)pop_bits32(vm, KindInt32);
                    const address_t base   = (address_t)pop_bits32(vm, KindAddress);
                    const s64       index  = (s64)address_index(base) + offset;
                    ASSERT(index >= 0 && index <= AddressIndexMask);
                    push_bits32(vm, KindAddress, make_address(address_segment(base), (u32)index));
                    break;
                }
                case OpDereference:
                {
                    const evaluekind_t kind = instruction_kind(instruction);
                    ASSERT(value_kind_size(kind) != 0);
                    push_from_memory(vm, (address_t)pop_bits32(vm, KindAddress), kind);
                    break;
                }
                case OpAssign:
                {
                    const evaluekind_t kind = instruction_kind(instruction);
                    ASSERT(value_kind_size(kind) != 0);
                    pop_to_memory(vm, (address_t)pop_bits32(vm, KindAddress), kind);
                    break;
                }
                case OpCompare: push_bits32(vm, KindBool, execute_comparison(vm, instruction_kind(instruction), instruction_compare_op(instruction)) ? 1 : 0); break;
                case OpJumpIfFalse:
                {
                    const u32 target = read_u32(&text, &vm->m_pc);
                    if (pop_bits32(vm, KindBool) == 0)
                    {
                        ASSERT(target < text.m_size);
                        vm->m_pc = target;
                    }
                    break;
                }
                case OpJump:
                {
                    const u32 target = read_u32(&text, &vm->m_pc);
                    ASSERT(target < text.m_size);
                    vm->m_pc = target;
                    break;
                }
                case OpCall:
                {
                    const u32 target    = read_u32(&text, &vm->m_pc);
                    const u32 return_pc = vm->m_pc;
                    s_enter_script_function(vm, target, return_pc, true);
                    break;
                }
                case OpCallExtern:
                {
                    const u32 import_id = read_u32(&text, &vm->m_pc);
                    ASSERT(vm->m_extern_dispatcher != nullptr);
                    vm->m_extern_dispatcher(vm->m_host_context, vm, import_id);
                    break;
                }
                case OpRet:
                    if (return_from_function(vm))
                        return;
                    break;
                default: ASSERT(false); return;
            }
        }
    }
} // namespace ncore
