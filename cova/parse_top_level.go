package cova

import "fmt"

func (core *parserCore) parseProgram() (*AstProgramNode, bool) {
	program := &AstProgramNode{}
	for !core.isEOF() {
		if core.peek().Kind == TokStruct {
			decl, ok := core.parseStructDecl()
			if !ok {
				return nil, false
			}
			program.Structs = append(program.Structs, decl)
			continue
		}
		if core.peek().Kind == TokExtern {
			decl, ok := core.parseExternDecl()
			if !ok {
				return nil, false
			}
			program.Decls = append(program.Decls, decl)
			continue
		}

		decl, function, ok := core.parseTopLevelDeclOrFunction()
		if !ok {
			return nil, false
		}
		if decl != nil {
			program.Decls = append(program.Decls, decl)
			continue
		}
		program.Functions = append(program.Functions, function)
	}

	return program, true
}

func (core *parserCore) parseStructDecl() (*AstStructDeclNode, bool) {
	line := core.peek().Line
	if _, ok := core.expect(TokStruct); !ok {
		return nil, false
	}
	nameToken, ok := core.expect(TokIdent)
	if !ok {
		return nil, false
	}
	if _, exists := core.namedTypes[nameToken.Text]; exists {
		return nil, core.errorf(nameToken, fmt.Sprintf("duplicate type name %q", nameToken.Text))
	}
	if _, ok := core.expect(TokLBrace); !ok {
		return nil, false
	}
	fields := make([]StructField, 0, 8)
	fieldNames := make(map[string]struct{}, 8)
	for core.peek().Kind != TokRBrace {
		if core.isEOF() {
			return nil, core.errorf(core.peek(), "expected closing brace")
		}
		fieldType, ok := core.parseType()
		if !ok {
			return nil, false
		}
		fieldName, ok := core.expect(TokIdent)
		if !ok {
			return nil, false
		}
		if _, exists := fieldNames[fieldName.Text]; exists {
			return nil, core.errorf(fieldName, fmt.Sprintf("struct %q has duplicate field %q", nameToken.Text, fieldName.Text))
		}
		fieldType, ok = core.parseArrayDeclarator(fieldType)
		if !ok {
			return nil, false
		}
		if fieldType == nil || fieldType.Kind == TypeVoid || fieldType.Size <= 0 {
			return nil, core.errorf(fieldName, fmt.Sprintf("struct %q field %q must have a complete type", nameToken.Text, fieldName.Text))
		}
		if _, ok := core.expect(TokSemicolon); !ok {
			return nil, false
		}
		fieldNames[fieldName.Text] = struct{}{}
		fields = append(fields, StructField{Name: fieldName.Text, Type: fieldType})
	}
	core.pos++
	if _, ok := core.expect(TokSemicolon); !ok {
		return nil, false
	}
	if len(fields) == 0 {
		return nil, core.errorf(nameToken, fmt.Sprintf("struct %q must declare at least one field", nameToken.Text))
	}
	typ, ok := NewStructType(core.ctx, nameToken.Text, fields)
	if !ok {
		return nil, false
	}
	core.namedTypes[nameToken.Text] = typ
	return &AstStructDeclNode{Name: nameToken.Text, Type: typ, Line: line}, true
}

func (core *parserCore) parseExternDecl() (*AstTopLevelDeclNode, bool) {
	line := core.peek().Line
	if _, ok := core.expect(TokExtern); !ok {
		return nil, false
	}
	index := -1
	if core.match(TokLParen) {
		indexToken, ok := core.expect(TokInteger)
		if !ok {
			return nil, false
		}
		if _, ok := core.expect(TokRParen); !ok {
			return nil, false
		}
		index = int(indexToken.IntValue)
	}
	if core.peek().Kind == TokConst {
		return nil, core.errorf(core.peek(), "extern declarations cannot be const")
	}

	typ, ok := core.parseType()
	if !ok {
		return nil, false
	}
	nameToken, ok := core.expect(TokIdent)
	if !ok {
		return nil, false
	}

	typ, ok = core.parseArrayDeclarator(typ)
	if !ok {
		return nil, false
	}
	decl := &AstTopLevelDeclNode{Index: index, Name: nameToken.Text, Type: typ, Scope: ScopeExtern, Line: line}
	if core.match(TokLParen) {
		if index < 0 {
			return nil, core.errorf(nameToken, "extern functions require an explicit import slot")
		}
		params, ok := core.parseParameters()
		if !ok {
			return nil, false
		}
		decl.Params = params
		decl.Kind = DeclFunction
		if _, ok := core.expect(TokRParen); !ok {
			return nil, false
		}
	} else {
		if index >= 0 {
			return nil, core.errorf(nameToken, "extern variable offsets are automatic; remove the parenthesized offset")
		}
		decl.Kind = DeclVariable
	}

	if _, ok := core.expect(TokSemicolon); !ok {
		return nil, false
	}
	return decl, true
}

func (core *parserCore) parseTopLevelDeclOrFunction() (*AstTopLevelDeclNode, *AstFunctionNode, bool) {
	line := core.peek().Line
	returnType, ok := core.parseType()
	if !ok {
		return nil, nil, false
	}
	nameToken, ok := core.expect(TokIdent)
	if !ok {
		return nil, nil, false
	}
	if core.match(TokLParen) {
		params, ok := core.parseParameters()
		if !ok {
			return nil, nil, false
		}
		if _, ok := core.expect(TokRParen); !ok {
			return nil, nil, false
		}
		body, ok := core.parseBlock()
		if !ok {
			return nil, nil, false
		}
		return nil, &AstFunctionNode{ReturnType: returnType, Name: nameToken.Text, Params: params, Body: body, Line: line}, true
	}
	returnType, ok = core.parseArrayDeclarator(returnType)
	if !ok {
		return nil, nil, false
	}
	if returnType.Kind == TypeVoid {
		return nil, nil, core.errorf(nameToken, "internal variable \""+nameToken.Text+"\" cannot have type void")
	}
	var initializer AstExprNode
	scope := ScopeBSS
	if core.match(TokAssign) {
		initializer, ok = core.parseExpression()
		if !ok {
			return nil, nil, false
		}
		scope = ScopeData
	}
	if IsTopLevelConst(returnType) {
		scope = ScopeConst
	}
	if _, ok := core.expect(TokSemicolon); !ok {
		return nil, nil, false
	}
	decl := &AstTopLevelDeclNode{
		Index:       -1,
		Name:        nameToken.Text,
		Type:        returnType,
		Kind:        DeclVariable,
		Scope:       scope,
		Initializer: initializer,
		Line:        line,
	}
	return decl, nil, true
}

func (core *parserCore) parseParameters() ([]AstParameter, bool) {
	if core.peek().Kind == TokRParen {
		return nil, true
	}

	params := make([]AstParameter, 0, 4)
	for {
		typ, ok := core.parseType()
		if !ok {
			return nil, false
		}
		nameToken, ok := core.expect(TokIdent)
		if !ok {
			return nil, false
		}
		typ, ok = core.parseArrayDeclarator(typ)
		if !ok {
			return nil, false
		}
		params = append(params, AstParameter{Type: typ, Name: nameToken.Text, Line: nameToken.Line})

		if !core.match(TokComma) {
			break
		}
	}
	return params, true
}
