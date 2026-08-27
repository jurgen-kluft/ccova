# TODO

## Language

- Aggregate initializers for arrays and structs
  - Positional initialization in declaration order
  - Designated initialization by member name
- Address-of, dereference, and pointer member access
- Passing aggregate values or references to functions
- Optional per-array runtime bounds checking
- Local declarations in `for` initializers
- Ternary expressions and increment/decrement operators

## Built-in functions

- time
  - `float time::frame_time()`
  - Timers (`0 <= id < N`)
    - `void time::timer_start(id, timeout_ms)`
    - `void time::timer_reset(id)`
    - `bool time::timer_query(id)`
    - `void time::timer_stop(id)`

- [DONE] math
  - `math::min`, `math::max`
  - `math::map`
  - `math::random`
  - `math::clamp`
  - `math::smoothstep`
  - `math::interpolate`
  - `math::lerp`, `math::slerp`


