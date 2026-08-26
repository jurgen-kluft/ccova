package cova

import (
	"strings"
	"testing"
)

func parseProgram(t *testing.T, script string) *AstProgramNode {
	t.Helper()
	return mustParseScript(t, script)
}

func TestParseExpressions(t *testing.T) {
	script := `
bool bool1;
bool bool2;
bool bool3;
int counter;

int script_main() {
	int total = (1 + 2) * 3;
	if (bool1 && (bool2 || bool3)) {
		total = total + counter;
	}
	return total + helper(counter, 4);
}

int helper(int left, int right) {
	return left + right;
}
`

	program := parseProgram(t, script)
	if len(program.Decls) != 4 {
		t.Fatalf("expected 4 top-level declarations, got %d", len(program.Decls))
	}
	if len(program.Functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(program.Functions))
	}
	mainFn := program.Functions[0]
	if len(mainFn.Body.Statements) != 3 {
		t.Fatalf("expected 3 statements in script_main, got %d", len(mainFn.Body.Statements))
	}
	if _, ok := mainFn.Body.Statements[1].(*AstIfStmt); !ok {
		t.Fatalf("expected second statement to be if, got %T", mainFn.Body.Statements[1])
	}
}

func TestParseRejectsLocalDeclarationInForInitializer(t *testing.T) {
	script := `
int script_main() {
	for (int i = 0; i < 4; i = i + 1) {
	}
	return 0;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	if _, ok := Parse(ctx, tokens); ok {
		t.Fatal("expected parser to reject local declaration in for initializer")
	}
}

func TestParseControlFlow(t *testing.T) {
	script := `
int counter;
bool cond;

int helper(int left, int right) {
	return left + right;
}

int script_main() {
	int total = helper(counter, 1 + 2);
	while (counter < 3 && cond || total == 0) {
		total = total + helper(counter, 2);
		counter = counter + 1;
	}
	for (counter = 0; counter < 2; counter = counter + 1) {
		switch (helper(counter, total)) {
		case 1:
			total = total + 10;
			break;
		default:
			total = total + 1;
			continue;
		}
	}
	return total;
}
`

	program := parseProgram(t, script)
	mainFn := program.Functions[1]
	if len(mainFn.Body.Statements) != 4 {
		t.Fatalf("expected 4 top-level statements in script_main, got %d", len(mainFn.Body.Statements))
	}
	if _, ok := mainFn.Body.Statements[1].(*AstWhileStmt); !ok {
		t.Fatalf("expected second statement to be while, got %T", mainFn.Body.Statements[1])
	}
	if _, ok := mainFn.Body.Statements[2].(*AstForStmt); !ok {
		t.Fatalf("expected third statement to be for, got %T", mainFn.Body.Statements[2])
	}
	if _, ok := mainFn.Body.Statements[3].(*AstReturnStmt); !ok {
		t.Fatalf("expected fourth statement to be return, got %T", mainFn.Body.Statements[3])
	}
}

func TestParseLocalInitializersAndReturns(t *testing.T) {
	script := `
float32 ratio;
bool ready;

float64 helper(float32 value) {
	return value + 2.5d;
}

float64 script_main() {
	float32 base = 1.5f;
	float64 total = helper((base + ratio) * 2.0f);
	if (ready) {
		return total + 3e1D;
	}
	return total;
}
`

	program := parseProgram(t, script)
	mainFn := program.Functions[1]
	if len(mainFn.Body.Statements) != 4 {
		t.Fatalf("expected 4 statements in script_main, got %d", len(mainFn.Body.Statements))
	}
	if _, ok := mainFn.Body.Statements[0].(*AstLocalDeclStmt); !ok {
		t.Fatalf("expected first statement to be local declaration, got %T", mainFn.Body.Statements[0])
	}
	if _, ok := mainFn.Body.Statements[1].(*AstLocalDeclStmt); !ok {
		t.Fatalf("expected second statement to be local declaration, got %T", mainFn.Body.Statements[1])
	}
	if _, ok := mainFn.Body.Statements[2].(*AstIfStmt); !ok {
		t.Fatalf("expected third statement to be if, got %T", mainFn.Body.Statements[2])
	}
	if _, ok := mainFn.Body.Statements[3].(*AstReturnStmt); !ok {
		t.Fatalf("expected fourth statement to be return, got %T", mainFn.Body.Statements[3])
	}
}

func TestParseExpandedPrimitiveTypes(t *testing.T) {
	script := `
extern int8 small_value;
extern uint64 flags;
bool ready;
float32 ratio;

float64 script_main(int input, byte tag) {
	return ratio;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)
	if len(program.Decls) != 4 {
		t.Fatalf("expected 4 top-level declarations, got %d", len(program.Decls))
	}
	if program.Decls[0].Type != Int8Type {
		t.Fatalf("expected first decl type int8, got %v", program.Decls[0].Type)
	}
	if program.Decls[1].Type != Uint64Type {
		t.Fatalf("expected second decl type uint64, got %v", program.Decls[1].Type)
	}
	if program.Decls[2].Type != BoolType {
		t.Fatalf("expected third decl type bool, got %v", program.Decls[2].Type)
	}
	if program.Decls[3].Type != Float32Type {
		t.Fatalf("expected fourth decl type float32, got %v", program.Decls[3].Type)
	}
	if len(program.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(program.Functions))
	}
	function := program.Functions[0]
	if function.ReturnType != Float64Type {
		t.Fatalf("expected return type float64, got %v", function.ReturnType)
	}
	if len(function.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(function.Params))
	}
	if function.Params[0].Type != IntType {
		t.Fatalf("expected int param to alias int32, got %v", function.Params[0].Type)
	}
	if function.Params[1].Type != ByteType {
		t.Fatalf("expected second param type byte, got %v", function.Params[1].Type)
	}
}

func TestLookupNamedTypeIntAlias(t *testing.T) {
	if LookupNamedType("int") != Int32Type {
		t.Fatalf("expected int to alias int32")
	}
}

func TestParsePrimitiveTypeAliases(t *testing.T) {
	program := parseProgram(t, `
i8 signed8;
i16 signed16;
i32 signed32;
i64 signed64;
u8 unsigned8;
u16 unsigned16;
u32 unsigned32;
u64 unsigned64;
f32 float32_value;
f64 float64_value;
float float_value;
double double_value;
char character;

void script_main() {
	return;
}
`)

	expected := []*Type{
		Int8Type, Int16Type, Int32Type, Int64Type,
		Uint8Type, Uint16Type, Uint32Type, Uint64Type,
		Float32Type, Float64Type, Float32Type, Float64Type, Uint8Type,
	}
	if len(program.Decls) != len(expected) {
		t.Fatalf("expected %d declarations, got %d", len(expected), len(program.Decls))
	}
	for index, want := range expected {
		if got := program.Decls[index].Type; got != want {
			t.Fatalf("declaration %d: expected type %v, got %v", index, want, got)
		}
	}
}

func TestLookupNamedTypeAliases(t *testing.T) {
	expected := map[string]*Type{
		"char":   Uint8Type,
		"float":  Float32Type,
		"double": Float64Type,
		"i8":     Int8Type, "i16": Int16Type, "i32": Int32Type, "i64": Int64Type,
		"u8": Uint8Type, "u16": Uint16Type, "u32": Uint32Type, "u64": Uint64Type,
		"f32": Float32Type, "f64": Float64Type,
	}
	for name, want := range expected {
		if got := LookupNamedType(name); got != want {
			t.Fatalf("expected %s to alias %v, got %v", name, want, got)
		}
	}
	if CharType != Uint8Type {
		t.Fatalf("expected CharType to alias Uint8Type")
	}
}

func TestStructTypePreservesNaturalLayoutAndNamedFields(t *testing.T) {
	ctx := NewContext()
	typ, ok := NewStructType(ctx, "sample_t", []StructField{
		{Name: "first", Type: Uint8Type},
		{Name: "second", Type: Uint32Type},
		{Name: "third", Type: Uint16Type},
	})
	if !ok {
		t.Fatalf("NewStructType failed: %v", issueDescriptions(ctx))
	}

	if typ.Kind != TypeStruct || typ.Size != 12 || typ.Alignment() != 4 {
		t.Fatalf("unexpected struct layout: kind=%d size=%d alignment=%d", typ.Kind, typ.Size, typ.Alignment())
	}
	wantOffsets := []int{0, 4, 8}
	for index, wantOffset := range wantOffsets {
		if got := typ.Struct.Fields[index].ByteOffset; got != wantOffset {
			t.Fatalf("field %d offset: got %d, want %d", index, got, wantOffset)
		}
	}
	if got := typ.Struct.FieldsByName["second"]; got != 1 {
		t.Fatalf("second field lookup: got %d, want 1", got)
	}
}

func TestParseStructAndAutomaticExternArray(t *testing.T) {
	program := parseProgram(t, `
struct light_state_t {
	uint32 m_hsv;
	uint8 m_on_off;
};

struct home_state_t {
	char m_date[16];
	light_state_t m_light;
};

extern home_state_t g_home;

int script_main() {
	return 0;
}
`)

	if len(program.Structs) != 2 {
		t.Fatalf("expected 2 struct definitions, got %d", len(program.Structs))
	}
	homeType := program.Structs[1].Type
	if homeType.Kind != TypeStruct || homeType.Size != 24 || homeType.Alignment() != 4 {
		t.Fatalf("unexpected home layout: kind=%d size=%d alignment=%d", homeType.Kind, homeType.Size, homeType.Alignment())
	}
	dateField := homeType.Struct.Fields[0]
	if dateField.Type.Kind != TypeArray || dateField.Type.Base != Uint8Type || dateField.Type.ElementCount != 16 {
		t.Fatalf("unexpected date field type: %#v", dateField.Type)
	}
	if len(program.Decls) != 1 || program.Decls[0].Index != -1 || program.Decls[0].Type != homeType {
		t.Fatalf("unexpected extern declaration: %#v", program.Decls)
	}
}

func TestParseRejectsExplicitExternVariableOffset(t *testing.T) {
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, "extern(4) int value;")
	if _, ok := Parse(ctx, tokens); ok {
		t.Fatal("expected automatic-offset migration error")
	}
	issues := issueDescriptions(ctx)
	if len(issues) == 0 || !strings.Contains(issues[0], "offsets are automatic") {
		t.Fatalf("expected automatic-offset migration error, got %v", issues)
	}
}

func TestCompileRejectsAggregateLocalAndConstantOutOfBoundsIndex(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		message string
	}{
		{
			name: "aggregate local",
			script: `
struct value_t { int member; };
void script_main() { value_t local; return; }
`,
			message: "cannot have aggregate type",
		},
		{
			name: "constant index",
			script: `
struct value_t { byte values[2]; };
extern value_t value;
void script_main() { value.values[2] = 1; return; }
`,
			message: "outside [0, 2)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := NewContext()
			program := parseProgram(t, test.script)
			_, err := NewCompiler(ctx).Compile(program)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected %q compile error, got %v", test.message, err)
			}
		})
	}
}

func TestParseScientificNotationLiteralAsFloat(t *testing.T) {
	script := `
float64 script_main() {
	return 1e3;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)
	ret, ok := program.Functions[0].Body.Statements[0].(*AstReturnStmt)
	if !ok {
		t.Fatalf("expected return statement, got %T", program.Functions[0].Body.Statements[0])
	}
	lit, ok := ret.Value.(*AstNumberLiteral)
	if !ok {
		t.Fatalf("expected numeric literal, got %T", ret.Value)
	}
	if !lit.IsFloat {
		t.Fatal("expected scientific notation literal to parse as float")
	}
	if lit.FloatValue != 1000 {
		t.Fatalf("expected float value 1000, got %v", lit.FloatValue)
	}
	if lit.FloatType != Float32Type {
		t.Fatalf("expected unsuffixed scientific notation literal to default to float32, got %v", lit.FloatType)
	}
}

func TestParseFloatLiteralSuffixes(t *testing.T) {
	script := `
float64 script_main() {
	0.5;
	1.5f;
	2.5d;
	return 3e1D;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)
	statements := program.Functions[0].Body.Statements
	checks := []struct {
		index int
		want  *Type
		value float64
	}{
		{index: 0, want: Float32Type, value: 0.5},
		{index: 1, want: Float32Type, value: 1.5},
		{index: 2, want: Float64Type, value: 2.5},
		{index: 3, want: Float64Type, value: 30},
	}
	for _, check := range checks {
		var expr AstExprNode
		switch node := statements[check.index].(type) {
		case *AstExprStmt:
			expr = node.Expr
		case *AstReturnStmt:
			expr = node.Value
		default:
			t.Fatalf("expected numeric expression statement, got %T", statements[check.index])
		}
		lit, ok := expr.(*AstNumberLiteral)
		if !ok {
			t.Fatalf("expected numeric literal at statement %d, got %T", check.index, expr)
		}
		if !lit.IsFloat {
			t.Fatalf("expected float literal at statement %d", check.index)
		}
		if lit.FloatType != check.want {
			t.Fatalf("expected float type %v at statement %d, got %v", check.want, check.index, lit.FloatType)
		}
		if lit.FloatValue != check.value {
			t.Fatalf("expected float value %v at statement %d, got %v", check.value, check.index, lit.FloatValue)
		}
	}
}

func TestParseStringLiteralAsExpression(t *testing.T) {
	script := `
void script_main() {
	"asset/button_off";
	return;
}
`

	program := parseProgram(t, script)
	stmt, ok := program.Functions[0].Body.Statements[0].(*AstExprStmt)
	if !ok {
		t.Fatalf("expected expression statement, got %T", program.Functions[0].Body.Statements[0])
	}
	literal, ok := stmt.Expr.(*AstStringLiteral)
	if !ok {
		t.Fatalf("expected string literal, got %T", stmt.Expr)
	}
	if literal.Value != "asset/button_off" {
		t.Fatalf("expected string literal value %q, got %q", "asset/button_off", literal.Value)
	}
}

func TestParseGlobalPointerStringInitializer(t *testing.T) {
	script := `
const uint8* asset_path = "asset/button_off";

void script_main() {
	return;
}
`

	program := parseProgram(t, script)
	decl := program.Decls[0]
	if decl.Scope != ScopeData {
		t.Fatalf("expected initialized pointer global to use data scope, got %d", decl.Scope)
	}
	if decl.Type == nil || !decl.Type.Base.IsConst || decl.Type.IsConst {
		t.Fatalf("expected parsed type const uint8*, got %v", decl.Type)
	}
	literal, ok := decl.Initializer.(*AstStringLiteral)
	if !ok {
		t.Fatalf("expected string literal initializer, got %T", decl.Initializer)
	}
	if literal.Value != "asset/button_off" {
		t.Fatalf("expected initializer value %q, got %q", "asset/button_off", literal.Value)
	}
}

func TestParseBooleanLiteralsAndLogicalPrecedence(t *testing.T) {
	script := `
int cond;

int script_main() {
	return true || false && cond == 1;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)
	ret, ok := program.Functions[0].Body.Statements[0].(*AstReturnStmt)
	if !ok {
		t.Fatalf("expected return statement, got %T", program.Functions[0].Body.Statements[0])
	}
	orExpr, ok := ret.Value.(*AstBinaryExpr)
	if !ok {
		t.Fatalf("expected top-level binary expression, got %T", ret.Value)
	}
	if orExpr.Op != "||" {
		t.Fatalf("expected top-level operator ||, got %q", orExpr.Op)
	}
	leftLiteral, ok := orExpr.Left.(*AstNumberLiteral)
	if !ok {
		t.Fatalf("expected left operand to be numeric literal, got %T", orExpr.Left)
	}
	if leftLiteral.IntValue != 1 {
		t.Fatalf("expected true to lower to 1, got %d", leftLiteral.IntValue)
	}
	andExpr, ok := orExpr.Right.(*AstBinaryExpr)
	if !ok {
		t.Fatalf("expected right operand to be binary expression, got %T", orExpr.Right)
	}
	if andExpr.Op != "&&" {
		t.Fatalf("expected right operand operator &&, got %q", andExpr.Op)
	}
	falseLiteral, ok := andExpr.Left.(*AstNumberLiteral)
	if !ok {
		t.Fatalf("expected false literal to lower to numeric literal, got %T", andExpr.Left)
	}
	if falseLiteral.IntValue != 0 {
		t.Fatalf("expected false to lower to 0, got %d", falseLiteral.IntValue)
	}
	compareExpr, ok := andExpr.Right.(*AstBinaryExpr)
	if !ok {
		t.Fatalf("expected comparison expression on && right operand, got %T", andExpr.Right)
	}
	if compareExpr.Op != "==" {
		t.Fatalf("expected comparison operator ==, got %q", compareExpr.Op)
	}
}

func TestParseLogicalPrecedenceWithExplicitGrouping(t *testing.T) {
	script := `
bool bool1;
bool bool2;
bool bool3;

int script_main() {
	return bool1 && (bool2 || bool3);
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)
	ret, ok := program.Functions[0].Body.Statements[0].(*AstReturnStmt)
	if !ok {
		t.Fatalf("expected return statement, got %T", program.Functions[0].Body.Statements[0])
	}
	andExpr, ok := ret.Value.(*AstBinaryExpr)
	if !ok {
		t.Fatalf("expected top-level binary expression, got %T", ret.Value)
	}
	if andExpr.Op != "&&" {
		t.Fatalf("expected top-level operator &&, got %q", andExpr.Op)
	}
	leftIdent, ok := andExpr.Left.(*AstIdentNode)
	if !ok {
		t.Fatalf("expected left operand identifier, got %T", andExpr.Left)
	}
	if leftIdent.Name != "bool1" {
		t.Fatalf("expected left operand bool1, got %q", leftIdent.Name)
	}
	orExpr, ok := andExpr.Right.(*AstBinaryExpr)
	if !ok {
		t.Fatalf("expected grouped right operand to be binary expression, got %T", andExpr.Right)
	}
	if orExpr.Op != "||" {
		t.Fatalf("expected grouped right operand operator ||, got %q", orExpr.Op)
	}
	leftGrouped, ok := orExpr.Left.(*AstIdentNode)
	if !ok {
		t.Fatalf("expected grouped left operand identifier, got %T", orExpr.Left)
	}
	if leftGrouped.Name != "bool2" {
		t.Fatalf("expected grouped left operand bool2, got %q", leftGrouped.Name)
	}
	rightGrouped, ok := orExpr.Right.(*AstIdentNode)
	if !ok {
		t.Fatalf("expected grouped right operand identifier, got %T", orExpr.Right)
	}
	if rightGrouped.Name != "bool3" {
		t.Fatalf("expected grouped right operand bool3, got %q", rightGrouped.Name)
	}
}

func TestParseBitwiseShiftModuloAndUnaryOperators(t *testing.T) {
	script := `
int a;
int b;
int c;

int script_main() {
	a += b | c ^ a & b == c << 1 + 1;
	return !~-a % 3;
}
`
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)

	assignment, ok := program.Functions[0].Body.Statements[0].(*AstAssignStmt)
	if !ok {
		t.Fatalf("expected compound assignment, got %T", program.Functions[0].Body.Statements[0])
	}
	if assignment.Op != AssignAdd {
		t.Fatalf("expected += assignment, got %q", assignment.Op)
	}
	orExpr, ok := assignment.Value.(*AstBinaryExpr)
	if !ok || orExpr.Op != BinaryBitwiseOr {
		t.Fatalf("expected top-level bitwise OR, got %#v", assignment.Value)
	}
	xorExpr, ok := orExpr.Right.(*AstBinaryExpr)
	if !ok || xorExpr.Op != BinaryBitwiseXor {
		t.Fatalf("expected XOR below OR, got %#v", orExpr.Right)
	}
	andExpr, ok := xorExpr.Right.(*AstBinaryExpr)
	if !ok || andExpr.Op != BinaryBitwiseAnd {
		t.Fatalf("expected AND below XOR, got %#v", xorExpr.Right)
	}
	equalExpr, ok := andExpr.Right.(*AstBinaryExpr)
	if !ok || equalExpr.Op != BinaryEqual {
		t.Fatalf("expected equality below AND, got %#v", andExpr.Right)
	}
	shiftExpr, ok := equalExpr.Right.(*AstBinaryExpr)
	if !ok || shiftExpr.Op != BinaryShiftLeft {
		t.Fatalf("expected shift below equality, got %#v", equalExpr.Right)
	}
	addExpr, ok := shiftExpr.Right.(*AstBinaryExpr)
	if !ok || addExpr.Op != BinaryAdd {
		t.Fatalf("expected addition below shift, got %#v", shiftExpr.Right)
	}

	ret := program.Functions[0].Body.Statements[1].(*AstReturnStmt)
	moduloExpr, ok := ret.Value.(*AstBinaryExpr)
	if !ok || moduloExpr.Op != BinaryModulo {
		t.Fatalf("expected top-level modulo expression, got %#v", ret.Value)
	}
	logicalNot, ok := moduloExpr.Left.(*AstUnaryExpr)
	if !ok || logicalNot.Op != UnaryLogicalNot {
		t.Fatalf("expected logical-not expression, got %#v", moduloExpr.Left)
	}
	bitwiseNot, ok := logicalNot.Operand.(*AstUnaryExpr)
	if !ok || bitwiseNot.Op != UnaryBitwiseNot {
		t.Fatalf("expected bitwise-not expression, got %#v", logicalNot.Operand)
	}
	negate, ok := bitwiseNot.Operand.(*AstUnaryExpr)
	if !ok || negate.Op != UnaryNegate {
		t.Fatalf("expected negate expression, got %#v", bitwiseNot.Operand)
	}
}

func TestParseReportsReservedExpressionPunctuators(t *testing.T) {
	tests := []struct {
		expression string
		message    string
	}{
		{"value->member", "member access"},
		{"value::member", "qualified names"},
		{"value ? 1 : 0", "ternary expressions"},
		{"value++", "increment and decrement"},
		{"#value", "preprocessor syntax"},
	}
	for _, test := range tests {
		script := "int value; int script_main() { return " + test.expression + "; }"
		ctx := NewContext()
		tokens := mustTokenize(t, ctx, script)
		if _, ok := Parse(ctx, tokens); ok {
			t.Fatalf("expected %q error for %q", test.message, test.expression)
		}
		issues := issueDescriptions(ctx)
		if len(issues) == 0 || !strings.Contains(issues[0], test.message) {
			t.Fatalf("expected %q error for %q, got %v", test.message, test.expression, issues)
		}
	}
}

func TestParseQualifiedBuiltInCall(t *testing.T) {
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, `float script_main() { return math::sin(0.0f); }`)
	program := mustParseTokens(t, ctx, tokens)
	ret := program.Functions[0].Body.Statements[0].(*AstReturnStmt)
	call, ok := ret.Value.(*AstCallExpr)
	if !ok || call.Callee != "math::sin" {
		t.Fatalf("expected math::sin call, got %#v", ret.Value)
	}
}

func TestParseLocalDeclarations(t *testing.T) {
	script := `
int script_main() {
	int count;
	const bool ready = true;
	{
		int count = 3;
		count;
	}
	return count;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)
	statements := program.Functions[0].Body.Statements
	countDecl, ok := statements[0].(*AstLocalDeclStmt)
	if !ok {
		t.Fatalf("expected first statement to be local declaration, got %T", statements[0])
	}
	if countDecl.Type != IntType || countDecl.Name != "count" || countDecl.Initializer != nil {
		t.Fatalf("unexpected first local declaration: type=%v name=%q initializer=%T", countDecl.Type, countDecl.Name, countDecl.Initializer)
	}
	readyDecl, ok := statements[1].(*AstLocalDeclStmt)
	if !ok {
		t.Fatalf("expected second statement to be local declaration, got %T", statements[1])
	}
	if readyDecl.Type == nil || readyDecl.Type.Kind != TypeBool || !readyDecl.Type.IsConst || readyDecl.Name != "ready" {
		t.Fatalf("unexpected second local declaration: type=%v name=%q", readyDecl.Type, readyDecl.Name)
	}
	readyInit, ok := readyDecl.Initializer.(*AstNumberLiteral)
	if !ok || readyInit.IntValue != 1 {
		t.Fatalf("expected bool initializer to lower to numeric literal 1, got %T value=%v", readyDecl.Initializer, readyDecl.Initializer)
	}
	innerBlock, ok := statements[2].(*AstBlockStmt)
	if !ok {
		t.Fatalf("expected third statement to be inner block, got %T", statements[2])
	}
	innerDecl, ok := innerBlock.Statements[0].(*AstLocalDeclStmt)
	if !ok {
		t.Fatalf("expected first inner statement to be local declaration, got %T", innerBlock.Statements[0])
	}
	if innerDecl.Name != "count" {
		t.Fatalf("expected inner declaration to shadow count, got %q", innerDecl.Name)
	}
}

func TestParseConstGlobalDeclaration(t *testing.T) {
	script := `
const uint8* asset_path = "asset/button_off";

void script_main() {
	return;
}
`

	program := parseProgram(t, script)
	decl := program.Decls[0]
	if decl.Type == nil || decl.Type.Kind != TypePointer || decl.Type.Base == nil {
		t.Fatalf("expected const global pointer type, got %v", decl.Type)
	}
	if decl.Type.IsConst {
		t.Fatalf("expected pointer itself to remain mutable for const uint8*, got %v", decl.Type)
	}
	if decl.Type.Base.Kind != TypeUint8 || !decl.Type.Base.IsConst {
		t.Fatalf("expected const global type uint8*, got %v", decl.Type)
	}
}

func TestParseExternConstDeclarationRejected(t *testing.T) {
	script := `extern(0) const uint8* asset_path;`
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	if _, ok := Parse(ctx, tokens); ok {
		t.Fatal("expected const extern declaration to fail")
	}
}

func TestParseConstFunctionReturnType(t *testing.T) {
	script := `
const int script_main() {
	return 0;
}
`
	program := parseProgram(t, script)
	if len(program.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(program.Functions))
	}
	if program.Functions[0].ReturnType == nil || !program.Functions[0].ReturnType.IsConst || program.Functions[0].ReturnType.Kind != TypeInt32 {
		t.Fatalf("expected const int return type, got %v", program.Functions[0].ReturnType)
	}
}

func TestParseControlFlowStatements(t *testing.T) {
	script := `
int counter;

void script_main() {
	while (counter < 10) {
		counter = counter + 1;
	}
	for (counter = 0; counter < 4; counter = counter + 1) {
		switch (counter) {
		case 1:
			break;
		default:
			continue;
		}
	}
	return;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)
	statements := program.Functions[0].Body.Statements
	if _, ok := statements[0].(*AstWhileStmt); !ok {
		t.Fatalf("expected first statement to be while, got %T", statements[0])
	}
	forStmt, ok := statements[1].(*AstForStmt)
	if !ok {
		t.Fatalf("expected second statement to be for, got %T", statements[1])
	}
	body, ok := forStmt.Body.(*AstBlockStmt)
	if !ok {
		t.Fatalf("expected for body block, got %T", forStmt.Body)
	}
	if _, ok := body.Statements[0].(*AstSwitchStmt); !ok {
		t.Fatalf("expected switch statement inside for body, got %T", body.Statements[0])
	}
}
