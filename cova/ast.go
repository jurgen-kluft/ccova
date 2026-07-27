package cova

type AstProgramNode struct {
	Structs   []*AstStructDeclNode
	Decls     []*AstTopLevelDeclNode
	Functions []*AstFunctionNode
}

type AstStructDeclNode struct {
	Name string
	Type *Type
	Line int
}

type AstParameter struct {
	Type *Type
	Name string
	Line int
}

type AstTopLevelDeclNode struct {
	Index       int
	Name        string
	Type        *Type
	Params      []AstParameter
	Initializer AstExprNode
	Kind        DeclKind
	Scope       ScopeKind
	Line        int
}

type AstFunctionNode struct {
	ReturnType *Type
	Name       string
	Params     []AstParameter
	Body       *AstBlockStmt
	Line       int
}

type AstStmtNode interface {
	astStmtNode()
}

type AstExprNode interface {
	astExprNode()
}

type AstLvalueNode interface {
	AstExprNode
	astEmitAddress(compiler *functionCompiler)
}

type UnaryOp string

const (
	UnaryLogicalNot UnaryOp = "!"
	UnaryNegate     UnaryOp = "-"
	UnaryBitwiseNot UnaryOp = "~"
)

type BinaryOp string

const (
	BinaryLogicalOr    BinaryOp = "||"
	BinaryLogicalAnd   BinaryOp = "&&"
	BinaryBitwiseOr    BinaryOp = "|"
	BinaryBitwiseXor   BinaryOp = "^"
	BinaryBitwiseAnd   BinaryOp = "&"
	BinaryEqual        BinaryOp = "=="
	BinaryNotEqual     BinaryOp = "!="
	BinaryLess         BinaryOp = "<"
	BinaryLessEqual    BinaryOp = "<="
	BinaryGreater      BinaryOp = ">"
	BinaryGreaterEqual BinaryOp = ">="
	BinaryShiftLeft    BinaryOp = "<<"
	BinaryShiftRight   BinaryOp = ">>"
	BinaryAdd          BinaryOp = "+"
	BinarySub          BinaryOp = "-"
	BinaryMul          BinaryOp = "*"
	BinaryDiv          BinaryOp = "/"
	BinaryModulo       BinaryOp = "%"
)

type AssignOp string

const (
	AssignSimple     AssignOp = "="
	AssignAdd        AssignOp = "+="
	AssignSub        AssignOp = "-="
	AssignMul        AssignOp = "*="
	AssignDiv        AssignOp = "/="
	AssignModulo     AssignOp = "%="
	AssignShiftLeft  AssignOp = "<<="
	AssignShiftRight AssignOp = ">>="
	AssignBitwiseAnd AssignOp = "&="
	AssignBitwiseXor AssignOp = "^="
	AssignBitwiseOr  AssignOp = "|="
)

type AstBlockStmt struct {
	Statements []AstStmtNode
	Line       int
}

type AstIfStmt struct {
	Condition AstExprNode
	Then      AstStmtNode
	Else      AstStmtNode
	Line      int
}

type AstWhileStmt struct {
	Condition AstExprNode
	Body      AstStmtNode
	Line      int
}

type AstForStmt struct {
	Init      AstStmtNode
	Condition AstExprNode
	Post      AstStmtNode
	Body      AstStmtNode
	Line      int
}

type AstSwitchCase struct {
	Value AstExprNode
	Body  []AstStmtNode
	Line  int
}

type AstSwitchStmt struct {
	Value   AstExprNode
	Cases   []AstSwitchCase
	Default []AstStmtNode
	Line    int
}

type AstBreakStmt struct {
	Line int
}

type AstContinueStmt struct {
	Line int
}

type AstReturnStmt struct {
	Value AstExprNode
	Line  int
}

type AstExprStmt struct {
	Expr AstExprNode
	Line int
}

type AstAssignStmt struct {
	Target AstLvalueNode
	Op     AssignOp
	Value  AstExprNode
	Line   int
}

type AstLocalDeclStmt struct {
	Type        *Type
	Name        string
	Initializer AstExprNode
	Line        int
}

type AstNumberLiteral struct {
	IntValue   int
	FloatValue float64
	IsFloat    bool
	IsBool     bool
	FloatType  *Type
	Line       int
}

type AstStringLiteral struct {
	Value string
	Line  int
}

type AstIdentNode struct {
	Name string
	Line int
}

type AstMemberExpr struct {
	Base   AstLvalueNode
	Member string
	Line   int
}

type AstIndexExpr struct {
	Base  AstLvalueNode
	Index AstExprNode
	Line  int
}

type AstBinaryExpr struct {
	Op    BinaryOp
	Left  AstExprNode
	Right AstExprNode
	Line  int
}

type AstUnaryExpr struct {
	Op      UnaryOp
	Operand AstExprNode
	Line    int
}

type AstCallExpr struct {
	Callee string
	Args   []AstExprNode
	Line   int
}

func (*AstBlockStmt) astStmtNode()     {}
func (*AstIfStmt) astStmtNode()        {}
func (*AstWhileStmt) astStmtNode()     {}
func (*AstForStmt) astStmtNode()       {}
func (*AstSwitchStmt) astStmtNode()    {}
func (*AstReturnStmt) astStmtNode()    {}
func (*AstExprStmt) astStmtNode()      {}
func (*AstAssignStmt) astStmtNode()    {}
func (*AstLocalDeclStmt) astStmtNode() {}
func (*AstBreakStmt) astStmtNode()     {}
func (*AstContinueStmt) astStmtNode()  {}
func (*AstNumberLiteral) astExprNode() {}
func (*AstStringLiteral) astExprNode() {}
func (*AstIdentNode) astExprNode()     {}
func (*AstMemberExpr) astExprNode()    {}
func (*AstIndexExpr) astExprNode()     {}
func (*AstBinaryExpr) astExprNode()    {}
func (*AstUnaryExpr) astExprNode()     {}
func (*AstCallExpr) astExprNode()      {}
