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
  - `float frame_time()`
  - Timers (`0 <= id < 32`)
    - `void timer_start(id, timeout_ms)`
    - `void timer_reset(id)`
    - `bool timer_query(id)`
    - `void timer_stop(id)`

- math
  - `math_min`, `math_max`
  - `math_random`
  - `math_clamp`
  - `math_smoothstep`
  - `math_interpolate`
  - `math_lerp`, `math_slerp`


