package cova

import "testing"

func TestCompileComputesByteLayoutMetadata(t *testing.T) {
	script := `
extern uint64 flags;
bool ready;
int32 count;
int16 small;

void script_main(int8 tag, uint64 mask) {
	return;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)

	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}

	if len(compiled.ProgramSymbols.ExternSymbols) != 1 {
		t.Fatalf("expected 1 extern symbol, got %d", len(compiled.ProgramSymbols.ExternSymbols))
	}
	extern := compiled.ProgramSymbols.ExternSymbols[0]
	if extern.ByteOffset != 0 || extern.ByteSize != 8 || extern.ByteAlignment != 8 {
		t.Fatalf("unexpected extern layout: offset=%d size=%d align=%d", extern.ByteOffset, extern.ByteSize, extern.ByteAlignment)
	}

	if len(compiled.ProgramSymbols.BSSSymbols) != 3 {
		t.Fatalf("expected 3 bss symbols, got %d", len(compiled.ProgramSymbols.BSSSymbols))
	}
	if compiled.ProgramSymbols.BSSSymbols[0].ByteOffset != 0 {
		t.Fatalf("expected ready at bss offset 0, got %d", compiled.ProgramSymbols.BSSSymbols[0].ByteOffset)
	}
	if compiled.ProgramSymbols.BSSSymbols[1].ByteOffset != 4 {
		t.Fatalf("expected count at bss offset 4, got %d", compiled.ProgramSymbols.BSSSymbols[1].ByteOffset)
	}
	if compiled.ProgramSymbols.BSSSymbols[2].ByteOffset != 8 {
		t.Fatalf("expected small at bss offset 8, got %d", compiled.ProgramSymbols.BSSSymbols[2].ByteOffset)
	}
	if compiled.BSSByteSize != 10 {
		t.Fatalf("expected bss byte size 10, got %d", compiled.BSSByteSize)
	}
	if compiled.FrameByteSize != 16 {
		t.Fatalf("expected frame byte size 16, got %d", compiled.FrameByteSize)
	}
	if len(compiled.Functions) != 1 {
		t.Fatalf("expected 1 compiled function, got %d", len(compiled.Functions))
	}
	if compiled.Functions[0].FrameByteSize != 16 {
		t.Fatalf("expected script function frame byte size 16, got %d", compiled.Functions[0].FrameByteSize)
	}
}

func TestCompileAlignsAutomaticExternOffsets(t *testing.T) {
	script := `
extern uint8 prefix;
extern int32 value;

void script_main() {
	return;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)

	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}
	if len(compiled.ProgramSymbols.ExternSymbols) != 2 {
		t.Fatalf("expected 2 extern symbols, got %d", len(compiled.ProgramSymbols.ExternSymbols))
	}
	if got := compiled.ProgramSymbols.ExternSymbols[0].ByteOffset; got != 0 {
		t.Fatalf("expected prefix at offset 0, got %d", got)
	}
	if got := compiled.ProgramSymbols.ExternSymbols[1].ByteOffset; got != 4 {
		t.Fatalf("expected value at offset 4, got %d", got)
	}
}

func TestLinkAttachesDebugSymbolsSeparately(t *testing.T) {
	script := `
extern uint64 flags;
bool ready;

void script_main() {
	return;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)

	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}
	linked, success := NewLinker(ctx, 16, 0).Link(program, compiled)
	if !success {
		t.Fatalf("Link failed")
	}
	if linked.DebugSymbols == nil {
		t.Fatal("expected linked debug symbols")
	}
	if len(linked.DebugSymbols.ExternSymbols) != 1 {
		t.Fatalf("expected 1 extern debug symbol, got %d", len(linked.DebugSymbols.ExternSymbols))
	}
	if len(linked.DebugSymbols.BSSSymbols) != 1 {
		t.Fatalf("expected 1 bss debug symbol, got %d", len(linked.DebugSymbols.BSSSymbols))
	}
	if _, ok := linked.DebugSymbols.Symbols["ready"]; !ok {
		t.Fatal("expected named debug symbol for ready")
	}
}

func TestCompilePlacesConstGlobalsInConstLayout(t *testing.T) {
	script := `
const int32 threshold = 7;
const uint8* const asset_path = "asset/button_off";

void script_main() {
	return;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)

	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}
	if len(compiled.ProgramSymbols.ConstSymbols) != 2 {
		t.Fatalf("expected 2 const symbols, got %d", len(compiled.ProgramSymbols.ConstSymbols))
	}
	if len(compiled.ProgramSymbols.DataSymbols) != 0 {
		t.Fatalf("expected no data symbols, got %d", len(compiled.ProgramSymbols.DataSymbols))
	}
	if compiled.ConstByteSize <= 8 {
		t.Fatalf("expected const image to include globals plus literal bytes, got %d", compiled.ConstByteSize)
	}
	threshold := compiled.ProgramSymbols.Symbols["threshold"]
	assetPath := compiled.ProgramSymbols.Symbols["asset_path"]
	if threshold.Scope != ScopeConst || assetPath.Scope != ScopeConst {
		t.Fatalf("expected const scopes, got threshold=%d asset_path=%d", threshold.Scope, assetPath.Scope)
	}
	if threshold.ByteOffset != 0 {
		t.Fatalf("expected first const global at offset 0, got %d", threshold.ByteOffset)
	}
	if assetPath.ByteOffset < threshold.ByteOffset+threshold.ByteSize {
		t.Fatalf("expected pointer const global after threshold storage, got %d", assetPath.ByteOffset)
	}
	if !assetPath.Type.IsConst || assetPath.Type.Base == nil || !assetPath.Type.Base.IsConst {
		t.Fatalf("expected const pointer to const uint8, got %v", assetPath.Type)
	}
}
