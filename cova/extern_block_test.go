package cova

import (
	"encoding/binary"
	"testing"
)

func TestExternAddressUsesByteOffset(t *testing.T) {
	externMemory := make([]byte, 8)
	binary.LittleEndian.PutUint32(externMemory[4:], 21)

	script := `
struct extern_state_t {
	byte prefix;
	int value;
};

extern(0) extern_state_t state;

void script_main() {
	state.value = state.value + 9;
	return;
}
`

	linked := mustLinkProgram(t, script, len(externMemory), 0)
	vm := NewVM(testFrameCapacityBytes)
	vm.BindExternBlock(0, externMemory)
	if status := vm.Run(linked, 0); status != VMStatusOK {
		t.Fatalf("Run failed: %s", status)
	}
	if got := int(int32(binary.LittleEndian.Uint32(externMemory[4:]))); got != 30 {
		t.Fatalf("expected extern byte offset 4 to hold 30, got %d", got)
	}
	if got := int(int32(binary.LittleEndian.Uint32(externMemory[0:]))); got != 0 {
		t.Fatalf("expected extern byte offset 0 to remain 0, got %d", got)
	}
}

func TestExternStructMemberAndArrayAccess(t *testing.T) {
	externMemory := make([]byte, 12)
	script := `
struct light_state_t {
	uint32 m_hsv;
	uint8 m_on_off;
	uint8 m_warmth;
};

struct home_state_t {
	char m_data[3];
	light_state_t m_light;
};

extern(0) home_state_t g_home;

void script_main() {
	g_home.m_data[1] = 65;
	g_home.m_light.m_on_off = 1;
	if (g_home.m_light.m_on_off) {
		g_home.m_light.m_warmth = 7;
	}
	return;
}
`

	linked := mustLinkProgram(t, script, len(externMemory), 0)
	vm := NewVM(testFrameCapacityBytes)
	vm.BindExternBlock(0, externMemory)
	if status := vm.Run(linked, 0); status != VMStatusOK {
		t.Fatalf("Run failed: %s", status)
	}
	if got := externMemory[1]; got != 65 {
		t.Fatalf("expected indexed char at offset 1 to be 65, got %d", got)
	}
	if got := externMemory[8]; got != 1 {
		t.Fatalf("expected nested on/off member at offset 8 to be 1, got %d", got)
	}
	if got := externMemory[9]; got != 7 {
		t.Fatalf("expected conditional warmth write at offset 9 to be 7, got %d", got)
	}
}

func TestExternMultipleMemoryBanks(t *testing.T) {
	gameMemory := make([]byte, 8)
	hudMemory := make([]byte, 4)
	debugMemory := make([]byte, 4)
	binary.LittleEndian.PutUint32(gameMemory[0:], 10)

	script := `
struct game_state_t {
	int health;
	int score;
};

struct hud_state_t {
	int visible;
};

struct debug_state_t {
	int counter;
};

extern(0) game_state_t game;
extern(1) hud_state_t hud;
extern(3) debug_state_t debug;

void script_main() {
	game.score = game.health + 5;
	hud.visible = 1;
	debug.counter = debug.counter + 2;
	return;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)
	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}
	capacities := [ExternMemoryBankCount]int{0: len(gameMemory), 1: len(hudMemory), 3: len(debugMemory)}
	linked, success := NewLinker(ctx, capacities, 0).Link(program, compiled)
	if !success {
		t.Fatalf("Link failed")
	}
	if linked.ExternByteSizes != [ExternMemoryBankCount]uint32{0: 8, 1: 4, 3: 4} {
		t.Fatalf("unexpected linked extern bank sizes: %v", linked.ExternByteSizes)
	}

	vm := NewVM(testFrameCapacityBytes)
	vm.BindExternBlock(0, gameMemory)
	vm.BindExternBlock(1, hudMemory)
	vm.BindExternBlock(3, debugMemory)
	if status := vm.Run(linked, 0); status != VMStatusOK {
		t.Fatalf("Run failed: %s", status)
	}
	if got := int32(binary.LittleEndian.Uint32(gameMemory[4:])); got != 15 {
		t.Fatalf("expected game.score 15 in bank 0, got %d", got)
	}
	if got := int32(binary.LittleEndian.Uint32(hudMemory[0:])); got != 1 {
		t.Fatalf("expected hud.visible 1 in bank 1, got %d", got)
	}
	if got := int32(binary.LittleEndian.Uint32(debugMemory[0:])); got != 2 {
		t.Fatalf("expected debug.counter 2 in bank 3, got %d", got)
	}
}

func TestExternUnboundBankFaultsAtAccess(t *testing.T) {
	script := `
struct game_state_t {
	int health;
};

extern(1) game_state_t game;

void script_main() {
	game.health = 1;
	return;
}
`

	linked := mustLinkProgram(t, script, 8, 0)
	vm := NewVM(testFrameCapacityBytes)
	if status := vm.Run(linked, 0); status != VMStatusInvalidAddress {
		t.Fatalf("expected unbound memory bank access to fault with invalid address, got %s", status)
	}
}

func TestLoadProgramRejectsTooSmallBoundBank(t *testing.T) {
	script := `
struct game_state_t {
	int health;
	int score;
};

extern(2) game_state_t game;

void script_main() {
	game.health = 1;
	return;
}
`

	linked := mustLinkProgram(t, script, 8, 0)
	vm := NewVM(testFrameCapacityBytes)
	vm.BindExternBlock(2, make([]byte, 4))
	if status := vm.LoadProgram(linked); status != VMStatusInvalidImage {
		t.Fatalf("expected too-small bound bank to fail load, got %s", status)
	}
	vm.BindExternBlock(2, make([]byte, 8))
	if status := vm.LoadProgram(linked); status != VMStatusOK {
		t.Fatalf("expected sufficient bound bank to load, got %s", status)
	}
}

func TestLinkRejectsBankCapacityOverflow(t *testing.T) {
	script := `
struct game_state_t {
	int health;
	int score;
};

extern(1) game_state_t game;

void script_main() {
	return;
}
`

	ctx := NewContext()
	tokens := mustTokenize(t, ctx, script)
	program := mustParseTokens(t, ctx, tokens)
	compiled, ok := NewCompiler(ctx).Compile(program)
	if !ok {
		t.Fatalf("Compile failed")
	}
	// Capacity is sufficient for bank 0 but the struct lives in bank 1.
	if _, success := NewLinker(ctx, [ExternMemoryBankCount]int{0: 8, 1: 4}, 0).Link(program, compiled); success {
		t.Fatal("expected link failure for over-capacity memory bank")
	}
}
