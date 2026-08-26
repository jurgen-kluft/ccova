package cova

import (
	"math"
	"testing"
)

func TestOptimizeFoldsNestedArithmeticWithDestinationType(t *testing.T) {
	program := parseOptimizerTestProgram(t, `
int8 folded;
void script_main() {
	folded = (120 + 10) * 2;
}
`)
	ctx := NewContext()
	if !Optimize(ctx, program) {
		t.Fatalf("Optimize failed: %v", issueDescriptions(ctx))
	}
	assignment := program.Functions[0].Body.Statements[0].(*AstAssignStmt)
	literal, ok := assignment.Value.(*AstNumberLiteral)
	if !ok {
		t.Fatalf("expected folded number literal, got %T", assignment.Value)
	}
	if literal.IntValue != 4 {
		t.Fatalf("expected int8 wrapping result 4, got %d", literal.IntValue)
	}
}

func TestOptimizeFoldsComparisonAndLogicalExpressions(t *testing.T) {
	program := parseOptimizerTestProgram(t, `
void script_main() {
	int result = (2 + 3 == 5) && (9 > 4);
}
`)
	ctx := NewContext()
	if !Optimize(ctx, program) {
		t.Fatalf("Optimize failed: %v", issueDescriptions(ctx))
	}
	declaration := program.Functions[0].Body.Statements[0].(*AstLocalDeclStmt)
	literal, ok := declaration.Initializer.(*AstNumberLiteral)
	if !ok || literal.IntValue != 1 {
		t.Fatalf("expected folded true literal, got %#v", declaration.Initializer)
	}
}

func TestOptimizeSkipsUnreachableLogicalBranch(t *testing.T) {
	program := parseOptimizerTestProgram(t, `
void script_main() {
	int result = false && (1 / 0);
}
`)
	ctx := NewContext()
	if !Optimize(ctx, program) {
		t.Fatalf("Optimize evaluated unreachable branch: %v", issueDescriptions(ctx))
	}
}

func TestOptimizeReportsReachableDivisionByZero(t *testing.T) {
	program := parseOptimizerTestProgram(t, `
void script_main() {
	int result = 1 / 0;
}
`)
	ctx := NewContext()
	if Optimize(ctx, program) || !hasIssueContaining(ctx, "optimization error on line 3: division by zero") {
		t.Fatalf("expected line-numbered division error, got %v", issueDescriptions(ctx))
	}
}

func TestOptimizeFoldsGlobalInitializerAndCallArgument(t *testing.T) {
	program := parseOptimizerTestProgram(t, `
int total = 2 + 3 * 4;
extern(0) void consume(float32 value);
void script_main() {
	consume(1.25f + 2.5f);
}
`)
	ctx := NewContext()
	if !Optimize(ctx, program) {
		t.Fatalf("Optimize failed: %v", issueDescriptions(ctx))
	}
	global, ok := program.Decls[0].Initializer.(*AstNumberLiteral)
	if !ok || global.IntValue != 14 {
		t.Fatalf("expected folded global initializer 14, got %#v", program.Decls[0].Initializer)
	}
	if _, ok := NewCompiler(ctx).Compile(program); !ok {
		t.Fatalf("Compile failed after folding global initializer")
	}
	call := program.Functions[0].Body.Statements[0].(*AstExprStmt).Expr.(*AstCallExpr)
	argument, ok := call.Args[0].(*AstNumberLiteral)
	if !ok || !argument.IsFloat || argument.FloatType != Float32Type || argument.FloatValue != 3.75 {
		t.Fatalf("expected folded float32 call argument, got %#v", call.Args[0])
	}
}

func TestOptimizePreservesFloatComparisonSemantics(t *testing.T) {
	nan := &AstNumberLiteral{FloatValue: math.NaN(), IsFloat: true, FloatType: Float64Type, Line: 1}
	program := &AstProgramNode{Functions: []*AstFunctionNode{{
		Name:       "script_main",
		ReturnType: VoidType,
		Body: &AstBlockStmt{Statements: []AstStmtNode{&AstLocalDeclStmt{
			Name:        "result",
			Type:        Int32Type,
			Initializer: &AstBinaryExpr{Op: "!=", Left: nan, Right: nan, Line: 1},
		}}},
	}}}
	ctx := NewContext()
	if !Optimize(ctx, program) {
		t.Fatalf("Optimize failed: %v", issueDescriptions(ctx))
	}
	literal := program.Functions[0].Body.Statements[0].(*AstLocalDeclStmt).Initializer.(*AstNumberLiteral)
	if literal.IntValue != 1 {
		t.Fatalf("expected NaN != NaN to fold true, got %d", literal.IntValue)
	}
}

func TestOptimizeMatchesUnoptimizedRuntimeResult(t *testing.T) {
	source := `
int result;
void script_main() {
	int8 narrow = (120 + 10) * 2;
	float32 fraction = (7.0f / 3.0f) * 3.0f;
	result = narrow + (fraction > 6.9f);
}
`
	unoptimized := runOptimizerTestProgram(t, source, false)
	optimized := runOptimizerTestProgram(t, source, true)
	if optimized != unoptimized {
		t.Fatalf("optimized result %d differs from unoptimized result %d", optimized, unoptimized)
	}
}

func TestOptimizeRejectsNilProgram(t *testing.T) {
	ctx := NewContext()
	if Optimize(ctx, nil) || !hasIssueContaining(ctx, "optimization error: program is nil") {
		t.Fatalf("expected nil program error, got %v", issueDescriptions(ctx))
	}
}

func TestOptimizeSuccessIgnoresExistingContextErrors(t *testing.T) {
	ctx := NewContext()
	ctx.AddError("old error")
	if !Optimize(ctx, &AstProgramNode{}) {
		t.Fatalf("Optimize failed because of existing context errors: %v", issueDescriptions(ctx))
	}
}

func runOptimizerTestProgram(t *testing.T, source string, optimize bool) int32 {
	t.Helper()
	ctx := NewContext()
	program := parseOptimizerTestProgram(t, source)
	if optimize {
		if !Optimize(ctx, program) {
			t.Fatalf("Optimize failed: %v", issueDescriptions(ctx))
		}
	}
	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}
	linked, success := NewLinker(ctx, 0, 0).Link(program, compiled)
	if !success {
		t.Fatalf("Link failed")
	}
	vm := NewVM(testFrameCapacityBytes)
	if status := vm.Run(linked); status != VMStatusOK {
		t.Fatalf("Run failed: %s", status)
	}
	offset := linked.DebugSymbols.Symbols["result"].ByteOffset
	result, status := vm.memory.ReadInt32(makeAddress(segmentBSS, offset))
	if status != VMStatusOK {
		t.Fatalf("ReadInt32 result failed: %s", status)
	}
	return result
}

func parseOptimizerTestProgram(t *testing.T, source string) *AstProgramNode {
	t.Helper()
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, source)
	return mustParseTokens(t, ctx, tokens)
}
