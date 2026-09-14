# Cova

`cova` is a small compiler, linker, and VM for a typed, C-like scripting language.

The language is designed for small host-integrated scripts that work with primitive numeric types, control flow, global state, and `extern` bindings for memory and host functions.

## What It Supports

- Primitive types: 
  - `bool`, `byte`/`char`
  - `int8`/`i8`, `uint8`/`u8`
  - `int16`/`i16`, `uint16`/`u16`
  - `int32`/`i32`, `uint32`/`u32`
  - `int64`/`i64`, `uint64`/`u64`
  - `float32`/`float`/`f32`
  - `float64`/`double`/`f64`
- Top-level globals and named structs with naturally aligned fields
- Fixed arrays with expression indexing and chained struct member access
- Typed, block-scoped local variables inside functions
- Script functions with typed parameters and returns
- String literals passed as pointer values
- Arithmetic, comparison, logical, unary, modulo, bitwise, and shift expressions
- Arithmetic, bitwise, and shift compound assignments
- Control flow: `if`, `if/else`, `while`, `for`, `switch`, `break`, `continue`, `return`
- Math built-ins: 
  - `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `sqrt`, `pow`
  - `min`, `max`, `clamp`, `map`
  - `random`, `smoothstep`, `interpolate`, `lerp`, `slerp`
- `//` single-line and `/* ... */` block comments
- `extern` variables with naturally aligned offsets assigned in source order
- `extern(slot)` functions dispatched by the host

## Current Limits

- No local declarations in `for` initializers
- Standalone expression statements must be function calls
- Struct definitions are top-level only; struct and array values cannot be locals, parameters, or return values
- Whole-aggregate assignment and aggregate initialization are not supported
- Constant array indexes are checked at compile time; dynamic indexes have segment bounds protection but no per-array bounds check
- Pointer types can be declared, but address-of, dereference, and pointer member access are not implemented
- Ternary expressions, increment/decrement, variadics, and preprocessing are not implemented
- Recursive script call cycles are rejected at compile time

String literals are stored in a CONST segment as NUL-terminated byte strings. Zero-initialized globals remain in BSS, while initialized writable globals are placed in DATA.

Booleans use numeric truthiness at runtime: `false` is `0`, and any non-zero value is true. Logical operators short-circuit and produce normalized `0` or `1` results.

## VM Workspace Policy

The host owns execution workspace sizing. `VMConfig.FrameCapacity`, `StackCapacity`, and `CallFrameCapacity` are fixed limits selected from the application's memory budget; the compiler and linker do not infer aggregate frame use, maximum operand-stack height, or maximum call depth. `LoadProgram` validates the program image independently of these limits. A limit that is too small produces `VMStatusFrameOverflow`, `VMStatusStackOverflow`, or `VMStatusCallFrameOverflow` only if execution reaches the operation that needs more workspace.

`LinkedProgram.FrameByteSize` is compiler metadata for the largest individual function frame. It is not aggregate call-path memory and is not a complete VM sizing recommendation.

VM runtime and stack APIs return the stable numeric `VMStatus` enum directly. `VMStatusOK` is zero. `VMStatus.String()` provides optional diagnostics outside the execution hot path, while `VM.FaultInfo()` exposes the failing PC, target, required/available capacity, and host callback status without formatting strings.

```go
vm := cova.NewVMWithConfig(cova.VMConfig{
    FrameCapacity:     512,
    StackCapacity:     128,
    CallFrameCapacity: 8,
})
if status := vm.LoadProgram(linked); status != cova.VMStatusOK {
    return fmt.Errorf("load failed: %s", status)
}
if status := vm.RunLoaded(); status != cova.VMStatusOK {
    fault := vm.FaultInfo()
    return fmt.Errorf("VM failed at PC %d: %s", fault.PC, status)
}
```

## Optimization

Optimization is an explicit, optional stage between parsing and compilation:

```go
ctx := cova.NewContext()
program, ok := cova.Parse(ctx, tokens)
if !ok {
    return fmt.Errorf("parse failed: %v", ctx.Issues())
}
if !cova.Optimize(ctx, program) {
    return fmt.Errorf("optimization failed: %v", ctx.Issues())
}
compiled, err := cova.NewCompiler().Compile(program)
```

`Optimize` mutates the AST in place. Its constant folding pass handles numeric arithmetic, comparisons, and logical expressions, including short-circuit branches. A reachable constant division by zero is reported as an optimization error; an unreachable short-circuit branch is not evaluated.

The optimizer is intentionally isolated from compiler and VM internals. It owns the small amount of type promotion, conversion, and evaluation logic required for folding, and optimized-versus-unoptimized tests guard that duplicated behavior against semantic drift.

## Example

```c
extern(0) void log_alert(int value);

struct player_state {
    int health;
    char alerts[4];
};

extern player_state player;

int health_drop;

void script_main() {
    health_drop = 5;
    if (player.health < 40) {
        player.alerts[0] = 1;
        log_alert(player.health);
        reduce_health(health_drop);
    }
    return;
}

void reduce_health(int delta) {
    player.health = player.health - delta;
    return;
}
```

Extern functions keep explicit host dispatch slots. Extern variables omit offsets; the compiler lays them out automatically using their natural alignment.

## Documentation

- Full language overview: [LANGUAGE.md](docs/LANGUAGE.md)

## Development

Run the test suite with:

```sh
cd cova
go test ./...
go test -race ./...
```

### Embedded native VM fixture

The native integration suite executes a program compiled and serialized by the Go toolchain. The binary fixture is tracked under `embedded/source/test/cpp`, and ccode converts it into an aligned C++ byte array under `source/test/cpp`.

Refresh the fixture after an intentional compiler, linker, or image-format change:

```sh
cd cova
CCOVA_UPDATE_EMBEDDED=1 go test -run TestEmbeddedProgramImageFixture -count=1
cd ..
go run ccova.go --dev=clay
```

Ordinary Go test runs only verify that the tracked fixture matches freshly generated bytes; they do not rewrite it. After regeneration, build and run the native integration test through Clay:

```sh
cd target/clay
./clay build --build release-dev-test
./build/darwin-arm64-release-dev-test/unittest_ccova/unittest_ccova
```
