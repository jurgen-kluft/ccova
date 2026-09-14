package cova

import "fmt"

type BuiltInFunction uint16

const (
	BuiltInInvalid BuiltInFunction = iota
)

type BuiltInOperation byte

const (
	BuiltInOperationInvalid BuiltInOperation = iota
	BuiltInAbs                               // math::abs(value)
	BuiltInSin                               // math::sin(value)
	BuiltInCos                               // math::cos(value)
	BuiltInTan                               // math::tan(value)
	BuiltInAsin                              // math::asin(value)
	BuiltInAcos                              // math::acos(value)
	BuiltInAtan                              // math::atan(value)
	BuiltInPow                               // math::pow(base, exponent)
	BuiltInSqrt                              // math::sqrt(value)
	BuiltInMin                               // math::min(a, b)
	BuiltInMax                               // math::max(a, b)
	BuiltInMap                               // math::map(value, inMin, inMax, outMin, outMax)
	BuiltInRandom                            // math::random() -> int32
	BuiltInClamp                             // math::clamp(value, min, max)
	BuiltInSmoothStep                        // math::smoothStep(edge0, edge1, x) or (start, end, t, shift)
	BuiltInLerp                              // math::lerp(a, b, t) or (start, end, t, shift)
	BuiltInSlerp                             // math::slerp(a, b, t)
	BuiltInFrameTime                         // time::frameTime() -> float32
	BuiltInTimerStart                        // time::timerStart(int32 id, u32 timeout_ms)
	BuiltInTimerStop                         // time::timerStop(int32 id)
	BuiltInTimerReset                        // time::timerReset(int32 id)
	BuiltInTimerElapsed                      // time::timerElapsed(int32 id) -> float32
)

func builtInNumArgs(operation BuiltInOperation) int {
	switch operation {
	case BuiltInTimerStart:
		return 2
	case BuiltInTimerStop, BuiltInTimerReset, BuiltInTimerElapsed:
		return 1
	case BuiltInAbs, BuiltInSin, BuiltInCos, BuiltInTan, BuiltInAsin, BuiltInAcos, BuiltInAtan, BuiltInSqrt:
		return 1
	case BuiltInPow, BuiltInMin, BuiltInMax:
		return 2
	case BuiltInClamp:
		return 3
	case BuiltInSmoothStep, BuiltInLerp, BuiltInSlerp:
		return 4
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
		// math
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
		// timer
		"time::frameTime":    BuiltInFrameTime,
		"time::timerStart":   BuiltInTimerStart,
		"time::timerStop":    BuiltInTimerStop,
		"time::timerReset":   BuiltInTimerReset,
		"time::timerElapsed": BuiltInTimerElapsed,
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
	if operation == BuiltInSmoothStep || operation == BuiltInLerp || operation == BuiltInSlerp {

		// Type is one of the following:
		// - int32, int64, float32, float64

		// Function signatures are as follows:
		// - type lerp(start type, end type, t type, shift uint8)
		// - type slerp(start type, end type, t type, shift uint8)
		// - type smoothstep(start type, end type, t type, shift uint8)

		// So the last argument is always 'uint8 shift', and the first three
		// arguments must be of the same type (int32, int64, float32, float64).

		resultType := Int32Type
		var ok bool
		for _, arg := range call.Args[:3] {
			valueType := fc.exprType(arg)
			if resultType, ok = PromoteType(resultType, valueType); !ok {
				return builtInSignature{}
			}
		}
		return builtInSignature{resultType: resultType, argTypes: []*Type{resultType, resultType, resultType, Uint8Type}}
	}
	switch operation {
	case BuiltInSin, BuiltInCos, BuiltInTan, BuiltInAsin, BuiltInAcos, BuiltInAtan, BuiltInPow, BuiltInSqrt:
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
