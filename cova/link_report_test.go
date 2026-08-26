package cova

import (
	"bytes"
	"strings"
	"testing"
)

func TestLinkerReportListsUnusedExternalFunctions(t *testing.T) {
	script := `
extern(0) int used();
extern(1) int never_used();
extern(2) int dead_only();
int dead() { return dead_only(); }
int script_main() { return used(); }
`
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)

	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}
	linker := NewLinker(ctx, 0, 3)
	linked, success := linker.Link(program, compiled)
	if !success {
		t.Fatalf("Link failed")
	}

	var output bytes.Buffer
	if success := linker.Report(&output, compiled, linked); !success {
		t.Fatalf("Report failed")
	}
	if !strings.Contains(output.String(), "External Functions: 3 functions, 2 unused\n") {
		t.Fatalf("unexpected external count:\n%s", output.String())
	}
	if !strings.Contains(output.String(), "Unused External Functions: 2\n  never_used\n  dead_only\n") {
		t.Fatalf("unexpected unused external list:\n%s", output.String())
	}
}

func TestLinkerReportOmitsUnusedSectionWhenAllExternalsUsed(t *testing.T) {
	script := `
extern(0) int first();
extern(1) int second();
int script_main() { return first() + second(); }
`
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)

	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}
	linker := NewLinker(ctx, 0, 2)
	linked, success := linker.Link(program, compiled)
	if !success {
		t.Fatalf("Link failed")
	}

	var output bytes.Buffer
	if success := linker.Report(&output, compiled, linked); !success {
		t.Fatalf("Report failed:\n%s", output.String())
	}
	if !strings.Contains(output.String(), "External Functions: 2 functions, 0 unused\n") {
		t.Fatalf("unexpected external count:\n%s", output.String())
	}
	if strings.Contains(output.String(), "Unused External Functions:") {
		t.Fatalf("unexpected unused section:\n%s", output.String())
	}
}

func TestLinkerStillValidatesUnusedExternalFunctionCapacity(t *testing.T) {
	script := `
extern(3) int unused();
int script_main() { return 1; }
`
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)

	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}
	if _, success := NewLinker(ctx, 0, 1).Link(program, compiled); success {
		t.Fatal("expected unused external function capacity error")
	}
}
