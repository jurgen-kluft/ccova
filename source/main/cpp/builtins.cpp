#include "ccova/builtins.h"
#include "ccova/float_bits.h"
#include "ccova/vm.h"

#include <cmath>

namespace ncore
{
    static const u64 scDefaultRandomSeed = 0x1234567890abcdefULL;
    static const u64 scRandomStateSeed   = 6364136223846793005ULL;

    static inline u64 next_random(vm_t* vm)
    {
        u64       s1     = vm->m_random_s0;
        const u64 s0     = vm->m_random_s1;
        const u64 result = s0 + s1;
        vm->m_random_s0  = s0;
        s1 ^= s1 << 23;
        vm->m_random_s1 = s1 ^ s0 ^ (s1 >> 18) ^ (s0 >> 5);
        return result;
    }

    void reset_builtins(vm_t* vm)
    {
        ASSERT(vm != nullptr);
        vm->m_random_s0 = vm->m_random_seed + 6364136223846793ULL;
        vm->m_random_s1 = scRandomStateSeed;
        next_random(vm);
        next_random(vm);
    }

    void initialize_builtins(vm_t* vm)
    {
        ASSERT(vm != nullptr);
        vm->m_random_seed = scDefaultRandomSeed;
        reset_builtins(vm);
    }

    void set_random_seed(vm_t* vm, u64 seed)
    {
        ASSERT(vm != nullptr);
        vm->m_random_seed = seed;
        reset_builtins(vm);
    }

    static void execute_builtin_abs(vm_t* vm, evaluekind_t kind)
    {
        if (value_kind_is_64_bit(kind))
        {
            u64 bits = pop_bits64(vm, kind);
            switch (kind)
            {
                case KindUint64: break;
                case KindInt64:
                    if ((bits & 0x8000000000000000ULL) != 0)
                        bits = 0ULL - bits;
                    break;
                case KindFloat64: bits = f64_to_bits(std::fabs(bits_to_f64(bits))); break;
                default: ASSERT(false); break;
            }
            push_bits64(vm, kind, bits);
        }
        else // if (value_kind_is_32_bit(kind))
        {
            u32 bits = pop_bits32(vm, kind);
            switch (kind)
            {
                case KindByte:
                case KindUint8:
                case KindUint16:
                case KindUint32: break;
                case KindInt8:
                    if ((bits & 0x80U) != 0)
                        bits = (u8)(0U - (u8)bits);
                    break;
                case KindInt16:
                    if ((bits & 0x8000U) != 0)
                        bits = (u16)(0U - (u16)bits);
                    break;
                case KindInt32:
                    if ((bits & 0x80000000U) != 0)
                        bits = (u32)(0U - (u32)bits);
                    break;
                case KindFloat32: bits = f32_to_bits((f32)std::fabs((f64)bits_to_f32(bits))); break;
                default: ASSERT(false); break;
            }
            push_bits32(vm, kind, bits);
        }
    }

    template <typename T> static bool builtin_min_or_max_fn(ebuiltinoperation_t operation, T left, T right) { return operation == BuiltInMin ? right < left : right > left; }

    static void execute_builtin_min_or_max_32(vm_t* vm, ebuiltinoperation_t operation, evaluekind_t kind)
    {
        ASSERT(operation == BuiltInMin || operation == BuiltInMax);
        ASSERT(value_kind_is_32_bit(kind));

        const u32 right_bits   = pop_bits32(vm, kind);
        const u32 left_bits    = pop_bits32(vm, kind);
        bool      choose_right = false;
        switch (kind)
        {
            case KindByte:
            case KindUint8:
            case KindUint16:
            case KindUint32: choose_right = builtin_min_or_max_fn(operation, left_bits, right_bits); break;
            case KindInt8: choose_right = builtin_min_or_max_fn(operation, (i32)(s8)(u8)left_bits, (i32)(s8)(u8)right_bits); break;
            case KindInt16: choose_right = builtin_min_or_max_fn(operation, (i32)(s16)(u16)left_bits, (i32)(s16)(u16)right_bits); break;
            case KindInt32: choose_right = builtin_min_or_max_fn(operation, (i32)left_bits, (i32)right_bits); break;
            case KindFloat32: choose_right = builtin_min_or_max_fn(operation, bits_to_f32(left_bits), bits_to_f32(right_bits)); break;
            default: ASSERT(false); break;
        }
        push_bits32(vm, kind, choose_right ? right_bits : left_bits);
    }

    static void execute_builtin_min_or_max_64(vm_t* vm, ebuiltinoperation_t operation, evaluekind_t kind)
    {
        ASSERT(operation == BuiltInMin || operation == BuiltInMax);
        ASSERT(value_kind_is_64_bit(kind));

        const u64 right_bits   = pop_bits64(vm, kind);
        const u64 left_bits    = pop_bits64(vm, kind);
        bool      choose_right = false;
        switch (kind)
        {
            case KindInt64: choose_right = builtin_min_or_max_fn(operation, (s64)left_bits, (s64)right_bits); break;
            case KindUint64: choose_right = builtin_min_or_max_fn(operation, left_bits, right_bits); break;
            case KindFloat64: choose_right = builtin_min_or_max_fn(operation, bits_to_f64(left_bits), bits_to_f64(right_bits)); break;
            default: ASSERT(false); break;
        }
        push_bits64(vm, kind, choose_right ? right_bits : left_bits);
    }

    template <typename T> static T clamp_value(T value, T low, T high)
    {
        ASSERT(low <= high);
        return value < low ? low : value > high ? high : value;
    }

    template <typename T> static void sort_bounds(T& low, T& high)
    {
        if (low > high)
        {
            const T swap = low;
            low          = high;
            high         = swap;
        }
    }

    template <typename T> static T map_value(T value, T in_low, T in_high, T out_low, T out_high)
    {
        sort_bounds(in_low, in_high);
        sort_bounds(out_low, out_high);
        ASSERT(in_low < in_high);
        value = clamp_value(value, in_low, in_high);
        return (value - in_low) * (out_high - out_low) / (in_high - in_low) + out_low;
    }

    static i32 bits_to_i32(u32 bits, evaluekind_t kind)
    {
        switch (kind)
        {
            case KindInt8: return (i32)(s8)(u8)bits;
            case KindInt16: return (i32)(s16)(u16)bits;
            case KindInt32: return (i32)bits;
            default: ASSERT(false); return (i32)bits;
        }
    }

    static void execute_builtin_clamp_i32(vm_t* vm, evaluekind_t kind)
    {
        const i32 high  = bits_to_i32(pop_bits32(vm, kind), kind);
        const i32 low   = bits_to_i32(pop_bits32(vm, kind), kind);
        const i32 value = bits_to_i32(pop_bits32(vm, kind), kind);
        push_bits32(vm, kind, (u32)clamp_value(value, low, high));
    }

    static void execute_builtin_clamp_u32(vm_t* vm, evaluekind_t kind)
    {
        const u32 high  = pop_bits32(vm, kind);
        const u32 low   = pop_bits32(vm, kind);
        const u32 value = pop_bits32(vm, kind);
        push_bits32(vm, kind, clamp_value(value, low, high));
    }

    template <typename T> static void execute_builtin_clamp64(vm_t* vm, evaluekind_t kind)
    {
        const T high  = (T)pop_bits64(vm, kind);
        const T low   = (T)pop_bits64(vm, kind);
        const T value = (T)pop_bits64(vm, kind);
        push_bits64(vm, kind, (u64)clamp_value(value, low, high));
    }

    static void execute_builtin_clamp_float32(vm_t* vm)
    {
        const evaluekind_t kind  = KindFloat32;
        const f32          high  = bits_to_f32(pop_bits32(vm, kind));
        const f32          low   = bits_to_f32(pop_bits32(vm, kind));
        const f32          value = bits_to_f32(pop_bits32(vm, kind));
        push_bits32(vm, kind, f32_to_bits(clamp_value(value, low, high)));
    }

    static void execute_builtin_clamp_float64(vm_t* vm)
    {
        const evaluekind_t kind  = KindFloat64;
        const f64          high  = bits_to_f64(pop_bits64(vm, kind));
        const f64          low   = bits_to_f64(pop_bits64(vm, kind));
        const f64          value = bits_to_f64(pop_bits64(vm, kind));
        push_bits64(vm, kind, f64_to_bits(clamp_value(value, low, high)));
    }

    static void execute_builtin_clamp(vm_t* vm, evaluekind_t kind)
    {
        switch (kind)
        {
            case KindInt8:
            case KindInt16:
            case KindInt32: execute_builtin_clamp_i32(vm, kind); break;
            case KindByte:
            case KindUint8:
            case KindUint16:
            case KindUint32: execute_builtin_clamp_u32(vm, kind); break;
            case KindFloat32: execute_builtin_clamp_float32(vm); break;
            case KindInt64: execute_builtin_clamp64<i64>(vm, kind); break;
            case KindUint64: execute_builtin_clamp64<u64>(vm, kind); break;
            case KindFloat64: execute_builtin_clamp_float64(vm); break;
            default: ASSERT(false); break;
        }
    }

    static void execute_builtin_map_i32(vm_t* vm, evaluekind_t kind)
    {
        i32       out_high = bits_to_i32(pop_bits32(vm, kind), kind);
        i32       out_low  = bits_to_i32(pop_bits32(vm, kind), kind);
        i32       in_high  = bits_to_i32(pop_bits32(vm, kind), kind);
        i32       in_low   = bits_to_i32(pop_bits32(vm, kind), kind);
        const i32 value    = bits_to_i32(pop_bits32(vm, kind), kind);
        push_bits32(vm, kind, (u32)map_value(value, in_low, in_high, out_low, out_high));
    }

    static void execute_builtin_map_u32(vm_t* vm, evaluekind_t kind)
    {
        u32       out_high = pop_bits32(vm, kind);
        u32       out_low  = pop_bits32(vm, kind);
        u32       in_high  = pop_bits32(vm, kind);
        u32       in_low   = pop_bits32(vm, kind);
        const u32 value    = pop_bits32(vm, kind);
        push_bits32(vm, kind, map_value(value, in_low, in_high, out_low, out_high));
    }

    template <typename T> static void execute_builtin_map64(vm_t* vm, evaluekind_t kind)
    {
        T       out_high = (T)pop_bits64(vm, kind);
        T       out_low  = (T)pop_bits64(vm, kind);
        T       in_high  = (T)pop_bits64(vm, kind);
        T       in_low   = (T)pop_bits64(vm, kind);
        const T value    = (T)pop_bits64(vm, kind);
        push_bits64(vm, kind, (u64)map_value(value, in_low, in_high, out_low, out_high));
    }

    static void execute_builtin_map_float32(vm_t* vm)
    {
        const evaluekind_t kind     = KindFloat32;
        f32                out_high = bits_to_f32(pop_bits32(vm, kind));
        f32                out_low  = bits_to_f32(pop_bits32(vm, kind));
        f32                in_high  = bits_to_f32(pop_bits32(vm, kind));
        f32                in_low   = bits_to_f32(pop_bits32(vm, kind));
        const f32          value    = bits_to_f32(pop_bits32(vm, kind));
        push_bits32(vm, kind, f32_to_bits(map_value(value, in_low, in_high, out_low, out_high)));
    }

    static void execute_builtin_map_float64(vm_t* vm)
    {
        const evaluekind_t kind     = KindFloat64;
        f64                out_high = bits_to_f64(pop_bits64(vm, kind));
        f64                out_low  = bits_to_f64(pop_bits64(vm, kind));
        f64                in_high  = bits_to_f64(pop_bits64(vm, kind));
        f64                in_low   = bits_to_f64(pop_bits64(vm, kind));
        const f64          value    = bits_to_f64(pop_bits64(vm, kind));
        push_bits64(vm, kind, f64_to_bits(map_value(value, in_low, in_high, out_low, out_high)));
    }

    static void execute_builtin_map(vm_t* vm, evaluekind_t kind)
    {
        switch (kind)
        {
            case KindInt8:
            case KindInt16:
            case KindInt32: execute_builtin_map_i32(vm, kind); break;
            case KindByte:
            case KindUint8:
            case KindUint16:
            case KindUint32: execute_builtin_map_u32(vm, kind); break;
            case KindFloat32: execute_builtin_map_float32(vm); break;
            case KindInt64: execute_builtin_map64<i64>(vm, kind); break;
            case KindUint64: execute_builtin_map64<u64>(vm, kind); break;
            case KindFloat64: execute_builtin_map_float64(vm); break;
            default: ASSERT(false); break;
        }
    }

    template <typename T> static T pop_bits(vm_t* vm, evaluekind_t kind)
    {
        static_assert(sizeof(T) == sizeof(u32) || sizeof(T) == sizeof(u64), "unsupported builtin stack value width");
        return sizeof(T) == sizeof(u32) ? (T)pop_bits32(vm, kind) : (T)pop_bits64(vm, kind);
    }

    template <typename T> static void push_bits(vm_t* vm, evaluekind_t kind, T value)
    {
        static_assert(sizeof(T) == sizeof(u32) || sizeof(T) == sizeof(u64), "unsupported builtin stack value width");
        if (sizeof(T) == sizeof(u32))
            push_bits32(vm, kind, (u32)value);
        else
            push_bits64(vm, kind, (u64)value);
    }

    template <typename T> static void execute_builtin_fixed_point_t(vm_t* vm, ebuiltinoperation_t operation, evaluekind_t kind)
    {
        ASSERT(operation == BuiltInLerp || operation == BuiltInSmoothStep);
        ASSERT((sizeof(T) == sizeof(s32) && kind == KindInt32) || (sizeof(T) == sizeof(s64) && kind == KindInt64));
        const u8 shift = (u8)pop_bits32(vm, KindUint8);
        ASSERT(shift < (sizeof(T) * 8) - 1);
        const T t     = pop_bits<T>(vm, kind);
        const T end   = pop_bits<T>(vm, kind);
        const T start = pop_bits<T>(vm, kind);
        const T max_t = (T)1 << shift;
        T       result;
        if (t <= 0)
            result = start;
        else if (t >= max_t)
            result = end;
        else if (operation == BuiltInLerp)
            result = start + (((end - start) * t) >> shift);
        else
        {
            const u8  curve_shift = shift > 15 ? 15 : shift;
            const s64 smoothed_t  = ((s64)t * (s64)t * (((s64)3 << curve_shift) - ((s64)t << 1))) >> (curve_shift * 2);
            result                = start + (((end - start) * (T)smoothed_t) >> curve_shift);
        }
        push_bits(vm, kind, result);
    }

    static void execute_builtin_fixed_point(vm_t* vm, ebuiltinoperation_t operation, evaluekind_t kind)
    {
        switch (kind)
        {
            case KindInt32: execute_builtin_fixed_point_t<s32>(vm, operation, kind); break;
            case KindInt64: execute_builtin_fixed_point_t<s64>(vm, operation, kind); break;
            default: ASSERT(false); break;
        }
    }

    typedef f32 (*builtin_unary_float32_fn)(f32 value);
    typedef f64 (*builtin_unary_float64_fn)(f64 value);

    static void execute_builtin_unary_float32(vm_t* vm, builtin_unary_float32_fn function)
    {
        const evaluekind_t kind  = KindFloat32;
        const f32          value = bits_to_f32(pop_bits32(vm, kind));
        push_bits32(vm, kind, f32_to_bits(function(value)));
    }

    static void execute_builtin_unary_float64(vm_t* vm, builtin_unary_float64_fn function)
    {
        const evaluekind_t kind  = KindFloat64;
        const f64          value = bits_to_f64(pop_bits64(vm, kind));
        push_bits64(vm, kind, f64_to_bits(function(value)));
    }

    static f32 builtin_sin_float32(f32 value) { return std::sin(value); }
    static f64 builtin_sin_float64(f64 value) { return std::sin(value); }
    static f32 builtin_cos_float32(f32 value) { return std::cos(value); }
    static f64 builtin_cos_float64(f64 value) { return std::cos(value); }
    static f32 builtin_tan_float32(f32 value) { return std::tan(value); }
    static f64 builtin_tan_float64(f64 value) { return std::tan(value); }
    static f32 builtin_asin_float32(f32 value) { return std::asin(value); }
    static f64 builtin_asin_float64(f64 value) { return std::asin(value); }
    static f32 builtin_acos_float32(f32 value) { return std::acos(value); }
    static f64 builtin_acos_float64(f64 value) { return std::acos(value); }
    static f32 builtin_atan_float32(f32 value) { return std::atan(value); }
    static f64 builtin_atan_float64(f64 value) { return std::atan(value); }
    static f32 builtin_sqrt_float32(f32 value) { return std::sqrt(value); }
    static f64 builtin_sqrt_float64(f64 value) { return std::sqrt(value); }

    static void execute_builtin_pow_float32(vm_t* vm)
    {
        const evaluekind_t kind     = KindFloat32;
        const f32          exponent = bits_to_f32(pop_bits32(vm, kind));
        const f32          base     = bits_to_f32(pop_bits32(vm, kind));
        push_bits32(vm, kind, f32_to_bits((f32)std::pow((f64)base, (f64)exponent)));
    }

    static void execute_builtin_pow_float64(vm_t* vm)
    {
        const evaluekind_t kind     = KindFloat64;
        const f64          exponent = bits_to_f64(pop_bits64(vm, kind));
        const f64          base     = bits_to_f64(pop_bits64(vm, kind));
        push_bits64(vm, kind, f64_to_bits(std::pow(base, exponent)));
    }

    static void execute_builtin_lerp_float32(vm_t* vm)
    {
        const evaluekind_t kind = KindFloat32;
        const f32          t    = bits_to_f32(pop_bits32(vm, kind));
        const f32          b    = bits_to_f32(pop_bits32(vm, kind));
        const f32          a    = bits_to_f32(pop_bits32(vm, kind));
        push_bits32(vm, kind, f32_to_bits(a + (b - a) * t));
    }

    static void execute_builtin_lerp_float64(vm_t* vm)
    {
        const evaluekind_t kind = KindFloat64;
        const f64          t    = bits_to_f64(pop_bits64(vm, kind));
        const f64          b    = bits_to_f64(pop_bits64(vm, kind));
        const f64          a    = bits_to_f64(pop_bits64(vm, kind));
        push_bits64(vm, kind, f64_to_bits(a + (b - a) * t));
    }

    static void execute_builtin_smooth_step_float32(vm_t* vm)
    {
        const evaluekind_t kind  = KindFloat32;
        const f32          x     = bits_to_f32(pop_bits32(vm, kind));
        const f32          edge1 = bits_to_f32(pop_bits32(vm, kind));
        const f32          edge0 = bits_to_f32(pop_bits32(vm, kind));
        f32                result;
        if (x <= edge0)
            result = 0.0f;
        else if (x >= edge1)
            result = 1.0f;
        else
        {
            const f32 t = (x - edge0) / (edge1 - edge0);
            result      = t * t * (3.0f - 2.0f * t);
        }
        push_bits32(vm, kind, f32_to_bits(result));
    }

    static void execute_builtin_smooth_step_float64(vm_t* vm)
    {
        const evaluekind_t kind  = KindFloat64;
        const f64          x     = bits_to_f64(pop_bits64(vm, kind));
        const f64          edge1 = bits_to_f64(pop_bits64(vm, kind));
        const f64          edge0 = bits_to_f64(pop_bits64(vm, kind));
        f64                result;
        if (x <= edge0)
            result = 0.0;
        else if (x >= edge1)
            result = 1.0;
        else
        {
            const f64 t = (x - edge0) / (edge1 - edge0);
            result      = t * t * (3.0 - 2.0 * t);
        }
        push_bits64(vm, kind, f64_to_bits(result));
    }

    static void execute_builtin_slerp_float32(vm_t* vm)
    {
        const evaluekind_t kind   = KindFloat32;
        const f32          t      = bits_to_f32(pop_bits32(vm, kind));
        const f32          b      = bits_to_f32(pop_bits32(vm, kind));
        const f32          a      = bits_to_f32(pop_bits32(vm, kind));
        const f32          theta  = std::acos(a * b);
        const f32          result = std::fabs(theta) < 0.00001f ? a : (std::sin((1.0f - t) * theta) * a + std::sin(t * theta) * b) / std::sin(theta);
        push_bits32(vm, kind, f32_to_bits(result));
    }

    static void execute_builtin_slerp_float64(vm_t* vm)
    {
        const evaluekind_t kind   = KindFloat64;
        const f64          t      = bits_to_f64(pop_bits64(vm, kind));
        const f64          b      = bits_to_f64(pop_bits64(vm, kind));
        const f64          a      = bits_to_f64(pop_bits64(vm, kind));
        const f64          theta  = std::acos(a * b);
        const f64          result = std::fabs(theta) < 0.00001 ? a : (std::sin((1.0 - t) * theta) * a + std::sin(t * theta) * b) / std::sin(theta);
        push_bits64(vm, kind, f64_to_bits(result));
    }

    void execute_builtin(vm_t* vm, builtin_function_t function)
    {
        const ebuiltinoperation_t operation = builtin_function_operation(function);
        const evaluekind_t        kind      = builtin_function_kind(function);
        switch (operation)
        {
            case BuiltInAbs: execute_builtin_abs(vm, kind); break;
            case BuiltInSin:
                switch (kind)
                {
                    case KindFloat32: execute_builtin_unary_float32(vm, builtin_sin_float32); break;
                    case KindFloat64: execute_builtin_unary_float64(vm, builtin_sin_float64); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInCos:
                switch (kind)
                {
                    case KindFloat32: execute_builtin_unary_float32(vm, builtin_cos_float32); break;
                    case KindFloat64: execute_builtin_unary_float64(vm, builtin_cos_float64); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInTan:
                switch (kind)
                {
                    case KindFloat32: execute_builtin_unary_float32(vm, builtin_tan_float32); break;
                    case KindFloat64: execute_builtin_unary_float64(vm, builtin_tan_float64); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInAsin:
                switch (kind)
                {
                    case KindFloat32: execute_builtin_unary_float32(vm, builtin_asin_float32); break;
                    case KindFloat64: execute_builtin_unary_float64(vm, builtin_asin_float64); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInAcos:
                switch (kind)
                {
                    case KindFloat32: execute_builtin_unary_float32(vm, builtin_acos_float32); break;
                    case KindFloat64: execute_builtin_unary_float64(vm, builtin_acos_float64); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInAtan:
                switch (kind)
                {
                    case KindFloat32: execute_builtin_unary_float32(vm, builtin_atan_float32); break;
                    case KindFloat64: execute_builtin_unary_float64(vm, builtin_atan_float64); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInPow:
                switch (kind)
                {
                    case KindFloat32: execute_builtin_pow_float32(vm); break;
                    case KindFloat64: execute_builtin_pow_float64(vm); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInSqrt:
                switch (kind)
                {
                    case KindFloat32: execute_builtin_unary_float32(vm, builtin_sqrt_float32); break;
                    case KindFloat64: execute_builtin_unary_float64(vm, builtin_sqrt_float64); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInMin:
                switch (value_kind_size(kind))
                {
                    case 1:
                    case 2:
                    case 4: execute_builtin_min_or_max_32(vm, operation, kind); break;
                    case 8: execute_builtin_min_or_max_64(vm, operation, kind); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInMax:
                switch (value_kind_size(kind))
                {
                    case 1:
                    case 2:
                    case 4: execute_builtin_min_or_max_32(vm, operation, kind); break;
                    case 8: execute_builtin_min_or_max_64(vm, operation, kind); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInMap: execute_builtin_map(vm, kind); break;
            case BuiltInSmoothStep:
                switch (kind)
                {
                    case KindInt32:
                    case KindInt64: execute_builtin_fixed_point(vm, operation, kind); break;
                    case KindFloat32: execute_builtin_smooth_step_float32(vm); break;
                    case KindFloat64: execute_builtin_smooth_step_float64(vm); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInRandom:
                ASSERT(kind == KindInt32);
                push_bits32(vm, kind, (u32)next_random(vm) & 0x7fffffffU);
                break;
            case BuiltInClamp: execute_builtin_clamp(vm, kind); break;
            case BuiltInLerp:
                switch (kind)
                {
                    case KindInt32:
                    case KindInt64: execute_builtin_fixed_point(vm, operation, kind); break;
                    case KindFloat32: execute_builtin_lerp_float32(vm); break;
                    case KindFloat64: execute_builtin_lerp_float64(vm); break;
                    default: ASSERT(false); break;
                }
                break;
            case BuiltInSlerp:
                switch (kind)
                {
                    case KindFloat32: execute_builtin_slerp_float32(vm); break;
                    case KindFloat64: execute_builtin_slerp_float64(vm); break;
                    default: ASSERT(false); break;
                }
                break;
            default: ASSERT(false); break;
        }
    }

    instruction_t make_builtin_instruction(builtin_function_t function)
    {
        ASSERT(function <= 0x07ffU);
        return (instruction_t)((u16)OpBuiltIn | ((function & 0x07ffU) << 5));
    }

    builtin_function_t make_builtin_function(ebuiltinoperation_t operation, evaluekind_t kind)
    {
        ASSERT((u32)operation < 128U);
        ASSERT((u32)kind < (u32)KindCount);
        return (builtin_function_t)(((u16)operation << 4) | (u16)kind);
    }

    ebuiltinoperation_t builtin_function_operation(builtin_function_t function) { return (ebuiltinoperation_t)((function >> 4) & 0x7fU); }
    evaluekind_t        builtin_function_kind(builtin_function_t function) { return (evaluekind_t)(function & 0x0fU); }
    builtin_function_t  instruction_builtin_function(instruction_t instruction) { return (builtin_function_t)((instruction >> 5) & 0x07ffU); }

} // namespace ncore
