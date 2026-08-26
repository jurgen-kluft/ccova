package cova

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokenKind int

const (
	TokEOF TokenKind = iota
	TokIdent
	TokString
	TokInteger
	TokFloat32
	TokFloat64
	TokNewline

	TokBreak
	TokCase
	TokConst
	TokContinue
	TokDefault
	TokElse
	TokExtern
	TokFalse
	TokStruct
	TokBool
	TokByte
	TokChar
	TokFloat32Type
	TokFloat64Type
	TokFor
	TokIf
	TokInt8
	TokInt16
	TokInt32
	TokInt64
	TokReturn
	TokSwitch
	TokTrue
	TokUint8
	TokUint16
	TokUint32
	TokUint64
	TokVoid
	TokWhile
	TokInt
	TokFloat

	TokPlus
	TokMinus
	TokStar
	TokSlash
	TokPercent
	TokIncrement
	TokDecrement
	TokArrow
	TokEqual
	TokNotEqual
	TokLess
	TokLessEqual
	TokGreater
	TokGreaterEqual
	TokLogicalAnd
	TokLogicalOr
	TokShiftLeft
	TokShiftRight
	TokAssign
	TokPlusAssign
	TokMinusAssign
	TokStarAssign
	TokSlashAssign
	TokPercentAssign
	TokShiftLeftAssign
	TokShiftRightAssign
	TokAndAssign
	TokOrAssign
	TokXorAssign
	TokAmp
	TokPipe
	TokCaret
	TokTilde
	TokBang

	TokLParen
	TokRParen
	TokLBrace
	TokRBrace
	TokLBracket
	TokRBracket
	TokSemicolon
	TokComma
	TokColon
	TokColonColon
	TokDot
	TokEllipsis
	TokQuestion
	TokHash
	TokHashHash

	TokI8
	TokI16
	TokI32
	TokI64
	TokU8
	TokU16
	TokU32
	TokU64
	TokF32
	TokF64
	TokDouble
)

type Token struct {
	Kind       TokenKind
	Text       string
	IntValue   int64
	FloatValue float64
	Line       int
	Column     int
	Offset     int
	Length     int
}

var keywords = map[string]TokenKind{
	"break": TokBreak, "case": TokCase, "const": TokConst, "continue": TokContinue,
	"default": TokDefault, "else": TokElse, "extern": TokExtern, "false": TokFalse,
	"struct": TokStruct, "bool": TokBool, "byte": TokByte, "char": TokChar,
	"float32": TokFloat32Type, "float64": TokFloat64Type,
	"for": TokFor, "if": TokIf, "int8": TokInt8, "int16": TokInt16, "int32": TokInt32,
	"int64": TokInt64, "return": TokReturn, "switch": TokSwitch, "true": TokTrue,
	"uint8": TokUint8, "uint16": TokUint16, "uint32": TokUint32, "uint64": TokUint64,
	"void": TokVoid, "while": TokWhile, "int": TokInt, "float": TokFloat,
	"i8": TokI8, "i16": TokI16, "i32": TokI32, "i64": TokI64,
	"u8": TokU8, "u16": TokU16, "u32": TokU32, "u64": TokU64,
	"f32": TokF32, "f64": TokF64, "double": TokDouble,
}

func Tokenize(ctx *Context, src string) ([]Token, bool) {
	tokens := make([]Token, 0, len(src)/2)
	line := 1
	column := 1

	for index := 0; index < len(src); {
		char, charWidth := utf8.DecodeRuneInString(src[index:])
		startLine := line
		startColumn := column

		switch {
		case char == '\n' || char == '\r':
			width := charWidth
			if char == '\r' && index+charWidth < len(src) && src[index+charWidth] == '\n' {
				width++
			}
			tokens = append(tokens, Token{Kind: TokNewline, Line: line, Column: column, Offset: index, Length: width})
			line++
			column = 1
			index += width
		case unicode.IsSpace(char):
			index += charWidth
			column++
		case char == '/' && index+1 < len(src) && src[index+1] == '/':
			index += 2
			column += 2
			for index < len(src) && src[index] != '\n' && src[index] != '\r' {
				index++
				column++
			}
		case char == '/' && index+1 < len(src) && src[index+1] == '*':
			index += 2
			column += 2
			closed := false
			for index < len(src) {
				if index+1 < len(src) && src[index] == '*' && src[index+1] == '/' {
					index += 2
					column += 2
					closed = true
					break
				}
				if src[index] == '\n' || src[index] == '\r' {
					if src[index] == '\r' && index+1 < len(src) && src[index+1] == '\n' {
						index++
					}
					line++
					column = 1
					index++
					continue
				}
				index++
				column++
			}
			if !closed {
				return nil, lexFail(ctx, "lex error on line %d, column %d: unterminated block comment", startLine, startColumn)
			}
		case isIdentStart(char):
			start := index
			index += charWidth
			column++
			for index < len(src) {
				part, width := utf8.DecodeRuneInString(src[index:])
				if !isIdentPart(part) {
					break
				}
				index += width
				column++
			}
			value := src[start:index]
			kind := TokIdent
			if keywordKind, ok := keywords[value]; ok {
				kind = keywordKind
			}
			tokens = append(tokens, Token{Kind: kind, Text: value, Line: startLine, Column: startColumn, Offset: start, Length: index - start})
		case unicode.IsDigit(char):
			token, newIndex, ok := tokenizeNumericLiteral(ctx, src, index, line, column)
			if !ok {
				return nil, false
			}
			tokens = append(tokens, token)
			column += newIndex - index
			index = newIndex
		case char == '"':
			token, newIndex, ok := tokenizeStringLiteral(ctx, src, index, line, column)
			if !ok {
				return nil, false
			}
			tokens = append(tokens, token)
			column += newIndex - index
			index = newIndex
		case isOperator(char):
			value, width := tokenizeOperator(src, index)
			tokens = append(tokens, Token{Kind: operatorKinds[value], Line: line, Column: column, Offset: index, Length: width})
			index += width
			column += width
		case isDelimiter(char):
			value, width := tokenizeDelimiter(src, index)
			tokens = append(tokens, Token{Kind: delimiterKinds[value], Line: line, Column: column, Offset: index, Length: width})
			index += width
			column += width
		default:
			return nil, lexFail(ctx, "lex error on line %d, column %d: unrecognized symbol %q", line, column, string(char))
		}
	}

	tokens = append(tokens, Token{Kind: TokEOF, Line: line, Column: column, Offset: len(src)})
	return tokens, true
}

func tokenizeNumericLiteral(ctx *Context, src string, start int, line int, column int) (Token, int, bool) {
	index := start
	for index < len(src) && unicode.IsDigit(rune(src[index])) {
		index++
	}
	if index < len(src) && src[index] == '.' {
		index++
		if index >= len(src) || !unicode.IsDigit(rune(src[index])) {
			return Token{}, index, lexFail(ctx, "lex error on line %d, column %d: invalid numeric literal", line, column)
		}
		for index < len(src) && unicode.IsDigit(rune(src[index])) {
			index++
		}
	}
	if index < len(src) && (src[index] == 'e' || src[index] == 'E') {
		index++
		if index < len(src) && (src[index] == '+' || src[index] == '-') {
			index++
		}
		if index >= len(src) || !unicode.IsDigit(rune(src[index])) {
			return Token{}, index, lexFail(ctx, "lex error on line %d, column %d: invalid numeric literal", line, column)
		}
		for index < len(src) && unicode.IsDigit(rune(src[index])) {
			index++
		}
	}
	kind := TokInteger
	if strings.ContainsAny(src[start:index], ".eE") {
		kind = TokFloat32
	}
	if index < len(src) {
		switch src[index] {
		case 'f', 'F':
			kind = TokFloat32
			index++
		case 'd', 'D':
			kind = TokFloat64
			index++
		}
	}
	value := src[start:index]
	token := Token{Kind: kind, Line: line, Column: column, Offset: start, Length: index - start}
	if kind == TokInteger {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return Token{}, index, lexFail(ctx, "lex error on line %d, column %d: invalid integer literal %q", line, column, value)
		}
		token.IntValue = parsed
		return token, index, true
	}
	floatText := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(value, "f"), "F"), "d"), "D")
	parsed, err := strconv.ParseFloat(floatText, 64)
	if err != nil {
		return Token{}, index, lexFail(ctx, "lex error on line %d, column %d: invalid float literal %q", line, column, value)
	}
	token.FloatValue = parsed
	return token, index, true
}

func tokenizeStringLiteral(ctx *Context, src string, start int, line int, column int) (Token, int, bool) {
	index := start + 1
	value := make([]rune, 0, 16)
	for index < len(src) {
		char := rune(src[index])
		switch char {
		case '"':
			text := string(value)
			return Token{Kind: TokString, Text: text, Line: line, Column: column, Offset: start, Length: index + 1 - start}, index + 1, true
		case '\\':
			if index+1 >= len(src) {
				return Token{}, index, lexFail(ctx, "lex error on line %d: unterminated string literal", line)
			}
			escaped := rune(src[index+1])
			switch escaped {
			case '"':
				value = append(value, '"')
			case '\\':
				value = append(value, '\\')
			case 'n':
				value = append(value, '\n')
			case 'r':
				value = append(value, '\r')
			case 't':
				value = append(value, '\t')
			case '0':
				value = append(value, 0)
			default:
				return Token{}, index, lexFail(ctx, "lex error on line %d: unsupported string escape %q", line, "\\"+string(escaped))
			}
			index += 2
		case '\n', '\r':
			return Token{}, index, lexFail(ctx, "lex error on line %d: unterminated string literal", line)
		default:
			value = append(value, char)
			index++
		}
	}
	return Token{}, index, lexFail(ctx, "lex error on line %d: unterminated string literal", line)
}

func lexFail(ctx *Context, format string, args ...any) bool {
	if ctx != nil {
		ctx.AddError(format, args...)
	}
	return false
}

func isIdentStart(char rune) bool {
	return char == '_' || unicode.IsLetter(char)
}

func isIdentPart(char rune) bool {
	return isIdentStart(char) || unicode.IsDigit(char)
}

func isOperator(char rune) bool {
	switch char {
	case '+', '-', '*', '/', '%', '=', '!', '<', '>', '&', '|', '^', '~':
		return true
	default:
		return false
	}
}

var operatorKinds = map[string]TokenKind{
	"+": TokPlus, "-": TokMinus, "*": TokStar, "/": TokSlash, "%": TokPercent,
	"++": TokIncrement, "--": TokDecrement, "->": TokArrow,
	"==": TokEqual, "!=": TokNotEqual, "<": TokLess, "<=": TokLessEqual,
	">": TokGreater, ">=": TokGreaterEqual, "&&": TokLogicalAnd, "||": TokLogicalOr,
	"<<": TokShiftLeft, ">>": TokShiftRight, "=": TokAssign,
	"+=": TokPlusAssign, "-=": TokMinusAssign, "*=": TokStarAssign, "/=": TokSlashAssign,
	"%=": TokPercentAssign, "<<=": TokShiftLeftAssign, ">>=": TokShiftRightAssign,
	"&=": TokAndAssign, "|=": TokOrAssign, "^=": TokXorAssign,
	"&": TokAmp, "|": TokPipe, "^": TokCaret, "~": TokTilde, "!": TokBang,
}

func tokenizeOperator(src string, index int) (string, int) {
	if index+2 < len(src) {
		switch src[index : index+3] {
		case "<<=", ">>=":
			return src[index : index+3], 3
		}
	}
	if index+1 < len(src) {
		switch src[index : index+2] {
		case "++", "--", "->",
			"==", "!=", "<=", ">=", "&&", "||", "<<", ">>",
			"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=":
			return src[index : index+2], 2
		}
	}
	return src[index : index+1], 1
}

func isDelimiter(char rune) bool {
	switch char {
	case '(', ')', '{', '}', '[', ']', ';', ',', ':', '.', '?', '#':
		return true
	default:
		return false
	}
}

var delimiterKinds = map[string]TokenKind{
	"(": TokLParen, ")": TokRParen, "{": TokLBrace, "}": TokRBrace,
	"[": TokLBracket, "]": TokRBracket, ";": TokSemicolon, ",": TokComma,
	":": TokColon, "::": TokColonColon, ".": TokDot, "...": TokEllipsis,
	"?": TokQuestion, "#": TokHash, "##": TokHashHash,
}

var tokenSpellings = func() map[TokenKind]string {
	spellings := make(map[TokenKind]string, len(keywords)+len(operatorKinds)+len(delimiterKinds))
	for spelling, kind := range keywords {
		spellings[kind] = spelling
	}
	for spelling, kind := range operatorKinds {
		spellings[kind] = spelling
	}
	for spelling, kind := range delimiterKinds {
		spellings[kind] = spelling
	}
	return spellings
}()

func tokenizeDelimiter(src string, index int) (string, int) {
	if index+2 < len(src) && src[index:index+3] == "..." {
		return src[index : index+3], 3
	}
	if index+1 < len(src) {
		switch src[index : index+2] {
		case "::", "##":
			return src[index : index+2], 2
		}
	}
	return src[index : index+1], 1
}
