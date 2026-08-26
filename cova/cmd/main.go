package main

import (
	"encoding/binary"
	"fmt"
	"os"

	cova "github.com/jurgen-kluft/ccova/cova"
)

func main() {
	externMemory := make([]byte, 8)
	binary.LittleEndian.PutUint32(externMemory[4:], 45)
	script := `
extern(0) void log_alert(int data);
extern int player_health;
int health_drop;

void script_main() {
	health_drop = 5;
	if ((player_health - 40) + 1) {
		log_alert(player_health);
		reduce_health(health_drop);
	}
	return;
}

void reduce_health(int delta) {
	player_health = player_health - delta;
	return;
}
`

	ctx := cova.NewContext()
	tokens, ok := cova.Tokenize(ctx, script)
	checkOK(ctx, ok)

	program, ok := cova.Parse(ctx, tokens)
	checkOK(ctx, ok)
	checkOK(ctx, cova.Optimize(ctx, program))

	compiler := cova.NewCompiler(ctx)
	compiled, ok := compiler.Compile(program)
	checkOK(ctx, ok)

	linker := cova.NewLinker(ctx, len(externMemory), 1)
	linked, success := linker.Link(program, compiled)
	checkOK(ctx, success)
	checkOK(ctx, linker.Report(os.Stdout, compiled, linked))

	vm := cova.NewVM(256)
	vm.BindExternBlock(externMemory)
	vm.RegisterExternDispatcher(0, func(_ uintptr, vm *cova.VM, importID uint32) cova.VMStatus {
		if importID != 0 {
			return cova.VMStatusHostFailure
		}
		value, status := vm.PopInt32()
		if status != cova.VMStatusOK {
			return status
		}
		fmt.Printf("host log_alert(%d)\n", value)
		return cova.VMStatusOK
	})

	checkStatus(vm.Run(linked))
	fmt.Printf("hostPlayerHealth=%d\n", int(int32(binary.LittleEndian.Uint32(externMemory[4:]))))
}
func check(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func checkOK(ctx *cova.Context, ok bool) {
	if ok {
		return
	}
	if ctx != nil {
		ctx.Report(os.Stdout, os.Stderr)
	}
	os.Exit(1)
}

func checkStatus(status cova.VMStatus) {
	if status == cova.VMStatusOK {
		return
	}
	fmt.Fprintf(os.Stderr, "VM failed: %s\n", status)
	os.Exit(1)
}
