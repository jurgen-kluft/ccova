# Cova Script Language

This document describes the scripting language currently supported by `ccova`.

The language is a small, typed, C-like scripting language intended to compile into bytecode and run inside the `ccova` VM. It supports primitive numeric types, global variables, functions, control flow, and host interop through `extern` declarations.

## Example

```c
extern(0) void log_alert(int value);
extern int player_health;

int health_drop;

void script_main() {
    health_drop = 5;
    if ((player_health - 40) + 1) {
        log_alert(player_health);
        reduce_health(health_drop);
    }
    return;
}

void reduce_health(int delta) {
    player_health = player_health - delta;
    return;
}
```

## Overview

The language currently supports:

- Top-level global variables
- Local variables inside blocks
- `const` globals, locals, and parameters
- Host-linked `extern` variables
- Host-linked `extern` functions
- Top-level named structs with naturally aligned fields
- Fixed arrays and `char`
- Chained struct member and array index access
- Script-defined functions
- Primitive numeric and boolean-like types
- String literals lowered to pointer values
- `//` single-line comments
- `/* ... */` block comments
- Numeric expressions
- Unary, bitwise, shift, and modulo expressions
- Arithmetic, bitwise, and shift compound assignments
- Function calls
- `if`, `if/else`, `while`, `for`, `switch`, `break`, `continue`, and `return`

The language does not currently support:

- Local variable declarations inside `for` initializers
- Address-of and dereference expressions such as `&x` and `*ptr`
- Usable source-level pointer operations
- Whole-aggregate assignment, aggregate initializers, and aggregate function values
- Pointer member access (`->`), ternary expressions, increment/decrement, variadics, and preprocessing

## Compact Grammar

This grammar is intentionally compact. It shows the syntax that is currently accepted by the parser, not a full semantic specification.

```ebnf
program        ::= top_level*

top_level      ::= struct_decl | extern_decl | global_decl | function_decl

struct_decl    ::= "struct" ident "{" field_decl* "}" ";"
field_decl     ::= type declarator ";"
extern_decl    ::= "extern" type declarator ";"
                 | "extern" "(" number ")" type ident "(" param_list? ")" ";"
global_decl    ::= type declarator ("=" expr)? ";"
function_decl  ::= type ident "(" param_list? ")" block
declarator     ::= ident ("[" integer_literal "]")*

param_list     ::= param ("," param)*
param          ::= type declarator
const_qualifier ::= "const"
type           ::= const_qualifier? named_type const_qualifier? ("*" const_qualifier?)*
named_type     ::= ident | "void" | "bool" | "byte" | "char" | "int" |
                   "int8" | "int16" | "int32" | "int64" |
                   "uint8" | "uint16" | "uint32" | "uint64" |
                   "float" | "float32" | "float64" |
                   "i8" | "i16" | "i32" | "i64" |
                   "u8" | "u16" | "u32" | "u64" |
                   "f32" | "f64" | "double"

block          ::= "{" stmt* "}"
stmt           ::= block
                 | local_decl
                 | if_stmt
                 | while_stmt
                 | for_stmt
                 | switch_stmt
                 | return_stmt
                 | break_stmt
                 | continue_stmt
                 | assign_stmt
                 | expr_stmt

if_stmt        ::= "if" "(" expr ")" stmt ("else" stmt)?
while_stmt     ::= "while" "(" expr ")" stmt
for_stmt       ::= "for" "(" for_init? ";" expr? ";" for_post? ")" stmt
for_init       ::= assign_stmt_no_semi | expr
for_post       ::= assign_stmt_no_semi | expr
switch_stmt    ::= "switch" "(" expr ")" "{" switch_case* default_case? "}"
switch_case    ::= "case" expr ":" stmt*
default_case   ::= "default" ":" stmt*
return_stmt    ::= "return" expr? ";"
break_stmt     ::= "break" ";"
continue_stmt  ::= "continue" ";"
assign_stmt    ::= lvalue assign_op expr ";"
assign_stmt_no_semi ::= lvalue assign_op expr
assign_op      ::= "=" | "+=" | "-=" | "*=" | "/=" | "%=" |
                   "<<=" | ">>=" | "&=" | "^=" | "|="
local_decl     ::= const_qualifier? type ident ("=" expr)? ";"
expr_stmt      ::= expr ";"

 expr           ::= logical_or
 logical_or     ::= logical_and ("||" logical_and)*
 logical_and    ::= bitwise_or ("&&" bitwise_or)*
 bitwise_or     ::= bitwise_xor ("|" bitwise_xor)*
 bitwise_xor    ::= bitwise_and ("^" bitwise_and)*
 bitwise_and    ::= equality ("&" equality)*
 equality       ::= relational (("==" | "!=") relational)*
 relational     ::= shift (("<" | "<=" | ">" | ">=") shift)*
 shift          ::= additive (("<<" | ">>") additive)*
 additive       ::= multiplicative (("+" | "-") multiplicative)*
 multiplicative ::= unary (("*" | "/" | "%") unary)*
 unary          ::= ("!" | "~" | "-") unary | postfix
 postfix        ::= primary (("." ident) | ("[" expr "]"))*
 primary        ::= boolean
                  | number
                  | string
                 | ident
                 | call
                 | "(" expr ")"
call           ::= ident "(" arg_list? ")"
arg_list       ::= expr ("," expr)*
lvalue         ::= ident (("." ident) | ("[" expr "]"))*

ident          ::= identifier
 boolean       ::= "true" | "false"
number         ::= integer_literal | float_literal
string         ::= string_literal
```

Notes:

- `expr_stmt` is broader in the grammar than in the compiler. In practice, only function calls are accepted as standalone expression statements.
- Pointer type syntax is accepted in declarations and signatures, but source-level pointer operators are not implemented.
- Boolean conditions are numeric at runtime: `0` is false and non-zero is true.

## Comments

A line comment starts with `//` and continues to the end of the line. It can occupy a whole line or follow source code.

```c
extern int player_health;

void apply_damage() {
    // Use the host-provided health value.
    int health_drop = 5;
    player_health = player_health - health_drop; // Apply the damage.
}
```

Comments are discarded by the lexer and do not produce tokens. Block comments may span lines and do not nest.

The tokenizer emits physical newline tokens for tooling. The parser ignores them, and semicolons remain mandatory statement and declaration terminators.

## Recognized Punctuators

The tokenizer recognizes ordinary C-like punctuators, including `[]`, `.`, `->`, `?`, `++`, `--`, `...`, `#`, and `##`, plus `::`. Fixed-array indexing and `.` member access are implemented. Pointer member access, qualified names, ternary expressions, increment/decrement, variadics, and preprocessing remain unsupported. ISO C digraph aliases are not recognized.

## Top-Level Declarations

There are two kinds of top-level declarations:

- Internal globals
- `extern` declarations

### Internal globals

Internal globals without an initializer are stored in VM-managed BSS memory and are zero-initialized before execution.

```c
int counter;
float32 ratio;
bool ready;
```

Internal globals with an initializer are stored in writable DATA memory.

```c
int retries = 3;
```

Globals whose top-level type is const are stored in the read-only CONST segment. A pointer-to-const type such as `const uint8*` is not itself a const global, because the pointer value remains mutable.

```c
const int32 threshold = 7;
const uint8* const asset_path = "asset/button_off";
```

### Extern variables

Extern variables map onto a host-provided memory block. The compiler assigns naturally aligned byte offsets in source declaration order.

```c
extern int32 health;       // offset 0
extern uint64 flags;       // offset 8
extern float64 temperature; // offset 16
```

Parenthesized slots remain required for extern functions, but explicit offsets are no longer accepted for extern variables.

`extern` variables cannot be declared with `const`.

### Extern functions

Extern functions are host callbacks identified by import slot.

```c
extern(0) void log_value(int value);
extern(1) uint64 bounce(uint64 value);
```

The VM calls the registered extern dispatcher with the given import ID.

Runtime dispatch uses a fixed-width `uint32` import ID, an opaque host context token, and a stable numeric `VMStatus` result. The host reads arguments and writes a return value through checked typed VM stack helpers. Detailed extern signature metadata and exact consumption/production enforcement are part of the portable host ABI work still in progress.

## VM Execution Capacities

Operand-stack, aggregate frame-buffer, and call-frame capacities are host policy supplied through `VMConfig`. They are not computed from script control flow and are not requirements embedded in a linked program. Linking and loading therefore remain independent of the selected workspace capacities. If a reached execution path exceeds one of the configured limits, the VM returns the corresponding stable overflow status at that operation.

`LinkedProgram.FrameByteSize` records only the largest individual script-function frame produced by the compiler. Nested calls can require more frame memory, so this value must not be interpreted as aggregate workspace sizing.

### Script functions

Script functions are defined directly in the script.

```c
int add_one(int value) {
    return value + 1;
}
```

The required entry point is `script_main`; compilation fails if it is absent. Every script function body is compiled and validated as an independent bytecode block. Only blocks reachable from `script_main` are assembled into the executable text, preserving their source order.

### Local variables

Local variables use an explicit type and can be declared in any function block. An initializer is optional. A local without an initializer receives the zero value for its type.

```c
int count;        // Initialized to 0.
bool ready = true;
```

Local names are visible from their declaration to the end of the containing block. A nested block may declare a local with the same name, temporarily shadowing the outer local. Declaring the same name twice in one block is rejected.

```c
int choose_value() {
    int value = 1;
    {
        int value = 2; // Shadows the outer value in this block.
        value = value + 1;
    }
    return value; // Returns 1.
}
```

Mutable locals can be assigned after declaration. A local whose top-level type is `const` cannot be assigned after its declaration. If a const local has no initializer, it retains its zero value and cannot be assigned later.

```c
const int answer = 42;
```

`const uint8*` means a mutable pointer to const bytes, so the pointer can still be reassigned. `uint8* const` means a const pointer to mutable bytes.

String literals can initialize pointer locals only when the pointed-to type is `const uint8`.

```c
const uint8* asset_path = "asset/button_off";
```

Local declarations are not accepted in a `for` initializer. Declare the local before the loop instead.

```c
int count_to_four() {
    int index;
    for (index = 0; index < 4; index = index + 1) {
        // Use index.
    }
    return index;
}
```

## Types

Supported type names:

- `void`
- `bool`
- `byte`
- `char`
- `int`
- `int8`
- `int16`
- `int32`
- `int64`
- `uint8`
- `uint16`
- `uint32`
- `uint64`
- `float`
- `float32`
- `float64`
- `double`
- `i8`
- `i16`
- `i32`
- `i64`
- `u8`
- `u16`
- `u32`
- `u64`
- `f32`
- `f64`

### Type aliases

The following reserved type names are aliases:

- `char` is an alias for `uint8`
- `int` and `i32` are aliases for `int32`
- `i8`, `i16`, and `i64` are aliases for `int8`, `int16`, and `int64`
- `u8`, `u16`, `u32`, and `u64` are aliases for `uint8`, `uint16`, `uint32`, and `uint64`
- `float` and `f32` are aliases for `float32`
- `double` and `f64` are aliases for `float64`

Aliases have the same type identity as their canonical types. Diagnostics use the canonical names.

`float`, `float32`, and `f32` have identical type identity and behavior.
`double`, `float64`, and `f64` likewise have identical type identity and behavior.

## String Literals

String literals use double quotes and support a small escape set: `\"`, `\\`, `\n`, `\r`, `\t`, and `\0`.

```c
extern(0) void inspect(const uint8* path);

void script_main() {
    inspect("asset/button_off");
    return;
}
```

Current behavior:

- String literals are stored in the CONST segment.
- Identical string literals are de-duplicated inside the CONST segment.
- Each literal is emitted as a NUL-terminated byte string for C-style `const char*` interop.
- Only globals whose top-level type is const are stored in CONST.
- Mutable globals with initializers are stored in DATA.
- Zero-initialized globals remain in BSS.
- String literals are assignable only to pointer targets whose pointee type is `const uint8`. Because `char` aliases `uint8`, this includes `const uint8*`, `const char*`, and their const-pointer forms.

## Literals

### Integer literals

```c
0
1
42
255
```

### Floating-point literals

```c
0.5
2.25
1e3
7.25e-2
```

### Float suffixes

- `f` or `F` means `float32`
- `d` or `D` means `float64`

```c
1.5f
2.5d
6e3F
3e1D
```

### Boolean literals

The language supports `true` and `false` as boolean literals.

At runtime, boolean truth is still numeric:

- `false` is `0`
- any non-zero value is true

Examples:

```c
ready = true;
ready = false;
ready = 1;
ready = 7;
```

Unary minus is supported for integer and floating-point values. Unsigned negation uses modular arithmetic.

## Expressions

Supported expressions:

- Boolean literals
- Numeric literals
- Variable references
- Function calls
- Parenthesized expressions
- Binary arithmetic
- Unary arithmetic and logical operations
- Bitwise and shift operations
- Comparisons
- Logical operators

### Arithmetic

Supported operators:

- `+`
- `-`
- `*`
- `/`
- `%` for integer operands

Examples:

```c
counter + 1
total - delta
base * 2
amount / 4
remainder % 4
```

### Unary and bitwise operators

Prefix `-` negates numeric values, `!` tests numeric truthiness and produces `bool`, and `~` complements integer bits. Binary `&`, `|`, and `^` accept integer operands. Shifts accept integer operands; signed right shift is arithmetic and unsigned right shift is logical. Shift counts are masked to the promoted left operand width.

Compound forms `+=`, `-=`, `*=`, `/=`, `%=`, `<<=`, `>>=`, `&=`, `^=`, and `|=` update an assignable variable.

### Comparisons

Supported operators:

- `==`
- `!=`
- `<`
- `<=`
- `>`
- `>=`

Examples:

```c
counter < 10
value == 3
health >= limit
```

Comparisons produce a `bool` result using `false` and `true`.

### Logical operators

Supported operators:

- `&&`
- `||`

Examples:

```c
ready && enabled
counter > 0 || limit == 3
false && mark_true()
```

Logical operators use numeric truthiness, short-circuit evaluation, and produce a normalized boolean result:

- `0` when the expression is false
- `1` when the expression is true

From lowest to highest, binary precedence is `||`, `&&`, `|`, `^`, `&`, equality, relational comparisons, shifts, addition/subtraction, and multiplication/division/modulo. Prefix unary operators bind above those binary operators, and calls bind highest.

### Numeric promotion

The compiler promotes mixed numeric expressions to a suitable common type.

Examples:

- `int32 + float64` becomes `float64`
- `float32 + float64` becomes `float64`

Example:

```c
int base;

float64 script_main() {
    base = 2;
    return base + 1.5;
}
```

### Math built-ins

The following functions are implicit built-ins and do not require declarations:

- `abs(value)`
- `sin(value)`, `cos(value)`, and `tan(value)`
- `asin(value)`, `acos(value)`, and `atan(value)`
- `pow(base, exponent)`
- `sqrt(value)`

`abs` preserves the input numeric type. For the minimum value of a signed integer type, the result wraps to the same minimum value. Unsigned and `byte` inputs are unchanged.

The other built-ins preserve `float32` when all operands are `float32`; `float64` inputs produce `float64`. Integer operands are promoted to `float64`. Mixed `pow` operands use normal numeric promotion, with an integer-only result promoted to `float64`. Boolean arguments are rejected.

Trigonometric arguments and results use radians. Invalid floating-point domains, such as `sqrt(-1)` or `asin(2)`, follow IEEE 754 and produce NaN. Built-in names are reserved and cannot be redeclared.

## Statements

Supported statements:

- Block statements
- Assignment
- Function-call expression statements
- `if`
- `if/else`
- `while`
- `for`
- `switch`
- `break`
- `continue`
- `return`

### Blocks

```c
{
    counter = counter + 1;
    return;
}
```

### Assignment

```c
counter = 10;
total = total + 1;
player_health = player_health - delta;
```

Assignments require the left-hand side to be assignable. At the moment, that mainly means identifiers naming globals or parameters.

### Standalone expression statements

Only function calls are allowed as standalone expression statements.

Valid:

```c
log_value(counter);
```

Invalid:

```c
1 + 2;
counter;
```

### If / else

```c
if (counter == 3) {
    total = total + 10;
} else {
    total = total + 1;
}
```

### While

```c
while (counter < 5) {
    counter = counter + 1;
}
```

### For

```c
for (counter = 0; counter < 4; counter = counter + 1) {
    total = total + counter;
}
```

There are no local variable declarations in the `for` initializer.

### Switch

```c
switch (counter) {
case 1:
    total = total + 10;
    break;
case 2:
    total = total + 20;
    break;
default:
    total = total + 1;
}
```

Supported inside switch:

- `case`
- `default`
- `break`

`continue` is meaningful if the `switch` is inside a loop.

There is no explicit `fallthrough` keyword. If a case body does not `break`, execution naturally continues into later emitted case bodies.

### Return

```c
return;
return value + 1;
```

A non-void function can return a value. A void function can return without one.

## Truthiness

Conditions are numeric.

- `0` means false
- non-zero means true

Examples:

```c
if (counter) {
    return;
}

if (player_health - 40) {
    log_alert(player_health);
}
```

This is why code like this is valid today:

```c
if ((player_health - 40) + 1) {
    ...
}
```

## Functions

Functions take typed parameters and may return a typed value.

```c
int64 level3(int64 value, int64 extra) {
    return value + extra;
}

int64 level2(int64 left, int64 right) {
    return level3(left + right, right);
}
```

### Parameters

Parameters are supported and stored in the function frame.

```c
float64 blend(float32 a, float64 b) {
    return a + b;
}
```

### Recursion is rejected

Recursive script call cycles are currently rejected at compile time, including indirect recursion.

Invalid:

```c
int recurse(int value) {
    return recurse(value);
}
```

## Host Interop

Host interop happens through `extern` declarations.

### Extern memory

The host can bind a byte slice as extern memory. Extern variables read and write at compiler-generated, naturally aligned offsets.

```c
extern int64 total;
extern byte flag;
extern bool ready;
```

Script:

```c
void script_main() {
    total = 42;
    flag = 255;
    ready = 1;
    return;
}
```

### Extern functions

The host registers a dispatcher. Script calls into it by import slot.

```c
extern(0) void inspect(byte status, bool ready, int64 total);

void script_main() {
    inspect(255, 1, 7);
    return;
}
```

Typed host interop works for current primitive value kinds, including integers, floats, `byte`, `bool`, and `uint64`.

## Pointers

Pointer types exist in the type system and pointer syntax such as `int32*` parses as a type.

However, pointer usage is not really available at source level yet because the language currently lacks:

- address-of syntax
- dereference syntax
- pointer arithmetic syntax exposed through the parser

Treat pointers as reserved or incomplete rather than usable.

## Current Limits and Gotchas

### Incomplete pointer operators

Unary address-of `&x` and dereference `*ptr` are not currently supported. The same characters are supported as binary bitwise AND and multiplication operators.

### Aggregate restrictions

Struct definitions are top-level only. Struct and array values may be uninitialized extern globals or zero-initialized internal globals, but cannot be locals, parameters, or return values. Whole-aggregate assignment and aggregate initialization are not supported yet.

Each struct is one contiguous memory block. Fields are laid out in declaration order using natural alignment, and the final size is rounded to the maximum field alignment. The compiler retains the ordered fields, member names, types, and relative byte offsets; the runtime image does not contain struct reflection metadata.

Fixed arrays use `type name[count]` syntax. Constant indexes are checked at compile time. Dynamic indexes have VM segment bounds protection but no per-array bounds check.

### No local declarations in `for` initializers

This is still not supported:

```c
for (int i = 0; i < 4; i = i + 1) {
}
```

### Float type spelling

Use `float`, `float32`, or `f32` for 32-bit floating-point values. Use `double`, `float64`, or `f64` for 64-bit floating-point values.

## Minimal Working Examples

### Minimal script

```c
void script_main() {
    return;
}
```

### Integer arithmetic

```c
int script_main() {
    return 40 + 2;
}
```

### Float arithmetic

```c
float64 script_main() {
    return 1.5 + 2.25;
}
```

### Mixed int and float

```c
int base;

float64 script_main() {
    base = 2;
    return base + 1.5;
}
```

### Globals and loops

```c
int counter;
int total;

void script_main() {
    counter = 0;
    total = 0;
    while (counter < 5) {
        if (counter == 3) {
            total = total + 10;
        } else {
            total = total + 1;
        }
        counter = counter + 1;
    }
    return;
}
```

### Extern memory

```c
extern int value;

void script_main() {
    value = value + 9;
    return;
}
```

### Extern function

```c
extern(0) void log_value(int value);

void script_main() {
    log_value(42);
    return;
}
```

## Summary

The current language is a compact VM scripting language with:

- Typed primitive values
- Global state
- Functions
- Numeric expressions
- Structured control flow
- Host interop via extern variables and extern functions

It is best suited today for small rule scripts, control-flow-heavy logic, and host-integrated numeric scripting, rather than general-purpose programming.