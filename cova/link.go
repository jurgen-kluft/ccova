package cova

import (
	"fmt"
	"io"
)

type Linker struct {
	Ctx                *Context
	VariableCapacities [ExternMemoryBankCount]int
	FunctionCapacity   int
}

func NewLinker(ctx *Context, variableCapacities [ExternMemoryBankCount]int, functionCapacity int) *Linker {
	return &Linker{Ctx: ctx, VariableCapacities: variableCapacities, FunctionCapacity: functionCapacity}
}

// Report writes a size and symbol overview for a successfully linked program.
func (linker *Linker) Report(writer io.Writer, compiled *RelocatableProgram, linked *LinkedProgram) bool {
	if writer == nil {
		linker.Ctx.AddError("link report error: writer is nil")
		return false
	}
	if compiled == nil {
		linker.Ctx.AddError("link report error: compiled program is nil")
		return false
	}
	if linked == nil {
		linker.Ctx.AddError("link report error: linked program is nil")
		return false
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
	var externalBankSizes [ExternMemoryBankCount]uint64
	if linked.DebugSymbols != nil {
		for _, variable := range linked.DebugSymbols.ExternSymbols {
			if variable.Kind != DeclVariable || variable.SlotIndex >= ExternMemoryBankCount {
				continue
			}
			externalVariables++
			end := uint64(variable.ByteOffset) + uint64(variable.ByteSize)
			if end > externalBankSizes[variable.SlotIndex] {
				externalBankSizes[variable.SlotIndex] = end
			}
		}
	}

	_, err := fmt.Fprintf(writer, "Text size: %d bytes\nBSS size: %d bytes\nData size: %d bytes\nConst size: %d bytes\nLocal Functions: %d functions\nExternal Functions: %d functions, %d unused\nExternal Variables: %d variables\n",
		len(linked.Text), linked.BSSByteSize, linked.DataByteSize, linked.ConstByteSize,
		len(linked.Functions), externalFunctions, len(unusedExternalFunctions), externalVariables)
	if err != nil {
		linker.Ctx.AddError("link report error: failed to write report header: %v", err)
		return false
	}
	for bank := 0; bank < ExternMemoryBankCount; bank++ {
		if externalBankSizes[bank] == 0 {
			continue
		}
		if _, err := fmt.Fprintf(writer, "External Memory %d: %d bytes\n", bank, externalBankSizes[bank]); err != nil {
			linker.Ctx.AddError("link report error: failed to write external memory bank %d: %v", bank, err)
			return false
		}
	}
	if len(unusedExternalFunctions) > 0 {
		if _, err := fmt.Fprintf(writer, "Unused External Functions: %d\n", len(unusedExternalFunctions)); err != nil {
			linker.Ctx.AddError("link report error: failed to write unused external functions header: %v", err)
			return false
		}
		for _, name := range unusedExternalFunctions {
			if _, err := fmt.Fprintf(writer, "  %s\n", name); err != nil {
				linker.Ctx.AddError("link report error: failed to write unused external function %q: %v", name, err)
				return false
			}
		}
	}
	return true
}

func (linker *Linker) failErrorf(format string, args ...any) {
	linker.Ctx.AddError(format, args...)
}

func (linker *Linker) Link(program *AstProgramNode, compiled *RelocatableProgram) (linked *LinkedProgram, success bool) {
	success = false
	linked = nil
	if program == nil {
		linker.failErrorf("link error: program is nil")
		return linked, success
	}
	if compiled == nil {
		linker.failErrorf("link error: compiled program is nil")
		return linked, success
	}
	if linker == nil {
		linker.failErrorf("link error: linker is nil")
		return linked, success
	}
	if linker.FunctionCapacity < 0 {
		linker.failErrorf("link error: function capacity %d is negative", linker.FunctionCapacity)
		return linked, success
	}
	for bank := 0; bank < ExternMemoryBankCount; bank++ {
		if linker.VariableCapacities[bank] < 0 {
			linker.failErrorf("link error: variable capacity %d for memory bank %d is negative", linker.VariableCapacities[bank], bank)
			return linked, success
		}
	}

	for _, binding := range compiled.ProgramSymbols.ExternSymbols {
		bank := binding.SlotIndex
		if bank >= ExternMemoryBankCount {
			linker.failErrorf("link error: extern variable %q uses invalid memory bank %d", binding.Name, bank)
			return linked, success
		}
		byteOffset := int(binding.ByteOffset)
		byteSize := int(binding.ByteSize)
		if byteOffset+byteSize > linker.VariableCapacities[bank] {
			linker.failErrorf("link error: extern variable %q requests byte range [%d,%d) in memory bank %d, but bank capacity is %d", binding.Name, byteOffset, byteOffset+byteSize, bank, linker.VariableCapacities[bank])
			return linked, success
		}
		if binding.ByteAlignment > 1 && binding.ByteOffset%binding.ByteAlignment != 0 {
			linker.failErrorf("link error: extern variable %q byte offset %d is not aligned to %d", binding.Name, byteOffset, binding.ByteAlignment)
			return linked, success
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
				linker.failErrorf("link error: host-linked function %q requests slot %d, but function capacity is %d", binding.Name, binding.SlotIndex, linker.FunctionCapacity)
				return linked, success
			}
		case ScopeBSS:
			if binding.ParamCount != lenU32(binding.ParamTypes) || binding.ParamCount != lenU32(binding.ParamOffsets) {
				linker.failErrorf("link error: function %q has inconsistent parameter metadata", binding.Name)
				return linked, success
			}
			paramStart := lenU32(paramKinds)
			for index, typ := range binding.ParamTypes {
				kind := valueKindFromType(typ)
				if kind == KindNone || kind == KindVoid {
					linker.failErrorf("link error: function %q parameter %d has unsupported kind %d", binding.Name, index, kind)
					return linked, success
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
			linker.failErrorf("link error: function %q has invalid scope %d", binding.Name, binding.Scope)
			return linked, success
		}
	}

	linkedText := compiled.Text.Clone()
	for _, patch := range compiled.CallPatches {
		functionIndex, ok := tempToFunction[patch.TempFuncID]
		if !ok {
			linker.failErrorf("link error on line %d: unresolved function id %d", patch.Line, patch.TempFuncID)
			return linked, success
		}
		linkedText.PatchUint32(patch.OperandPos, uint32(functionIndex))
	}

	entryPoint, ok := tempToFunction[compiled.EntryFunction]
	if !ok {
		linker.failErrorf("link error: entry function id %d was not finalized", compiled.EntryFunction)
		return linked, success
	}

	success = true
	linked = &LinkedProgram{
		Text:            linkedText,
		EntryPoint:      uint32(entryPoint),
		Functions:       functions,
		ParamKinds:      paramKinds,
		ParamOffsets:    paramOffsets,
		FrameSize:       compiled.FrameSize,
		FrameByteSize:   compiled.FrameByteSize,
		ConstByteSize:   compiled.ConstByteSize,
		ConstData:       append([]byte(nil), compiled.ConstData...),
		DataByteSize:    compiled.DataByteSize,
		DataData:        append([]byte(nil), compiled.DataData...),
		BSSSize:         compiled.BSSSize,
		BSSByteSize:     compiled.BSSByteSize,
		ExternByteSizes: compiled.ExternByteSizes,
		DebugSymbols:    CopyProgramSymbols(compiled.ProgramSymbols),
	}
	return linked, success
}
