package cova

import "testing"

func issueDescriptions(ctx *Context) []string {
	issues := ctx.Issues()
	descriptions := make([]string, 0, len(issues))
	for _, issue := range issues {
		descriptions = append(descriptions, issue.Description)
	}
	return descriptions
}

func hasIssueContaining(ctx *Context, substring string) bool {
	for _, issue := range ctx.Issues() {
		if contains(issue.Description, substring) {
			return true
		}
	}
	return false
}

func contains(text string, substring string) bool {
	if len(substring) == 0 {
		return true
	}
	if len(substring) > len(text) {
		return false
	}
	for i := 0; i+len(substring) <= len(text); i++ {
		if text[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}

func mustTokenize(t *testing.T, ctx *Context, src string) []Token {
	t.Helper()
	tokens, ok := Tokenize(ctx, src)
	if !ok {
		t.Fatalf("Tokenize failed: %v", issueDescriptions(ctx))
	}
	return tokens
}

func mustParseTokens(t *testing.T, ctx *Context, tokens []Token) *AstProgramNode {
	t.Helper()
	program, ok := Parse(ctx, tokens)
	if !ok {
		t.Fatalf("Parse failed: %v", issueDescriptions(ctx))
	}
	return program
}

func mustParseScript(t *testing.T, script string) *AstProgramNode {
	t.Helper()
	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	return mustParseTokens(t, ctx, tokens)
}
