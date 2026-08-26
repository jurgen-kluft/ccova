package cova

import "fmt"

type BuiltInFunction uint16

const (
	BuiltInInvalid BuiltInFunction = iota
)

type BuiltInOperation byte

const (
	BuiltInOperationInvalid BuiltInOperation = iota
	BuiltInAbs
	BuiltInSin
	BuiltInCos
	BuiltInTan
	BuiltInAsin
	BuiltInAcos
	BuiltInAtan
	BuiltInPow
	BuiltInSqrt
	BuiltInMin
	BuiltInMax
	BuiltInMap
	BuiltInRandom
	BuiltInClamp
	BuiltInSmoothStep
	BuiltInLerp
	BuiltInSlerp
)

func builtInNumArgs(operation BuiltInOperation) int {
	switch operation {
	case BuiltInAbs, BuiltInSin, BuiltInCos, BuiltInTan, BuiltInAsin, BuiltInAcos, BuiltInAtan, BuiltInSqrt:
		return 1
	case BuiltInPow, BuiltInMin, BuiltInMax:
		return 2
	case BuiltInClamp:
		return 3
	case BuiltInSmoothStep, BuiltInLerp, BuiltInSlerp:
		return 3
	case BuiltInMap:
		return 5
	case BuiltInRandom:
		return 0
	default:
		return 0
	}
}

type builtInSignature struct {
	resultType *Type
	argTypes   []*Type
}

func builtInAcceptsArity(operation BuiltInOperation, arity int) bool {
	if (operation == BuiltInSmoothStep || operation == BuiltInLerp) && arity == 4 {
		return true
	}
	return arity == builtInNumArgs(operation)
}

func makeBuiltInFunction(operation BuiltInOperation, kind ValueKind) BuiltInFunction {
	return BuiltInFunction(uint16(operation&0x7f)<<4 | uint16(kind&0x0f))
}

func (function BuiltInFunction) Operation() BuiltInOperation {
	return BuiltInOperation((function >> 4) & 0x7f)
}

func (function BuiltInFunction) Kind() ValueKind {
	return ValueKind(function & 0x0f)
}

func lookupBuiltInOperation(name string) (BuiltInOperation, bool) {
	operations := map[string]BuiltInOperation{
		"math::abs":        BuiltInAbs,
		"math::sin":        BuiltInSin,
		"math::cos":        BuiltInCos,
		"math::tan":        BuiltInTan,
		"math::asin":       BuiltInAsin,
		"math::acos":       BuiltInAcos,
		"math::atan":       BuiltInAtan,
		"math::pow":        BuiltInPow,
		"math::sqrt":       BuiltInSqrt,
		"math::min":        BuiltInMin,
		"math::max":        BuiltInMax,
		"math::map":        BuiltInMap,
		"math::random":     BuiltInRandom,
		"math::clamp":      BuiltInClamp,
		"math::smoothStep": BuiltInSmoothStep,
		"math::lerp":       BuiltInLerp,
		"math::slerp":      BuiltInSlerp,
	}
	operation, ok := operations[name]
	return operation, ok
}

func (fc *functionCompiler) builtInResultType(call *AstCallExpr, operation BuiltInOperation) *Type {
	signature := fc.resolveBuiltInSignature(call, operation)
	return signature.resultType
}

func (fc *functionCompiler) resolveBuiltInSignature(call *AstCallExpr, operation BuiltInOperation) builtInSignature {
	if !builtInAcceptsArity(operation, len(call.Args)) {
		return builtInSignature{}
	}
	if operation == BuiltInRandom {
		return builtInSignature{resultType: Int32Type}
	}
	for _, arg := range call.Args {
		kind := valueKindFromType(fc.exprType(arg))
		if !isNumericKind(kind) || kind == KindBool {
			return builtInSignature{}
		}
	}
	if operation == BuiltInAbs {
		resultType := fc.exprType(call.Args[0])
		return builtInSignature{resultType: resultType, argTypes: []*Type{resultType}}
	}
	if operation == BuiltInMin || operation == BuiltInMax || operation == BuiltInMap || operation == BuiltInClamp {
		resultType := fc.exprType(call.Args[0])
		for _, arg := range call.Args[1:] {
			resultType = promoteNumericType(resultType, fc.exprType(arg))
		}
		argTypes := make([]*Type, len(call.Args))
		for index := range argTypes {
			argTypes[index] = resultType
		}
		return builtInSignature{resultType: resultType, argTypes: argTypes}
	}
	if (operation == BuiltInSmoothStep || operation == BuiltInLerp) && len(call.Args) == 4 {
		resultType := Int32Type
		for _, arg := range call.Args[:3] {
			switch valueKindFromType(fc.exprType(arg)) {
			case KindInt8, KindInt16, KindInt32:
			case KindInt64:
				resultType = Int64Type
			default:
				return builtInSignature{}
			}
		}
		return builtInSignature{resultType: resultType, argTypes: []*Type{resultType, resultType, resultType, Uint8Type}}
	}
	switch operation {
	case BuiltInSin, BuiltInCos, BuiltInTan, BuiltInAsin, BuiltInAcos, BuiltInAtan, BuiltInPow, BuiltInSqrt, BuiltInSmoothStep, BuiltInLerp, BuiltInSlerp:
		resultType := Float32Type
		for _, arg := range call.Args {
			if valueKindFromType(fc.exprType(arg)).Size() == 8 {
				resultType = Float64Type
			}
		}
		argTypes := make([]*Type, len(call.Args))
		for index := range argTypes {
			argTypes[index] = resultType
		}
		return builtInSignature{resultType: resultType, argTypes: argTypes}
	default:
		return builtInSignature{}
	}
}

func (fc *functionCompiler) compileBuiltInCall(call *AstCallExpr, operation BuiltInOperation, expectedKind ValueKind) {
	if !builtInAcceptsArity(operation, len(call.Args)) {
		if operation == BuiltInSmoothStep || operation == BuiltInLerp {
			fc.fail(fmt.Errorf("compile error on line %d: built-in function %q expects 3 or 4 arguments, got %d", call.Line, call.Callee, len(call.Args)))
			return
		}
		fc.fail(fmt.Errorf("compile error on line %d: built-in function %q expects %d arguments, got %d", call.Line, call.Callee, builtInNumArgs(operation), len(call.Args)))
		return
	}
	for _, arg := range call.Args {
		argKind := valueKindFromType(fc.exprType(arg))
		if !isNumericKind(argKind) || argKind == KindBool {
			fc.fail(fmt.Errorf("compile error on line %d: built-in function %q requires numeric arguments", call.Line, call.Callee))
			return
		}
	}
	signature := fc.resolveBuiltInSignature(call, operation)
	resultType := signature.resultType
	resultKind := valueKindFromType(resultType)
	if resultKind == KindNone {
		fc.fail(fmt.Errorf("compile error on line %d: built-in function %q does not support these argument types", call.Line, call.Callee))
		return
	}
	for index, arg := range call.Args {
		fc.compileExprAs(arg, signature.argTypes[index])
		if fc.err != nil {
			return
		}
	}
	fc.code.AppendInstruction(makeBuiltInInstruction(makeBuiltInFunction(operation, resultKind)))
	fc.emitConvertIfNeeded(resultKind, expectedKind)
}
