package cova

import "testing"

const benchmarkVMIterations = 10000

var benchmarkVMStatus VMStatus

func benchmarkLinkedProgram(b *testing.B, source string) *LinkedProgram {
	b.Helper()
	ctx := NewContext()
	tokens, ok := Tokenize(ctx, source)
	if !ok {
		b.Fatalf("Tokenize failed: %v", issueDescriptions(ctx))
	}
	program, ok := Parse(ctx, tokens)
	if !ok {
		b.Fatalf("Parse failed: %v", issueDescriptions(ctx))
	}

	var err error
	compiled, err := NewCompiler(ctx).Compile(program)
	if err != nil {
		b.Fatalf("Compile failed: %v", err)
	}
	linked, success := NewLinker(ctx, 0, 0).Link(program, compiled)
	if !success {
		b.Fatalf("Link failed")
	}
	return linked
}

func benchmarkRunLoaded(b *testing.B, linked *LinkedProgram, operationsPerRun uint64) {
	b.Helper()
	frameCapacity := max(linked.FrameByteSize, uint32(256))
	vm := NewVMWithConfig(VMConfig{
		FrameCapacity:     frameCapacity,
		StackCapacity:     256,
		CallFrameCapacity: 8,
	})
	if status := vm.LoadProgram(linked); status != VMStatusOK {
		b.Fatalf("LoadProgram failed: %s", status)
	}
	b.ReportAllocs()
	b.ResetTimer()
	var status VMStatus
	for index := 0; index < b.N; index++ {
		status = vm.RunLoaded()
	}
	b.StopTimer()
	b.ReportMetric(float64(operationsPerRun)*float64(b.N)/b.Elapsed().Seconds(), "script-iterations/s")
	benchmarkVMStatus = status
	if status != VMStatusOK {
		b.Fatalf("RunLoaded failed: %s", status)
	}
}

func BenchmarkVMInt32Arithmetic(b *testing.B) {
	linked := benchmarkLinkedProgram(b, `
int script_main() {
	int index = 0;
	int value = 1;
	while (index < 10000) {
		value = value * 3 + 1;
		index = index + 1;
	}
	return value;
}
`)
	benchmarkRunLoaded(b, linked, benchmarkVMIterations)
}

func BenchmarkVMFloat32Arithmetic(b *testing.B) {
	linked := benchmarkLinkedProgram(b, `
float32 script_main() {
	int index = 0;
	float32 value = 1.0f;
	while (index < 10000) {
		value = value * 1.0001f + 0.25f;
		index = index + 1;
	}
	return value;
}
`)
	benchmarkRunLoaded(b, linked, benchmarkVMIterations)
}

func BenchmarkVMCompareAndBranch(b *testing.B) {
	linked := benchmarkLinkedProgram(b, `
int script_main() {
	int index = 0;
	int value = 0;
	while (index < 10000) {
		if (index < 5000) {
			value = value + 1;
		} else {
			value = value - 1;
		}
		index = index + 1;
	}
	return value;
}
`)
	benchmarkRunLoaded(b, linked, benchmarkVMIterations)
}

func BenchmarkVMMemoryAccess(b *testing.B) {
	linked := benchmarkLinkedProgram(b, `
int index;
int value;
int script_main() {
	index = 0;
	value = 1;
	while (index < 10000) {
		value = value + index;
		index = index + 1;
	}
	return value;
}
`)
	benchmarkRunLoaded(b, linked, benchmarkVMIterations)
}

func BenchmarkVMFunctionCalls(b *testing.B) {
	linked := benchmarkLinkedProgram(b, `
int add_one(int value) {
	return value + 1;
}
int script_main() {
	int index = 0;
	int value = 0;
	while (index < 10000) {
		value = add_one(value);
		index = index + 1;
	}
	return value;
}
`)
	benchmarkRunLoaded(b, linked, benchmarkVMIterations)
}

func BenchmarkVMReset(b *testing.B) {
	var code CodeMemory
	code.AppendInstruction(makeInstruction(OpRet, KindNone, ModeNone, FlagNone))
	linked := &LinkedProgram{
		Text:        code,
		EntryPoint:  0,
		BSSByteSize: 4096,
		Functions:   []ScriptFunctionDescriptor{{BodyAddress: 0, ReturnKind: KindVoid}},
	}
	vm := NewVMWithConfig(VMConfig{
		FrameCapacity:     linked.FrameByteSize,
		StackCapacity:     256,
		CallFrameCapacity: 8,
	})
	if status := vm.LoadProgram(linked); status != VMStatusOK {
		b.Fatalf("LoadProgram failed: %s", status)
	}
	b.ReportAllocs()
	b.ResetTimer()
	var status VMStatus
	for index := 0; index < b.N; index++ {
		status = vm.Reset()
	}
	benchmarkVMStatus = status
	if status != VMStatusOK {
		b.Fatalf("Reset failed: %s", status)
	}
}
