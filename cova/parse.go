package cova

func Parse(ctx *Context, tokens []Token) (*AstProgramNode, bool) {
	core := newParserCore(ctx, tokens)
	core.expr = newExpressionParser(&core)
	return core.parseProgram()
}
