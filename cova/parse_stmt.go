package cova

import "fmt"

func (core *parserCore) parseBlock() (*AstBlockStmt, error) {
	line := core.peek().Line
	if _, err := core.expect(TokLBrace); err != nil {
		return nil, err
	}

	block := &AstBlockStmt{Line: line}
	for core.peek().Kind != TokRBrace {
		if core.isEOF() {
			return nil, core.errorf(core.peek(), "expected closing brace")
		}
		stmt, err := core.parseStatement()
		if err != nil {
			return nil, err
		}
		block.Statements = append(block.Statements, stmt)
	}

	if _, err := core.expect(TokRBrace); err != nil {
		return nil, err
	}
	return block, nil
}

func (core *parserCore) parseStatement() (AstStmtNode, error) {
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
		if _, err := core.expect(TokSemicolon); err != nil {
			return nil, err
		}
		return &AstBreakStmt{Line: token.Line}, nil
	}
	if token.Kind == TokContinue {
		core.pos++
		if _, err := core.expect(TokSemicolon); err != nil {
			return nil, err
		}
		return &AstContinueStmt{Line: token.Line}, nil
	}

	line := token.Line
	expr, err := core.parseExpression()
	if err != nil {
		return nil, err
	}
	if assignmentOp, ok := core.matchAssignmentOperator(); ok {
		target, ok := expr.(AstLvalueNode)
		if !ok {
			return nil, core.errorf(token, "assignment target is not assignable")
		}
		value, err := core.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := core.expect(TokSemicolon); err != nil {
			return nil, err
		}
		return &AstAssignStmt{Target: target, Op: assignmentOp, Value: value, Line: line}, nil
	}
	if _, err := core.expect(TokSemicolon); err != nil {
		return nil, err
	}
	return &AstExprStmt{Expr: expr, Line: line}, nil
}

func (core *parserCore) parseLocalDeclStmt() (AstStmtNode, error) {
	line := core.peek().Line
	typ, err := core.parseType()
	if err != nil {
		return nil, err
	}
	nameToken, err := core.expect(TokIdent)
	if err != nil {
		return nil, err
	}
	if typ.Kind == TypeVoid {
		return nil, fmt.Errorf("syntax error on line %d: local variable %q cannot have type void", line, nameToken.Text)
	}
	typ, err = core.parseArrayDeclarator(typ)
	if err != nil {
		return nil, err
	}
	var initializer AstExprNode
	if core.match(TokAssign) {
		initializer, err = core.parseExpression()
		if err != nil {
			return nil, err
		}
	}
	if _, err := core.expect(TokSemicolon); err != nil {
		return nil, err
	}
	return &AstLocalDeclStmt{Type: typ, Name: nameToken.Text, Initializer: initializer, Line: line}, nil
}

func (core *parserCore) parseIfStmt() (AstStmtNode, error) {
	line := core.peek().Line
	if _, err := core.expect(TokIf); err != nil {
		return nil, err
	}
	if _, err := core.expect(TokLParen); err != nil {
		return nil, err
	}
	condition, err := core.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := core.expect(TokRParen); err != nil {
		return nil, err
	}
	thenStmt, err := core.parseStatement()
	if err != nil {
		return nil, err
	}
	var elseStmt AstStmtNode
	if core.peek().Kind == TokElse {
		core.pos++
		elseStmt, err = core.parseStatement()
		if err != nil {
			return nil, err
		}
	}
	return &AstIfStmt{Condition: condition, Then: thenStmt, Else: elseStmt, Line: line}, nil
}

func (core *parserCore) parseWhileStmt() (AstStmtNode, error) {
	line := core.peek().Line
	if _, err := core.expect(TokWhile); err != nil {
		return nil, err
	}
	if _, err := core.expect(TokLParen); err != nil {
		return nil, err
	}
	condition, err := core.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := core.expect(TokRParen); err != nil {
		return nil, err
	}
	body, err := core.parseStatement()
	if err != nil {
		return nil, err
	}
	return &AstWhileStmt{Condition: condition, Body: body, Line: line}, nil
}

func (core *parserCore) parseForStmt() (AstStmtNode, error) {
	line := core.peek().Line
	if _, err := core.expect(TokFor); err != nil {
		return nil, err
	}
	if _, err := core.expect(TokLParen); err != nil {
		return nil, err
	}
	var init AstStmtNode
	if core.peek().Kind != TokSemicolon {
		stmt, err := core.parseForClauseStatement()
		if err != nil {
			return nil, err
		}
		init = stmt
	}
	if _, err := core.expect(TokSemicolon); err != nil {
		return nil, err
	}
	var condition AstExprNode
	if core.peek().Kind != TokSemicolon {
		expr, err := core.parseExpression()
		if err != nil {
			return nil, err
		}
		condition = expr
	}
	if _, err := core.expect(TokSemicolon); err != nil {
		return nil, err
	}
	var post AstStmtNode
	if core.peek().Kind != TokRParen {
		stmt, err := core.parseForClauseStatement()
		if err != nil {
			return nil, err
		}
		post = stmt
	}
	if _, err := core.expect(TokRParen); err != nil {
		return nil, err
	}
	body, err := core.parseStatement()
	if err != nil {
		return nil, err
	}
	return &AstForStmt{Init: init, Condition: condition, Post: post, Body: body, Line: line}, nil
}

func (core *parserCore) parseSwitchStmt() (AstStmtNode, error) {
	line := core.peek().Line
	if _, err := core.expect(TokSwitch); err != nil {
		return nil, err
	}
	if _, err := core.expect(TokLParen); err != nil {
		return nil, err
	}
	value, err := core.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := core.expect(TokRParen); err != nil {
		return nil, err
	}
	if _, err := core.expect(TokLBrace); err != nil {
		return nil, err
	}
	stmt := &AstSwitchStmt{Value: value, Line: line}
	for core.peek().Kind != TokRBrace {
		if core.isEOF() {
			return nil, core.errorf(core.peek(), "expected closing brace")
		}
		keyword := core.peek()
		if keyword.Kind == TokCase {
			core.pos++
			caseValue, err := core.parseExpression()
			if err != nil {
				return nil, err
			}
			if _, err := core.expect(TokColon); err != nil {
				return nil, err
			}
			caseBody, err := core.parseSwitchClauseBody()
			if err != nil {
				return nil, err
			}
			stmt.Cases = append(stmt.Cases, AstSwitchCase{Value: caseValue, Body: caseBody, Line: keyword.Line})
			continue
		}
		if keyword.Kind == TokDefault {
			core.pos++
			if _, err := core.expect(TokColon); err != nil {
				return nil, err
			}
			defaultBody, err := core.parseSwitchClauseBody()
			if err != nil {
				return nil, err
			}
			stmt.Default = defaultBody
			continue
		}
		return nil, core.errorf(keyword, "expected case or default")
	}
	if _, err := core.expect(TokRBrace); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (core *parserCore) parseSwitchClauseBody() ([]AstStmtNode, error) {
	body := make([]AstStmtNode, 0, 4)
	for {
		token := core.peek()
		if token.Kind == TokRBrace {
			break
		}
		if token.Kind == TokCase || token.Kind == TokDefault {
			break
		}
		stmt, err := core.parseStatement()
		if err != nil {
			return nil, err
		}
		body = append(body, stmt)
	}
	return body, nil
}

func (core *parserCore) parseForClauseStatement() (AstStmtNode, error) {
	token := core.peek()
	line := token.Line
	expr, err := core.parseExpression()
	if err != nil {
		return nil, err
	}
	if assignmentOp, ok := core.matchAssignmentOperator(); ok {
		target, ok := expr.(AstLvalueNode)
		if !ok {
			return nil, core.errorf(token, "assignment target is not assignable")
		}
		value, err := core.parseExpression()
		if err != nil {
			return nil, err
		}
		return &AstAssignStmt{Target: target, Op: assignmentOp, Value: value, Line: line}, nil
	}
	return &AstExprStmt{Expr: expr, Line: line}, nil
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

func (core *parserCore) parseReturnStmt() (AstStmtNode, error) {
	line := core.peek().Line
	if _, err := core.expect(TokReturn); err != nil {
		return nil, err
	}
	if core.match(TokSemicolon) {
		return &AstReturnStmt{Line: line}, nil
	}
	value, err := core.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := core.expect(TokSemicolon); err != nil {
		return nil, err
	}
	return &AstReturnStmt{Value: value, Line: line}, nil
}
