package cova

import (
	"encoding/binary"
	"testing"
)

func TestExternAddressUsesByteOffset(t *testing.T) {
	externMemory := make([]byte, 8)
	binary.LittleEndian.PutUint32(externMemory[4:], 21)

	script := `
extern byte prefix;
extern int value;

void script_main() {
	value = value + 9;
	return;
}
`

	linked := mustLinkProgram(t, script, len(externMemory), 0)
	vm := NewVM(testFrameCapacityBytes)
	vm.BindExternBlock(externMemory)
	if status := vm.Run(linked); status != VMStatusOK {
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

extern home_state_t g_home;

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
	vm.BindExternBlock(externMemory)
	if status := vm.Run(linked); status != VMStatusOK {
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
