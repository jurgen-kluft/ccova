package cova

import (
	"fmt"
)

type exprParser interface {
	parseExpression() (AstExprNode, bool)
}

type parserCore struct {
	ctx        *Context
	tokens     []Token
	pos        int
	expr       exprParser
	namedTypes map[string]*Type
}

func newParserCore(ctx *Context, tokens []Token) parserCore {
	types := make(map[string]*Type, len(namedTypes))
	for name, typ := range namedTypes {
		types[name] = typ
	}
	return parserCore{ctx: ctx, tokens: tokens, namedTypes: types}
}

func (core *parserCore) parseExpression() (AstExprNode, bool) {
	if core.expr == nil {
		return nil, core.errorf(core.peek(), "expected expression")
	}
	return core.expr.parseExpression()
}

func (core *parserCore) expect(kind TokenKind) (Token, bool) {
	token := core.peek()
	if token.Kind != kind {
		return Token{}, core.errorf(token, expectedTokenLabel(kind))
	}
	core.pos++
	return token, true
}

func (core *parserCore) match(kind TokenKind) bool {
	if core.peek().Kind == kind {
		core.pos++
		return true
	}
	return false
}

func (core *parserCore) peek() Token {
	for core.pos < len(core.tokens) && core.tokens[core.pos].Kind == TokNewline {
		core.pos++
	}
	if core.pos >= len(core.tokens) {
		if len(core.tokens) == 0 {
			return Token{Kind: TokEOF, Line: 1, Column: 1}
		}
		last := core.tokens[len(core.tokens)-1]
		return Token{Kind: TokEOF, Line: last.Line, Column: last.Column + last.Length}
	}
	return core.tokens[core.pos]
}

func (core *parserCore) isEOF() bool {
	return core.peek().Kind == TokEOF
}

func (core *parserCore) parseType() (*Type, bool) {
	leadingConst := core.parseConstQualifier()
	token := core.peek()
	typ := tokenTypes[token.Kind]
	if typ == nil && token.Kind == TokIdent {
		typ = core.namedTypes[token.Text]
	}
	if typ == nil {
		return nil, core.errorf(token, "expected type")
	}
	core.pos++
	typ = QualifiedType(typ, leadingConst)
	if core.parseConstQualifier() {
		typ = QualifiedType(typ, true)
	}

	for core.peek().Kind == TokStar {
		core.pos++
		pointerConst := core.parseConstQualifier()
		typ = PointerToQualified(typ, pointerConst)
	}

	return typ, true
}

func (core *parserCore) parseConstQualifier() bool {
	if core.peek().Kind == TokConst {
		core.pos++
		return true
	}
	return false
}

func (core *parserCore) isTypeKeyword(token Token) bool {
	return token.Kind == TokConst || tokenTypes[token.Kind] != nil || token.Kind == TokIdent && core.namedTypes[token.Text] != nil
}

func (core *parserCore) parseArrayDeclarator(typ *Type) (*Type, bool) {
	for core.match(TokLBracket) {
		countToken, ok := core.expect(TokInteger)
		if !ok {
			return nil, false
		}
		if countToken.IntValue <= 0 || uint64(countToken.IntValue) > uint64(^uint(0)>>1) {
			return nil, core.errorf(countToken, "array element count must be a positive integer that fits int")
		}
		if _, ok := core.expect(TokRBracket); !ok {
			return nil, false
		}
		if typ == nil || typ.Kind == TypeVoid {
			return nil, core.errorf(countToken, "array element type must be complete")
		}
		if typ.Size > int(^uint(0)>>1)/int(countToken.IntValue) {
			return nil, core.errorf(countToken, "array byte size overflows int")
		}
		if uint64(typ.Size*int(countToken.IntValue)) > uint64(addressIndexMask)+1 {
			return nil, core.errorf(countToken, "array byte size exceeds the VM segment address space")
		}
		typ, ok = ArrayOf(core.ctx, typ, int(countToken.IntValue))
		if !ok {
			return nil, false
		}
	}
	return typ, true
}

func (core *parserCore) parseArguments() ([]AstExprNode, bool) {
	if core.peek().Kind == TokRParen {
		return nil, true
	}
	args := make([]AstExprNode, 0, 4)
	for {
		expr, ok := core.parseExpression()
		if !ok {
			return nil, false
		}
		args = append(args, expr)
		if !core.match(TokComma) {
			break
		}
	}
	return args, true
}

func (core *parserCore) parseLiteral() (AstExprNode, bool, bool) {
	token := core.peek()
	switch token.Kind {
	case TokInteger:
		core.pos++
		return &AstNumberLiteral{IntValue: int(token.IntValue), Line: token.Line}, true, true
	case TokFloat32, TokFloat64:
		core.pos++
		floatType := Float32Type
		if token.Kind == TokFloat64 {
			floatType = Float64Type
		}
		return &AstNumberLiteral{FloatValue: token.FloatValue, IsFloat: true, FloatType: floatType, Line: token.Line}, true, true
	case TokString:
		core.pos++
		return &AstStringLiteral{Value: token.Text, Line: token.Line}, true, true
	case TokTrue:
		core.pos++
		return &AstNumberLiteral{IntValue: 1, IsBool: true, Line: token.Line}, true, true
	case TokFalse:
		core.pos++
		return &AstNumberLiteral{IntValue: 0, IsBool: true, Line: token.Line}, true, true
	}
	return nil, false, true
}

func (core *parserCore) errorf(token Token, message string) bool {
	if core.ctx != nil {
		core.ctx.AddError("syntax error on line %d, column %d: %s", token.Line, token.Column, message)
	}
	return false
}

func expectedTokenLabel(kind TokenKind) string {
	if spelling := tokenSpellings[kind]; spelling != "" {
		return fmt.Sprintf("expected %q", spelling)
	}
	switch kind {
	case TokIdent:
		return "expected identifier"
	case TokInteger:
		return "expected integer"
	case TokString:
		return "expected string"
	default:
		return "expected token"
	}
}

var tokenTypes = map[TokenKind]*Type{
	TokVoid: VoidType, TokBool: BoolType, TokByte: ByteType, TokChar: CharType,
	TokInt: Int32Type, TokInt8: Int8Type, TokInt16: Int16Type, TokInt32: Int32Type, TokInt64: Int64Type,
	TokUint8: Uint8Type, TokUint16: Uint16Type, TokUint32: Uint32Type, TokUint64: Uint64Type,
	TokFloat: Float32Type, TokFloat32Type: Float32Type, TokFloat64Type: Float64Type,
	TokI8: Int8Type, TokI16: Int16Type, TokI32: Int32Type, TokI64: Int64Type,
	TokU8: Uint8Type, TokU16: Uint16Type, TokU32: Uint32Type, TokU64: Uint64Type,
	TokF32: Float32Type, TokF64: Float64Type, TokDouble: Float64Type,
}
