package cova

type expressionParser struct {
	core *parserCore
}

func newExpressionParser(core *parserCore) *expressionParser {
	return &expressionParser{core: core}
}

func (parser *expressionParser) parseExpression() (AstExprNode, bool) {
	return parser.parseExpressionWithBindingPower(0)
}

func (parser *expressionParser) parseExpressionWithBindingPower(minBindingPower int) (AstExprNode, bool) {
	left, ok := parser.parsePrefixExpression()
	if !ok {
		return nil, false
	}

	for {
		token := parser.core.peek()
		if token.Kind == TokColonColon {
			qualifier, ok := left.(*AstIdentNode)
			if !ok {
				return nil, parser.core.errorf(token, "qualified names are not supported")
			}
			parser.core.pos++
			name, ok := parser.core.expect(TokIdent)
			if !ok {
				return nil, false
			}
			if !parser.core.match(TokLParen) {
				return nil, parser.core.errorf(parser.core.peek(), "qualified names are only supported for built-in calls")
			}
			args, ok := parser.core.parseArguments()
			if !ok {
				return nil, false
			}
			if _, ok := parser.core.expect(TokRParen); !ok {
				return nil, false
			}
			left = &AstCallExpr{Callee: qualifier.Name + "::" + name.Text, Args: args, Line: qualifier.Line}
			continue
		}
		if message := unsupportedExpressionToken(token.Kind); message != "" {
			return nil, parser.core.errorf(token, message)
		}
		if token.Kind == TokLParen {
			const callBindingPower = 100
			if callBindingPower < minBindingPower {
				break
			}
			ident, ok := left.(*AstIdentNode)
			if !ok {
				return nil, parser.core.errorf(token, "expected expression")
			}
			parser.core.pos++
			args, ok := parser.core.parseArguments()
			if !ok {
				return nil, false
			}
			if _, ok := parser.core.expect(TokRParen); !ok {
				return nil, false
			}
			left = &AstCallExpr{Callee: ident.Name, Args: args, Line: ident.Line}
			continue
		}
		if token.Kind == TokDot {
			base, ok := left.(AstLvalueNode)
			if !ok {
				return nil, parser.core.errorf(token, "member base is not addressable")
			}
			parser.core.pos++
			member, ok := parser.core.expect(TokIdent)
			if !ok {
				return nil, false
			}
			left = &AstMemberExpr{Base: base, Member: member.Text, Line: token.Line}
			continue
		}
		if token.Kind == TokLBracket {
			base, ok := left.(AstLvalueNode)
			if !ok {
				return nil, parser.core.errorf(token, "index base is not addressable")
			}
			parser.core.pos++
			index, ok := parser.parseExpressionWithBindingPower(0)
			if !ok {
				return nil, false
			}
			if _, ok := parser.core.expect(TokRBracket); !ok {
				return nil, false
			}
			left = &AstIndexExpr{Base: base, Index: index, Line: token.Line}
			continue
		}

		leftBP, rightBP, ok := infixBindingPower(token)
		if !ok || leftBP < minBindingPower {
			break
		}
		parser.core.pos++
		right, ok := parser.parseExpressionWithBindingPower(rightBP)
		if !ok {
			return nil, false
		}
		left = &AstBinaryExpr{Op: binaryOps[token.Kind], Left: left, Right: right, Line: token.Line}
	}

	return left, true
}

func (parser *expressionParser) parsePrefixExpression() (AstExprNode, bool) {
	token := parser.core.peek()
	if message := unsupportedExpressionToken(token.Kind); message != "" {
		return nil, parser.core.errorf(token, message)
	}
	if op, ok := unaryOps[token.Kind]; ok {
		parser.core.pos++
		operand, ok := parser.parsePrefixExpression()
		if !ok {
			return nil, false
		}
		return &AstUnaryExpr{Op: op, Operand: operand, Line: token.Line}, true
	}
	if literal, matched, success := parser.core.parseLiteral(); matched || !success {
		return literal, success
	}

	switch token.Kind {
	case TokIdent:
		parser.core.pos++
		return &AstIdentNode{Name: token.Text, Line: token.Line}, true
	case TokLParen:
		parser.core.pos++
		expr, ok := parser.parseExpressionWithBindingPower(0)
		if !ok {
			return nil, false
		}
		if _, ok := parser.core.expect(TokRParen); !ok {
			return nil, false
		}
		return expr, true
	default:
		return nil, parser.core.errorf(token, "expected expression")
	}
}

func infixBindingPower(token Token) (int, int, bool) {
	switch token.Kind {
	case TokLogicalOr:
		return 10, 11, true
	case TokLogicalAnd:
		return 20, 21, true
	case TokPipe:
		return 25, 26, true
	case TokCaret:
		return 30, 31, true
	case TokAmp:
		return 35, 36, true
	case TokEqual, TokNotEqual:
		return 40, 41, true
	case TokLess, TokGreater, TokLessEqual, TokGreaterEqual:
		return 50, 51, true
	case TokShiftLeft, TokShiftRight:
		return 60, 61, true
	case TokPlus, TokMinus:
		return 70, 71, true
	case TokStar, TokSlash, TokPercent:
		return 80, 81, true
	default:
		return 0, 0, false
	}
}

var unaryOps = map[TokenKind]UnaryOp{
	TokBang: UnaryLogicalNot, TokMinus: UnaryNegate, TokTilde: UnaryBitwiseNot,
}

var binaryOps = map[TokenKind]BinaryOp{
	TokLogicalOr: BinaryLogicalOr, TokLogicalAnd: BinaryLogicalAnd,
	TokPipe: BinaryBitwiseOr, TokCaret: BinaryBitwiseXor, TokAmp: BinaryBitwiseAnd,
	TokEqual: BinaryEqual, TokNotEqual: BinaryNotEqual,
	TokLess: BinaryLess, TokLessEqual: BinaryLessEqual,
	TokGreater: BinaryGreater, TokGreaterEqual: BinaryGreaterEqual,
	TokShiftLeft: BinaryShiftLeft, TokShiftRight: BinaryShiftRight,
	TokPlus: BinaryAdd, TokMinus: BinarySub, TokStar: BinaryMul, TokSlash: BinaryDiv,
	TokPercent: BinaryModulo,
}

func unsupportedExpressionToken(kind TokenKind) string {
	switch kind {
	case TokArrow:
		return "member access is not supported"
	case TokColonColon:
		return "qualified names are not supported"
	case TokQuestion:
		return "ternary expressions are not supported"
	case TokIncrement, TokDecrement:
		return "increment and decrement are not supported"
	case TokEllipsis:
		return "variadic expressions are not supported"
	case TokHash, TokHashHash:
		return "preprocessor syntax is not supported"
	default:
		return ""
	}
}
