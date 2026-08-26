package cova

func (core *parserCore) parseBlock() (*AstBlockStmt, bool) {
	line := core.peek().Line
	if _, ok := core.expect(TokLBrace); !ok {
		return nil, false
	}

	block := &AstBlockStmt{Line: line}
	for core.peek().Kind != TokRBrace {
		if core.isEOF() {
			return nil, core.errorf(core.peek(), "expected closing brace")
		}
		stmt, ok := core.parseStatement()
		if !ok {
			return nil, false
		}
		block.Statements = append(block.Statements, stmt)
	}

	if _, ok := core.expect(TokRBrace); !ok {
		return nil, false
	}
	return block, true
}

func (core *parserCore) parseStatement() (AstStmtNode, bool) {
	token := core.peek()
	if token.Kind == TokLBrace {
		return core.parseBlock()
	}
	if core.isTypeKeyword(token) {
		return core.parseLocalDeclStmt()
	}
	if token.Kind == TokIf {
		return core.parseIfStmt()
	}
	if token.Kind == TokWhile {
		return core.parseWhileStmt()
	}
	if token.Kind == TokFor {
		return core.parseForStmt()
	}
	if token.Kind == TokSwitch {
		return core.parseSwitchStmt()
	}
	if token.Kind == TokReturn {
		return core.parseReturnStmt()
	}
	if token.Kind == TokBreak {
		core.pos++
		if _, ok := core.expect(TokSemicolon); !ok {
			return nil, false
		}
		return &AstBreakStmt{Line: token.Line}, true
	}
	if token.Kind == TokContinue {
		core.pos++
		if _, ok := core.expect(TokSemicolon); !ok {
			return nil, false
		}
		return &AstContinueStmt{Line: token.Line}, true
	}

	line := token.Line
	expr, ok := core.parseExpression()
	if !ok {
		return nil, false
	}
	if assignmentOp, ok := core.matchAssignmentOperator(); ok {
		target, ok := expr.(AstLvalueNode)
		if !ok {
			return nil, core.errorf(token, "assignment target is not assignable")
		}
		value, ok := core.parseExpression()
		if !ok {
			return nil, false
		}
		if _, ok := core.expect(TokSemicolon); !ok {
			return nil, false
		}
		return &AstAssignStmt{Target: target, Op: assignmentOp, Value: value, Line: line}, true
	}
	if _, ok := core.expect(TokSemicolon); !ok {
		return nil, false
	}
	return &AstExprStmt{Expr: expr, Line: line}, true
}

func (core *parserCore) parseLocalDeclStmt() (AstStmtNode, bool) {
	line := core.peek().Line
	typ, ok := core.parseType()
	if !ok {
		return nil, false
	}
	nameToken, ok := core.expect(TokIdent)
	if !ok {
		return nil, false
	}
	if typ.Kind == TypeVoid {
		return nil, core.errorf(nameToken, "local variable \""+nameToken.Text+"\" cannot have type void")
	}
	typ, ok = core.parseArrayDeclarator(typ)
	if !ok {
		return nil, false
	}
	var initializer AstExprNode
	if core.match(TokAssign) {
		initializer, ok = core.parseExpression()
		if !ok {
			return nil, false
		}
	}
	if _, ok := core.expect(TokSemicolon); !ok {
		return nil, false
	}
	return &AstLocalDeclStmt{Type: typ, Name: nameToken.Text, Initializer: initializer, Line: line}, true
}

func (core *parserCore) parseIfStmt() (AstStmtNode, bool) {
	line := core.peek().Line
	if _, ok := core.expect(TokIf); !ok {
		return nil, false
	}
	if _, ok := core.expect(TokLParen); !ok {
		return nil, false
	}
	condition, ok := core.parseExpression()
	if !ok {
		return nil, false
	}
	if _, ok := core.expect(TokRParen); !ok {
		return nil, false
	}
	thenStmt, ok := core.parseStatement()
	if !ok {
		return nil, false
	}
	var elseStmt AstStmtNode
	if core.peek().Kind == TokElse {
		core.pos++
		elseStmt, ok = core.parseStatement()
		if !ok {
			return nil, false
		}
	}
	return &AstIfStmt{Condition: condition, Then: thenStmt, Else: elseStmt, Line: line}, true
}

func (core *parserCore) parseWhileStmt() (AstStmtNode, bool) {
	line := core.peek().Line
	if _, ok := core.expect(TokWhile); !ok {
		return nil, false
	}
	if _, ok := core.expect(TokLParen); !ok {
		return nil, false
	}
	condition, ok := core.parseExpression()
	if !ok {
		return nil, false
	}
	if _, ok := core.expect(TokRParen); !ok {
		return nil, false
	}
	body, ok := core.parseStatement()
	if !ok {
		return nil, false
	}
	return &AstWhileStmt{Condition: condition, Body: body, Line: line}, true
}

func (core *parserCore) parseForStmt() (AstStmtNode, bool) {
	line := core.peek().Line
	if _, ok := core.expect(TokFor); !ok {
		return nil, false
	}
	if _, ok := core.expect(TokLParen); !ok {
		return nil, false
	}
	var init AstStmtNode
	if core.peek().Kind != TokSemicolon {
		stmt, ok := core.parseForClauseStatement()
		if !ok {
			return nil, false
		}
		init = stmt
	}
	if _, ok := core.expect(TokSemicolon); !ok {
		return nil, false
	}
	var condition AstExprNode
	if core.peek().Kind != TokSemicolon {
		expr, ok := core.parseExpression()
		if !ok {
			return nil, false
		}
		condition = expr
	}
	if _, ok := core.expect(TokSemicolon); !ok {
		return nil, false
	}
	var post AstStmtNode
	if core.peek().Kind != TokRParen {
		stmt, ok := core.parseForClauseStatement()
		if !ok {
			return nil, false
		}
		post = stmt
	}
	if _, ok := core.expect(TokRParen); !ok {
		return nil, false
	}
	body, ok := core.parseStatement()
	if !ok {
		return nil, false
	}
	return &AstForStmt{Init: init, Condition: condition, Post: post, Body: body, Line: line}, true
}

func (core *parserCore) parseSwitchStmt() (AstStmtNode, bool) {
	line := core.peek().Line
	if _, ok := core.expect(TokSwitch); !ok {
		return nil, false
	}
	if _, ok := core.expect(TokLParen); !ok {
		return nil, false
	}
	value, ok := core.parseExpression()
	if !ok {
		return nil, false
	}
	if _, ok := core.expect(TokRParen); !ok {
		return nil, false
	}
	if _, ok := core.expect(TokLBrace); !ok {
		return nil, false
	}
	stmt := &AstSwitchStmt{Value: value, Line: line}
	for core.peek().Kind != TokRBrace {
		if core.isEOF() {
			return nil, core.errorf(core.peek(), "expected closing brace")
		}
		keyword := core.peek()
		if keyword.Kind == TokCase {
			core.pos++
			caseValue, ok := core.parseExpression()
			if !ok {
				return nil, false
			}
			if _, ok := core.expect(TokColon); !ok {
				return nil, false
			}
			caseBody, ok := core.parseSwitchClauseBody()
			if !ok {
				return nil, false
			}
			stmt.Cases = append(stmt.Cases, AstSwitchCase{Value: caseValue, Body: caseBody, Line: keyword.Line})
			continue
		}
		if keyword.Kind == TokDefault {
			core.pos++
			if _, ok := core.expect(TokColon); !ok {
				return nil, false
			}
			defaultBody, ok := core.parseSwitchClauseBody()
			if !ok {
				return nil, false
			}
			stmt.Default = defaultBody
			continue
		}
		return nil, core.errorf(keyword, "expected case or default")
	}
	if _, ok := core.expect(TokRBrace); !ok {
		return nil, false
	}
	return stmt, true
}

func (core *parserCore) parseSwitchClauseBody() ([]AstStmtNode, bool) {
	body := make([]AstStmtNode, 0, 4)
	for {
		token := core.peek()
		if token.Kind == TokRBrace {
			break
		}
		if token.Kind == TokCase || token.Kind == TokDefault {
			break
		}
		stmt, ok := core.parseStatement()
		if !ok {
			return nil, false
		}
		body = append(body, stmt)
	}
	return body, true
}

func (core *parserCore) parseForClauseStatement() (AstStmtNode, bool) {
	token := core.peek()
	line := token.Line
	expr, ok := core.parseExpression()
	if !ok {
		return nil, false
	}
	if assignmentOp, ok := core.matchAssignmentOperator(); ok {
		target, ok := expr.(AstLvalueNode)
		if !ok {
			return nil, core.errorf(token, "assignment target is not assignable")
		}
		value, ok := core.parseExpression()
		if !ok {
			return nil, false
		}
		return &AstAssignStmt{Target: target, Op: assignmentOp, Value: value, Line: line}, true
	}
	return &AstExprStmt{Expr: expr, Line: line}, true
}

func (core *parserCore) matchAssignmentOperator() (AssignOp, bool) {
	op, ok := assignmentOps[core.peek().Kind]
	if !ok {
		return "", false
	}
	core.pos++
	return op, true
}

var assignmentOps = map[TokenKind]AssignOp{
	TokAssign: AssignSimple, TokPlusAssign: AssignAdd, TokMinusAssign: AssignSub,
	TokStarAssign: AssignMul, TokSlashAssign: AssignDiv, TokPercentAssign: AssignModulo,
	TokShiftLeftAssign: AssignShiftLeft, TokShiftRightAssign: AssignShiftRight,
	TokAndAssign: AssignBitwiseAnd, TokXorAssign: AssignBitwiseXor, TokOrAssign: AssignBitwiseOr,
}

func (core *parserCore) parseReturnStmt() (AstStmtNode, bool) {
	line := core.peek().Line
	if _, ok := core.expect(TokReturn); !ok {
		return nil, false
	}
	if core.match(TokSemicolon) {
		return &AstReturnStmt{Line: line}, true
	}
	value, ok := core.parseExpression()
	if !ok {
		return nil, false
	}
	if _, ok := core.expect(TokSemicolon); !ok {
		return nil, false
	}
	return &AstReturnStmt{Value: value, Line: line}, true
}
