package cova

import (
	"strings"
	"testing"
)

func TestTokenizeComparisonOperators(t *testing.T) {
	src := "if (a == b != c <= d >= e) { return; }"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	var operators []TokenKind
	for _, token := range tokens {
		switch token.Kind {
		case TokEqual, TokNotEqual, TokLessEqual, TokGreaterEqual:
			operators = append(operators, token.Kind)
		}
	}
	expected := []TokenKind{TokEqual, TokNotEqual, TokLessEqual, TokGreaterEqual}
	if len(operators) != len(expected) {
		t.Fatalf("expected %d operators, got %d (%v)", len(expected), len(operators), operators)
	}
	for index, want := range expected {
		if operators[index] != want {
			t.Fatalf("expected operator %d to have kind %d, got %d", index, want, operators[index])
		}
	}
}

func TestTokenizeLogicalOperators(t *testing.T) {
	src := "if (ready && active || enabled) { return; }"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	var operators []TokenKind
	for _, token := range tokens {
		if token.Kind == TokLogicalAnd || token.Kind == TokLogicalOr {
			operators = append(operators, token.Kind)
		}
	}
	expected := []TokenKind{TokLogicalAnd, TokLogicalOr}
	if len(operators) != len(expected) {
		t.Fatalf("expected %d operators, got %d (%v)", len(expected), len(operators), operators)
	}
	for index, want := range expected {
		if operators[index] != want {
			t.Fatalf("expected operator %d to have kind %d, got %d", index, want, operators[index])
		}
	}
}

func TestTokenizeCStyleOperatorsAndPunctuators(t *testing.T) {
	src := "+ - * / % ++ -- -> == != <= >= && || << >> += -= *= /= %= <<= >>= &= |= ^= & | ^ ~ = < > ! " +
		"( ) { } [ ] ; , : :: . ... ? # ##"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)

	expected := []TokenKind{
		TokPlus, TokMinus, TokStar, TokSlash, TokPercent,
		TokIncrement, TokDecrement, TokArrow,
		TokEqual, TokNotEqual, TokLessEqual, TokGreaterEqual,
		TokLogicalAnd, TokLogicalOr, TokShiftLeft, TokShiftRight,
		TokPlusAssign, TokMinusAssign, TokStarAssign, TokSlashAssign, TokPercentAssign,
		TokShiftLeftAssign, TokShiftRightAssign, TokAndAssign, TokOrAssign, TokXorAssign,
		TokAmp, TokPipe, TokCaret, TokTilde, TokAssign, TokLess, TokGreater, TokBang,
		TokLParen, TokRParen, TokLBrace, TokRBrace, TokLBracket, TokRBracket,
		TokSemicolon, TokComma, TokColon, TokColonColon, TokDot, TokEllipsis,
		TokQuestion, TokHash, TokHashHash,
	}
	if len(tokens) != len(expected)+1 {
		t.Fatalf("expected %d tokens plus eof, got %d", len(expected), len(tokens))
	}
	for index, want := range expected {
		if tokens[index].Kind != want {
			t.Fatalf("token %d: expected kind=%d, got kind=%d", index, want, tokens[index].Kind)
		}
	}
}

func TestTokenizeColonDelimiter(t *testing.T) {
	src := "switch (value) { case 1: default: }"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	colonCount := 0
	for _, token := range tokens {
		if token.Kind == TokColon {
			colonCount++
		}
	}
	if colonCount != 2 {
		t.Fatalf("expected 2 colon delimiters, got %d", colonCount)
	}
}

func TestTokenizeControlFlowKeywords(t *testing.T) {
	src := "break case continue default else for switch while"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	expected := []TokenKind{TokBreak, TokCase, TokContinue, TokDefault, TokElse, TokFor, TokSwitch, TokWhile}
	if len(tokens) != len(expected)+1 {
		t.Fatalf("expected %d tokens plus eof, got %d", len(expected), len(tokens))
	}
	for index, want := range expected {
		token := tokens[index]
		if token.Kind != want {
			t.Fatalf("expected token %d to have kind %d, got %d", index, want, token.Kind)
		}
	}
}

func TestTokenizeTypeAliasKeywords(t *testing.T) {
	src := "i8 i16 i32 i64 u8 u16 u32 u64 float f32 f64 double"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	expected := []TokenKind{TokI8, TokI16, TokI32, TokI64, TokU8, TokU16, TokU32, TokU64, TokFloat, TokF32, TokF64, TokDouble}
	if len(tokens) != len(expected)+1 {
		t.Fatalf("expected %d tokens plus eof, got %d", len(expected), len(tokens))
	}
	for index, want := range expected {
		if tokens[index].Kind != want {
			t.Fatalf("expected token %d to have kind %d, got %d", index, want, tokens[index].Kind)
		}
	}
}

func TestTokenizeConstKeyword(t *testing.T) {
	src := "const"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	if len(tokens) != 2 {
		t.Fatalf("expected const plus eof, got %d tokens", len(tokens))
	}
	if tokens[0].Kind != TokConst || tokens[0].Text != "const" {
		t.Fatalf("expected const keyword token, got kind=%d text=%q", tokens[0].Kind, tokens[0].Text)
	}
}

func TestTokenizeBooleanLiteralKeywords(t *testing.T) {
	src := "true false"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	expected := []TokenKind{TokTrue, TokFalse}
	if len(tokens) != len(expected)+1 {
		t.Fatalf("expected %d tokens plus eof, got %d", len(expected), len(tokens))
	}
	for index, want := range expected {
		token := tokens[index]
		if token.Kind != want {
			t.Fatalf("expected token %d to have kind %d, got %d", index, want, token.Kind)
		}
	}
}

func TestTokenizeNumericLiteralSupportsIntegerFloatAndScientific(t *testing.T) {
	src := "1 2.5 6e3 7.25e-2 8E+4"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	if tokens[0].Kind != TokInteger || tokens[0].IntValue != 1 {
		t.Fatalf("expected integer payload 1, got kind=%d value=%d", tokens[0].Kind, tokens[0].IntValue)
	}
	for index, want := range []float64{2.5, 6000, 0.0725, 80000} {
		if token := tokens[index+1]; token.Kind != TokFloat32 || token.FloatValue != want {
			t.Fatalf("expected float32 token %d value %v, got kind=%d value=%v", index+1, want, token.Kind, token.FloatValue)
		}
	}
}

func TestTokenizeNumericLiteralSupportsFloatSuffixes(t *testing.T) {
	src := "0.5 1.5f 2.5d 6e3F 7.25E-2D"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	expectedKinds := []TokenKind{TokFloat32, TokFloat32, TokFloat64, TokFloat32, TokFloat64}
	for index, want := range expectedKinds {
		if tokens[index].Kind != want {
			t.Fatalf("expected token %d kind %d, got %d", index, want, tokens[index].Kind)
		}
	}
}

func TestTokenizeNumericLiteralRejectsInvalidScientificNotation(t *testing.T) {
	for _, src := range []string{"1e", "1e+", "1e-", "2.E3"} {
		ctx := NewContext()
		if _, ok := Tokenize(ctx, src); ok {
			t.Fatalf("expected Tokenize to reject %q", src)
		}
	}
}

func TestTokenizeStringLiteralSupportsEscapes(t *testing.T) {
	src := "\"asset\\npath\\\"\\\\tail\""
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	if len(tokens) != 2 {
		t.Fatalf("expected string token plus eof, got %d tokens", len(tokens))
	}
	if tokens[0].Kind != TokString {
		t.Fatalf("expected first token kind %d, got %d", TokString, tokens[0].Kind)
	}
	if want := "asset\npath\"\\tail"; tokens[0].Text != want {
		t.Fatalf("expected decoded string %q, got %q", want, tokens[0].Text)
	}
}

func TestTokenizeStringLiteralRejectsUnterminatedLiteral(t *testing.T) {
	ctx := NewContext()
	_, ok := Tokenize(ctx, "\"asset/button_off")
	if ok {
		t.Fatal("expected unterminated string literal error")
	}
	issues := issueDescriptions(ctx)
	if len(issues) == 0 || !strings.Contains(issues[0], "unterminated string literal") {
		t.Fatalf("expected unterminated string error, got %v", issues)
	}
}

func TestTokenizeNewlinesCommentsAndSpans(t *testing.T) {
	src := "one\r\ntwo // comment\n/* block\ncomment */ three"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	expected := []TokenKind{TokIdent, TokNewline, TokIdent, TokNewline, TokIdent, TokEOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for index, want := range expected {
		if tokens[index].Kind != want {
			t.Fatalf("token %d: expected kind %d, got %d", index, want, tokens[index].Kind)
		}
	}
	if token := tokens[1]; token.Line != 1 || token.Column != 4 || token.Offset != 3 || token.Length != 2 {
		t.Fatalf("unexpected CRLF span: %+v", token)
	}
	if token := tokens[4]; token.Text != "three" || token.Line != 4 || token.Column != 12 {
		t.Fatalf("unexpected token after block comment: %+v", token)
	}
}

func TestTokenizeCommentsTrackCRLineEndings(t *testing.T) {
	src := "// line\rone\r/* block\r\ntwo */three"
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, src)
	expected := []TokenKind{TokNewline, TokIdent, TokNewline, TokIdent, TokEOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for index, want := range expected {
		if tokens[index].Kind != want {
			t.Fatalf("token %d: expected kind %d, got %d", index, want, tokens[index].Kind)
		}
	}
	if token := tokens[3]; token.Text != "three" || token.Line != 4 || token.Column != 7 {
		t.Fatalf("unexpected token after CR comments: %+v", token)
	}
}

func TestTokenizeRejectsUnterminatedBlockComment(t *testing.T) {
	ctx := NewContext()
	_, ok := Tokenize(ctx, "value /* missing")
	if ok {
		t.Fatal("expected unterminated block comment error")
	}
	issues := issueDescriptions(ctx)
	if len(issues) == 0 || !strings.Contains(issues[0], "unterminated block comment") {
		t.Fatalf("expected unterminated block comment error, got %v", issues)
	}
}
