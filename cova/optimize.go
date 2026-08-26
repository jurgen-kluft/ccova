package cova

import (
	"math"
)

// Optimize applies isolated, in-place AST optimizations to program.
func Optimize(ctx *Context, program *AstProgramNode) bool {
	if program == nil {
		ctx.AddError("optimization error: program is nil")
		return false
	}
	optimizer := newOptimizer(ctx, program)
	return optimizer.optimizeProgram(program)
}

type optimizer struct {
	ctx             *Context
	globals         map[string]*Type
	functions       map[string][]*Type
	functionReturns map[string]*Type
	locals          []map[string]*Type
	returnType      *Type
}

func newOptimizer(ctx *Context, program *AstProgramNode) *optimizer {
	result := &optimizer{
		ctx:             ctx,
		globals:         make(map[string]*Type, len(program.Decls)),
		functions:       make(map[string][]*Type, len(program.Decls)+len(program.Functions)),
		functionReturns: make(map[string]*Type, len(program.Decls)+len(program.Functions)),
	}
	for _, decl := range program.Decls {
		if decl == nil {
			continue
		}
		if decl.Kind == DeclFunction {
			result.functions[decl.Name] = optimizerParameterTypes(decl.Params)
			result.functionReturns[decl.Name] = decl.Type
		} else {
			result.globals[decl.Name] = decl.Type
		}
	}
	for _, function := range program.Functions {
		if function != nil {
			result.functions[function.Name] = optimizerParameterTypes(function.Params)
			result.functionReturns[function.Name] = function.ReturnType
		}
	}
	return result
}

func optimizerParameterTypes(params []AstParameter) []*Type {
	types := make([]*Type, len(params))
	for index, param := range params {
		types[index] = param.Type
	}
	return types
}

func (optimizer *optimizer) optimizeProgram(program *AstProgramNode) bool {
	for _, decl := range program.Decls {
		if decl == nil || decl.Initializer == nil {
			continue
		}
		optimized, ok := optimizer.optimizeExpr(decl.Initializer, decl.Type)
		if !ok {
			return false
		}
		decl.Initializer = optimized
	}
	for _, function := range program.Functions {
		if function == nil {
			continue
		}
		optimizer.returnType = function.ReturnType
		optimizer.locals = []map[string]*Type{make(map[string]*Type, len(function.Params))}
		for _, param := range function.Params {
			optimizer.locals[0][param.Name] = param.Type
		}
		if !optimizer.optimizeBlock(function.Body) {
			return false
		}
	}
	return true
}

func (optimizer *optimizer) optimizeBlock(block *AstBlockStmt) bool {
	if block == nil {
		return true
	}
	optimizer.locals = append(optimizer.locals, make(map[string]*Type))
	defer func() { optimizer.locals = optimizer.locals[:len(optimizer.locals)-1] }()
	for _, statement := range block.Statements {
		if !optimizer.optimizeStmt(statement) {
			return false
		}
	}
	return true
}

func (optimizer *optimizer) optimizeStmt(statement AstStmtNode) bool {
	ok := true
	switch node := statement.(type) {
	case *AstBlockStmt:
		return optimizer.optimizeBlock(node)
	case *AstLocalDeclStmt:
		if node.Initializer != nil {
			node.Initializer, ok = optimizer.optimizeExpr(node.Initializer, node.Type)
		}
		optimizer.locals[len(optimizer.locals)-1][node.Name] = node.Type
	case *AstIfStmt:
		node.Condition, ok = optimizer.optimizeExpr(node.Condition, Int32Type)
		if ok {
			ok = optimizer.optimizeStmt(node.Then)
		}
		if ok && node.Else != nil {
			ok = optimizer.optimizeStmt(node.Else)
		}
	case *AstWhileStmt:
		node.Condition, ok = optimizer.optimizeExpr(node.Condition, Int32Type)
		if ok {
			ok = optimizer.optimizeStmt(node.Body)
		}
	case *AstForStmt:
		if node.Init != nil {
			ok = optimizer.optimizeStmt(node.Init)
		}
		if ok && node.Condition != nil {
			node.Condition, ok = optimizer.optimizeExpr(node.Condition, Int32Type)
		}
		if ok && node.Post != nil {
			ok = optimizer.optimizeStmt(node.Post)
		}
		if ok {
			ok = optimizer.optimizeStmt(node.Body)
		}
	case *AstSwitchStmt:
		node.Value, ok = optimizer.optimizeExpr(node.Value, optimizer.exprType(node.Value))
		for index := range node.Cases {
			if !ok {
				break
			}
			switchCase := &node.Cases[index]
			switchCase.Value, ok = optimizer.optimizeExpr(switchCase.Value, optimizer.exprType(switchCase.Value))
			for _, child := range switchCase.Body {
				if ok {
					ok = optimizer.optimizeStmt(child)
				}
			}
		}
		for _, child := range node.Default {
			if ok {
				ok = optimizer.optimizeStmt(child)
			}
		}
	case *AstReturnStmt:
		if node.Value != nil {
			node.Value, ok = optimizer.optimizeExpr(node.Value, optimizer.returnType)
		}
	case *AstExprStmt:
		node.Expr, ok = optimizer.optimizeExpr(node.Expr, optimizer.exprType(node.Expr))
	case *AstAssignStmt:
		node.Value, ok = optimizer.optimizeExpr(node.Value, optimizer.exprType(node.Target))
	}
	return ok
}

func (optimizer *optimizer) optimizeExpr(expression AstExprNode, expected *Type) (AstExprNode, bool) {
	switch node := expression.(type) {
	case *AstUnaryExpr:
		operand, ok := optimizer.optimizeExpr(node.Operand, optimizer.exprType(node.Operand))
		if !ok {
			return nil, false
		}
		node.Operand = operand
		return node, true
	case *AstBinaryExpr:
		return optimizer.optimizeBinary(node, expected)
	case *AstCallExpr:
		params := optimizer.functions[node.Callee]
		for index, argument := range node.Args {
			var argumentType *Type
			if index < len(params) {
				argumentType = params[index]
			}
			optimized, ok := optimizer.optimizeExpr(argument, argumentType)
			if !ok {
				return nil, false
			}
			node.Args[index] = optimized
		}
	case *AstIndexExpr:
		optimized, ok := optimizer.optimizeExpr(node.Index, Int32Type)
		if !ok {
			return nil, false
		}
		node.Index = optimized
	}
	return expression, true
}

func (optimizer *optimizer) optimizeBinary(node *AstBinaryExpr, expected *Type) (AstExprNode, bool) {
	if node.Op == "&&" || node.Op == "||" {
		return optimizer.optimizeLogical(node)
	}
	operationType := expected
	if optimizerIsComparison(node.Op) {
		operationType = optimizerPromoteNumericType(optimizer.exprType(node.Left), optimizer.exprType(node.Right))
	} else if operationType == nil {
		operationType = optimizer.exprType(node)
	}
	if operationType == nil {
		operationType = Int32Type
	}
	left, ok := optimizer.optimizeExpr(node.Left, operationType)
	if !ok {
		return nil, false
	}
	right, ok := optimizer.optimizeExpr(node.Right, operationType)
	if !ok {
		return nil, false
	}
	node.Left, node.Right = left, right
	leftLiteral, leftOK := left.(*AstNumberLiteral)
	rightLiteral, rightOK := right.(*AstNumberLiteral)
	if !leftOK || !rightOK {
		return node, true
	}
	constantKind := optimizerKindFromType(operationType)
	leftConstant := optimizerLiteralBits(leftLiteral, constantKind)
	rightConstant := optimizerLiteralBits(rightLiteral, constantKind)
	if optimizerIsComparison(node.Op) {
		value := optimizerCompare(constantKind, node.Op, leftConstant, rightConstant)
		return &AstNumberLiteral{IntValue: value, IsBool: true, Line: node.Line}, true
	}
	bits, failure, ok := optimizerArithmetic(constantKind, node.Op, leftConstant, rightConstant)
	if !ok {
		optimizer.ctx.AddError("optimization error on line %d: %s", node.Line, failure)
		return nil, false
	}
	return optimizerLiteralFromBits(bits, constantKind, node.Line), true
}

func (optimizer *optimizer) optimizeLogical(node *AstBinaryExpr) (AstExprNode, bool) {
	left, ok := optimizer.optimizeExpr(node.Left, Int32Type)
	if !ok {
		return nil, false
	}
	node.Left = left
	if literal, ok := left.(*AstNumberLiteral); ok {
		truth := optimizerLiteralBits(literal, optimizerInt32) != 0
		if (node.Op == "&&" && !truth) || (node.Op == "||" && truth) {
			return &AstNumberLiteral{IntValue: optimizerBoolInt(truth), IsBool: true, Line: node.Line}, true
		}
	}
	right, ok := optimizer.optimizeExpr(node.Right, Int32Type)
	if !ok {
		return nil, false
	}
	node.Right = right
	leftLiteral, leftOK := left.(*AstNumberLiteral)
	rightLiteral, rightOK := right.(*AstNumberLiteral)
	if !leftOK || !rightOK {
		return node, true
	}
	leftTruth := optimizerLiteralBits(leftLiteral, optimizerInt32) != 0
	rightTruth := optimizerLiteralBits(rightLiteral, optimizerInt32) != 0
	if node.Op == "&&" {
		return &AstNumberLiteral{IntValue: optimizerBoolInt(leftTruth && rightTruth), IsBool: true, Line: node.Line}, true
	}
	return &AstNumberLiteral{IntValue: optimizerBoolInt(leftTruth || rightTruth), IsBool: true, Line: node.Line}, true
}

func (optimizer *optimizer) exprType(expression AstExprNode) *Type {
	switch node := expression.(type) {
	case *AstNumberLiteral:
		if node.IsBool {
			return BoolType
		}
		if node.IsFloat {
			if node.FloatType != nil {
				return node.FloatType
			}
			return Float32Type
		}
		return Int32Type
	case *AstIdentNode:
		for index := len(optimizer.locals) - 1; index >= 0; index-- {
			if typ, ok := optimizer.locals[index][node.Name]; ok {
				return typ
			}
		}
		return optimizer.globals[node.Name]
	case *AstMemberExpr:
		baseType := optimizer.exprType(node.Base)
		if baseType == nil || baseType.Kind != TypeStruct || baseType.Struct == nil {
			return nil
		}
		fieldIndex, ok := baseType.Struct.FieldsByName[node.Member]
		if !ok {
			return nil
		}
		return baseType.Struct.Fields[fieldIndex].Type
	case *AstIndexExpr:
		baseType := optimizer.exprType(node.Base)
		if baseType == nil || baseType.Kind != TypeArray {
			return nil
		}
		return baseType.Base
	case *AstUnaryExpr:
		if node.Op == UnaryLogicalNot {
			return BoolType
		}
		return optimizer.exprType(node.Operand)
	case *AstBinaryExpr:
		if node.Op == "&&" || node.Op == "||" {
			return BoolType
		}
		if optimizerIsComparison(node.Op) {
			return BoolType
		}
		return optimizerPromoteNumericType(optimizer.exprType(node.Left), optimizer.exprType(node.Right))
	case *AstCallExpr:
		return optimizer.functionReturns[node.Callee]
	}
	return nil
}

type optimizerNumericKind uint8

const (
	optimizerInt32 optimizerNumericKind = iota
	optimizerInt8
	optimizerInt16
	optimizerInt64
	optimizerUint8
	optimizerUint16
	optimizerUint32
	optimizerUint64
	optimizerFloat32
	optimizerFloat64
)

func optimizerKindFromType(typ *Type) optimizerNumericKind {
	if typ == nil {
		return optimizerInt32
	}
	switch typ.Kind {
	case TypeInt8:
		return optimizerInt8
	case TypeInt16:
		return optimizerInt16
	case TypeInt64:
		return optimizerInt64
	case TypeByte, TypeUint8, TypeBool:
		return optimizerUint8
	case TypeUint16:
		return optimizerUint16
	case TypeUint32:
		return optimizerUint32
	case TypeUint64:
		return optimizerUint64
	case TypeFloat32:
		return optimizerFloat32
	case TypeFloat64:
		return optimizerFloat64
	default:
		return optimizerInt32
	}
}

func optimizerLiteralBits(literal *AstNumberLiteral, kind optimizerNumericKind) uint64 {
	if literal.IsFloat {
		return optimizerFloatToBits(literal.FloatValue, kind)
	}
	return optimizerIntToBits(int64(literal.IntValue), kind)
}

func optimizerIntToBits(value int64, kind optimizerNumericKind) uint64 {
	switch kind {
	case optimizerFloat32:
		return uint64(math.Float32bits(float32(value)))
	case optimizerFloat64:
		return math.Float64bits(float64(value))
	case optimizerInt8, optimizerUint8:
		return uint64(uint8(value))
	case optimizerInt16, optimizerUint16:
		return uint64(uint16(value))
	case optimizerInt32, optimizerUint32:
		return uint64(uint32(value))
	default:
		return uint64(value)
	}
}

func optimizerFloatToBits(value float64, kind optimizerNumericKind) uint64 {
	switch kind {
	case optimizerFloat32:
		return uint64(math.Float32bits(float32(value)))
	case optimizerFloat64:
		return math.Float64bits(value)
	default:
		return optimizerIntToBits(int64(value), kind)
	}
}

func optimizerLiteralFromBits(bits uint64, kind optimizerNumericKind, line int) *AstNumberLiteral {
	switch kind {
	case optimizerFloat32:
		return &AstNumberLiteral{FloatValue: float64(math.Float32frombits(uint32(bits))), IsFloat: true, FloatType: Float32Type, Line: line}
	case optimizerFloat64:
		return &AstNumberLiteral{FloatValue: math.Float64frombits(bits), IsFloat: true, FloatType: Float64Type, Line: line}
	}
	return &AstNumberLiteral{IntValue: int(optimizerSignedOrUnsignedValue(bits, kind)), Line: line}
}

func optimizerSignedOrUnsignedValue(bits uint64, kind optimizerNumericKind) int64 {
	switch kind {
	case optimizerInt8:
		return int64(int8(bits))
	case optimizerInt16:
		return int64(int16(bits))
	case optimizerInt32:
		return int64(int32(bits))
	case optimizerInt64:
		return int64(bits)
	case optimizerUint8:
		return int64(uint8(bits))
	case optimizerUint16:
		return int64(uint16(bits))
	case optimizerUint32:
		return int64(uint32(bits))
	default:
		return int64(bits)
	}
}

func optimizerArithmetic(kind optimizerNumericKind, op BinaryOp, left, right uint64) (uint64, string, bool) {
	if (op == BinaryDiv || op == BinaryModulo) && optimizerIsZero(kind, right) {
		if op == BinaryModulo {
			return 0, "modulo by zero", false
		}
		return 0, "division by zero", false
	}
	switch kind {
	case optimizerFloat32:
		leftValue, rightValue := math.Float32frombits(uint32(left)), math.Float32frombits(uint32(right))
		return uint64(math.Float32bits(optimizerFloat32Arithmetic(op, leftValue, rightValue))), "", true
	case optimizerFloat64:
		return math.Float64bits(optimizerFloatArithmetic(op, math.Float64frombits(left), math.Float64frombits(right))), "", true
	case optimizerInt8:
		return optimizerSignedArithmetic(op, int64(int8(left)), int64(int8(right)), 8), "", true
	case optimizerInt16:
		return optimizerSignedArithmetic(op, int64(int16(left)), int64(int16(right)), 16), "", true
	case optimizerInt32:
		return optimizerSignedArithmetic(op, int64(int32(left)), int64(int32(right)), 32), "", true
	case optimizerInt64:
		return optimizerSignedArithmetic(op, int64(left), int64(right), 64), "", true
	default:
		return optimizerUnsignedArithmetic(op, optimizerUnsignedValue(left, kind), optimizerUnsignedValue(right, kind), kind), "", true
	}
}

func optimizerSignedArithmetic(op BinaryOp, left, right int64, width uint) uint64 {
	var result int64
	switch op {
	case BinaryAdd:
		result = left + right
	case BinarySub:
		result = left - right
	case BinaryMul:
		result = left * right
	case BinaryDiv:
		result = left / right
	case BinaryModulo:
		result = left % right
	case BinaryBitwiseAnd:
		result = left & right
	case BinaryBitwiseOr:
		result = left | right
	case BinaryBitwiseXor:
		result = left ^ right
	case BinaryShiftLeft:
		result = left << (uint64(right) & uint64(width-1))
	case BinaryShiftRight:
		result = left >> (uint64(right) & uint64(width-1))
	}
	if width == 64 {
		return uint64(result)
	}
	return uint64(result) & ((uint64(1) << width) - 1)
}

func optimizerUnsignedArithmetic(op BinaryOp, left, right uint64, kind optimizerNumericKind) uint64 {
	var result uint64
	width := uint(64)
	switch kind {
	case optimizerUint8:
		width = 8
	case optimizerUint16:
		width = 16
	case optimizerUint32:
		width = 32
	}
	switch op {
	case BinaryAdd:
		result = left + right
	case BinarySub:
		result = left - right
	case BinaryMul:
		result = left * right
	case BinaryDiv:
		result = left / right
	case BinaryModulo:
		result = left % right
	case BinaryBitwiseAnd:
		result = left & right
	case BinaryBitwiseOr:
		result = left | right
	case BinaryBitwiseXor:
		result = left ^ right
	case BinaryShiftLeft:
		result = left << (right & uint64(width-1))
	case BinaryShiftRight:
		result = left >> (right & uint64(width-1))
	}
	switch kind {
	case optimizerUint8:
		return uint64(uint8(result))
	case optimizerUint16:
		return uint64(uint16(result))
	case optimizerUint32:
		return uint64(uint32(result))
	default:
		return result
	}
}

func optimizerFloat32Arithmetic(op BinaryOp, left, right float32) float32 {
	switch op {
	case "+":
		return left + right
	case "-":
		return left - right
	case "*":
		return left * right
	case "/":
		return left / right
	default:
		return 0
	}
}

func optimizerFloatArithmetic(op BinaryOp, left, right float64) float64 {
	switch op {
	case "+":
		return left + right
	case "-":
		return left - right
	case "*":
		return left * right
	case "/":
		return left / right
	default:
		return 0
	}
}

func optimizerCompare(kind optimizerNumericKind, op BinaryOp, left, right uint64) int {
	switch kind {
	case optimizerFloat32:
		return optimizerCompareFloat(op, float64(math.Float32frombits(uint32(left))), float64(math.Float32frombits(uint32(right))))
	case optimizerFloat64:
		return optimizerCompareFloat(op, math.Float64frombits(left), math.Float64frombits(right))
	case optimizerInt8, optimizerInt16, optimizerInt32, optimizerInt64:
		return optimizerCompareInt(op, optimizerSignedOrUnsignedValue(left, kind), optimizerSignedOrUnsignedValue(right, kind))
	default:
		return optimizerCompareUint(op, optimizerUnsignedValue(left, kind), optimizerUnsignedValue(right, kind))
	}
}

func optimizerUnsignedValue(bits uint64, kind optimizerNumericKind) uint64 {
	switch kind {
	case optimizerUint8:
		return uint64(uint8(bits))
	case optimizerUint16:
		return uint64(uint16(bits))
	case optimizerUint32:
		return uint64(uint32(bits))
	default:
		return bits
	}
}

func optimizerCompareInt(op BinaryOp, left, right int64) int {
	switch op {
	case "==":
		return optimizerBoolInt(left == right)
	case "!=":
		return optimizerBoolInt(left != right)
	case "<":
		return optimizerBoolInt(left < right)
	case "<=":
		return optimizerBoolInt(left <= right)
	case ">":
		return optimizerBoolInt(left > right)
	case ">=":
		return optimizerBoolInt(left >= right)
	default:
		return 0
	}
}

func optimizerCompareUint(op BinaryOp, left, right uint64) int {
	switch op {
	case "==":
		return optimizerBoolInt(left == right)
	case "!=":
		return optimizerBoolInt(left != right)
	case "<":
		return optimizerBoolInt(left < right)
	case "<=":
		return optimizerBoolInt(left <= right)
	case ">":
		return optimizerBoolInt(left > right)
	case ">=":
		return optimizerBoolInt(left >= right)
	default:
		return 0
	}
}

func optimizerCompareFloat(op BinaryOp, left, right float64) int {
	switch op {
	case "==":
		return optimizerBoolInt(left == right)
	case "!=":
		return optimizerBoolInt(left != right)
	case "<":
		return optimizerBoolInt(left < right)
	case "<=":
		return optimizerBoolInt(left <= right)
	case ">":
		return optimizerBoolInt(left > right)
	case ">=":
		return optimizerBoolInt(left >= right)
	default:
		return 0
	}
}

func optimizerIsZero(kind optimizerNumericKind, bits uint64) bool {
	switch kind {
	case optimizerFloat32:
		return math.Float32frombits(uint32(bits)) == 0
	case optimizerFloat64:
		return math.Float64frombits(bits) == 0
	default:
		return optimizerUnsignedValue(bits, kind) == 0
	}
}

func optimizerIsComparison(op BinaryOp) bool {
	switch op {
	case "==", "!=", "<", "<=", ">", ">=":
		return true
	default:
		return false
	}
}

func optimizerPromoteNumericType(left, right *Type) *Type {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	if left == right {
		return left
	}
	if left.Kind == TypeFloat64 || right.Kind == TypeFloat64 {
		return Float64Type
	}
	if left.Kind == TypeFloat32 || right.Kind == TypeFloat32 {
		return Float32Type
	}
	if left.Kind == TypeUint64 || right.Kind == TypeUint64 {
		return Uint64Type
	}
	if left.Kind == TypeInt64 || right.Kind == TypeInt64 {
		return Int64Type
	}
	if left.Kind == TypeUint32 || right.Kind == TypeUint32 {
		return Uint32Type
	}
	if left.Kind == TypeUint16 || right.Kind == TypeUint16 {
		return Uint16Type
	}
	if left.Kind == TypeUint8 || right.Kind == TypeUint8 || left.Kind == TypeByte || right.Kind == TypeByte {
		return Uint8Type
	}
	return Int32Type
}

func optimizerBoolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
