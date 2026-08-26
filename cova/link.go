package cova

import (
	"fmt"
	"io"
)

type Linker struct {
	Ctx              *Context
	VariableCapacity int
	FunctionCapacity int
}

func NewLinker(ctx *Context, variableCapacity, functionCapacity int) *Linker {
	return &Linker{Ctx: ctx, VariableCapacity: variableCapacity, FunctionCapacity: functionCapacity}
}

// Report writes a size and symbol overview for a successfully linked program.
func (linker *Linker) Report(writer io.Writer, compiled *RelocatableProgram, linked *LinkedProgram) error {
	if linker == nil {
		return fmt.Errorf("link report error: linker is nil")
	}
	if writer == nil {
		return fmt.Errorf("link report error: writer is nil")
	}
	if compiled == nil {
		return fmt.Errorf("link report error: compiled program is nil")
	}
	if linked == nil {
		return fmt.Errorf("link report error: linked program is nil")
	}

	usedExternalFunctions := make(map[uint32]struct{}, len(compiled.UsedExternalFunctionIDs))
	for _, tempFuncID := range compiled.UsedExternalFunctionIDs {
		usedExternalFunctions[tempFuncID] = struct{}{}
	}
	externalFunctions := 0
	unusedExternalFunctions := make([]string, 0)
	for _, function := range compiled.Functions {
		if function.Scope == ScopeExtern {
			externalFunctions++
			if _, used := usedExternalFunctions[function.TempFuncID]; !used {
				unusedExternalFunctions = append(unusedExternalFunctions, function.Name)
			}
		}
	}

	externalVariables := 0
	var externalByteSize uint64
	if linked.DebugSymbols != nil {
		for _, variable := range linked.DebugSymbols.ExternSymbols {
			if variable.Kind != DeclVariable {
				continue
			}
			externalVariables++
			end := uint64(variable.ByteOffset) + uint64(variable.ByteSize)
			if end > externalByteSize {
				externalByteSize = end
			}
		}
	}

	_, err := fmt.Fprintf(writer, "Text size: %d bytes\nBSS size: %d bytes\nData size: %d bytes\nConst size: %d bytes\nLocal Functions: %d functions\nExternal Functions: %d functions, %d unused\nExternal Variables: %d variables, %d bytes\n",
		len(linked.Text), linked.BSSByteSize, linked.DataByteSize, linked.ConstByteSize,
		len(linked.Functions), externalFunctions, len(unusedExternalFunctions), externalVariables, externalByteSize)
	if err != nil || len(unusedExternalFunctions) == 0 {
		return err
	}
	if _, err := fmt.Fprintln(writer, "Unused External Functions:"); err != nil {
		return err
	}
	for _, name := range unusedExternalFunctions {
		if _, err := fmt.Fprintf(writer, "  %s\n", name); err != nil {
			return err
		}
	}
	return nil
}

func (linker *Linker) Link(program *AstProgramNode, compiled *RelocatableProgram) (*LinkedProgram, error) {
	if program == nil {
		return nil, fmt.Errorf("link error: program is nil")
	}
	if compiled == nil {
		return nil, fmt.Errorf("link error: compiled program is nil")
	}
	if linker == nil {
		return nil, fmt.Errorf("link error: linker is nil")
	}
	if linker.VariableCapacity < 0 {
		return nil, fmt.Errorf("link error: variable capacity %d is negative", linker.VariableCapacity)
	}
	if linker.FunctionCapacity < 0 {
		return nil, fmt.Errorf("link error: function capacity %d is negative", linker.FunctionCapacity)
	}

	for _, binding := range compiled.ProgramSymbols.ExternSymbols {
		byteOffset := int(binding.ByteOffset)
		byteSize := int(binding.ByteSize)
		if byteOffset+byteSize > linker.VariableCapacity {
			return nil, fmt.Errorf("link error: extern variable %q requests byte range [%d,%d), but extern memory capacity is %d", binding.Name, byteOffset, byteOffset+byteSize, linker.VariableCapacity)
		}
		if binding.ByteAlignment > 1 && binding.ByteOffset%binding.ByteAlignment != 0 {
			return nil, fmt.Errorf("link error: extern variable %q byte offset %d is not aligned to %d", binding.Name, byteOffset, binding.ByteAlignment)
		}
	}

	tempToFunction := make(map[uint32]int, len(compiled.Functions))
	functions := make([]ScriptFunctionDescriptor, 0, len(compiled.Functions))
	paramKinds := make([]ValueKind, 0)
	paramOffsets := make([]uint32, 0)
	for _, binding := range compiled.Functions {
		switch binding.Scope {
		case ScopeExtern:
			if int(binding.SlotIndex) >= linker.FunctionCapacity {
				return nil, fmt.Errorf("link error: host-linked function %q requests slot %d, but function capacity is %d", binding.Name, binding.SlotIndex, linker.FunctionCapacity)
			}
		case ScopeBSS:
			if binding.ParamCount != lenU32(binding.ParamTypes) || binding.ParamCount != lenU32(binding.ParamOffsets) {
				return nil, fmt.Errorf("link error: function %q has inconsistent parameter metadata", binding.Name)
			}
			paramStart := lenU32(paramKinds)
			for index, typ := range binding.ParamTypes {
				kind := valueKindFromType(typ)
				if kind == KindNone || kind == KindVoid {
					return nil, fmt.Errorf("link error: function %q parameter %d has unsupported kind %d", binding.Name, index, kind)
				}
				paramKinds = append(paramKinds, kind)
				paramOffsets = append(paramOffsets, binding.ParamOffsets[index])
			}
			tempToFunction[binding.TempFuncID] = len(functions)
			functions = append(functions, ScriptFunctionDescriptor{
				BodyAddress:   binding.ScriptAddress,
				ParamStart:    paramStart,
				ParamCount:    binding.ParamCount,
				FrameByteSize: binding.FrameByteSize,
				ReturnKind:    valueKindFromType(binding.Type),
			})
		default:
			return nil, fmt.Errorf("link error: function %q has invalid scope %d", binding.Name, binding.Scope)
		}
	}

	linkedText := compiled.Text.Clone()
	for _, patch := range compiled.CallPatches {
		functionIndex, ok := tempToFunction[patch.TempFuncID]
		if !ok {
			return nil, fmt.Errorf("link error on line %d: unresolved function id %d", patch.Line, patch.TempFuncID)
		}
		linkedText.PatchUint32(patch.OperandPos, uint32(functionIndex))
	}

	entryPoint, ok := tempToFunction[compiled.EntryFunction]
	if !ok {
		return nil, fmt.Errorf("link error: entry function id %d was not finalized", compiled.EntryFunction)
	}
	linked := &LinkedProgram{
		Text:          linkedText,
		EntryPoint:    uint32(entryPoint),
		Functions:     functions,
		ParamKinds:    paramKinds,
		ParamOffsets:  paramOffsets,
		FrameSize:     compiled.FrameSize,
		FrameByteSize: compiled.FrameByteSize,
		ConstByteSize: compiled.ConstByteSize,
		ConstData:     append([]byte(nil), compiled.ConstData...),
		DataByteSize:  compiled.DataByteSize,
		DataData:      append([]byte(nil), compiled.DataData...),
		BSSSize:       compiled.BSSSize,
		BSSByteSize:   compiled.BSSByteSize,
		DebugSymbols:  CopyProgramSymbols(compiled.ProgramSymbols),
	}
	return linked, nil
}
