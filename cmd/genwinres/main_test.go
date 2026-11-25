package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSysoFromRepoIcon(t *testing.T) {
	out := filepath.Join(t.TempDir(), "rsrc_windows_amd64.syso")
	if err := writeSyso("../../assets/icons/rnparmor.ico", out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 60 {
		t.Fatalf("syso length %d, want at least a COFF header and section", len(b))
	}
	if got := binary.LittleEndian.Uint16(b[0:2]); got != 0x8664 {
		t.Fatalf("machine = %#x, want AMD64 0x8664", got)
	}
	if got := binary.LittleEndian.Uint16(b[2:4]); got != 1 {
		t.Fatalf("sections = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint32(b[12:16]); got != 1 {
		t.Fatalf("symbols = %d, want 1", got)
	}
	symAt := binary.LittleEndian.Uint32(b[8:12])
	name := b[20:28]
	if string(name) != ".rsrc\x00\x00\x00" {
		t.Fatalf("section name = %q", name)
	}
	rawSize := binary.LittleEndian.Uint32(b[36:40])
	rawAt := binary.LittleEndian.Uint32(b[40:44])
	relocAt := binary.LittleEndian.Uint32(b[44:48])
	nreloc := int(binary.LittleEndian.Uint16(b[52:54]))
	chars := binary.LittleEndian.Uint32(b[56:60])
	if rawAt != 60 {
		t.Fatalf("raw pointer = %d, want 60", rawAt)
	}
	if chars != 0x40000040 {
		t.Fatalf("section characteristics = %#x, want 0x40000040", chars)
	}
	if relocAt != rawAt+rawSize {
		t.Fatalf("reloc pointer = %d, raw ends at %d", relocAt, rawAt+rawSize)
	}
	if symAt != relocAt+uint32(10*nreloc) {
		t.Fatalf("symbol pointer = %d, relocs end at %d", symAt, relocAt+uint32(10*nreloc))
	}
	if int(symAt)+18+4 != len(b) {
		t.Fatalf("file length = %d, symbol+string table ends at %d", len(b), int(symAt)+22)
	}
	if string(b[symAt:symAt+8]) != ".rsrc\x00\x00\x00" {
		t.Fatalf("symbol name = %q", b[symAt:symAt+8])
	}
	if binary.LittleEndian.Uint16(b[symAt+12:symAt+14]) != 1 || b[symAt+16] != 3 || b[symAt+17] != 0 {
		t.Fatalf("symbol section/class/aux = %d/%d/%d", binary.LittleEndian.Uint16(b[symAt+12:symAt+14]), b[symAt+16], b[symAt+17])
	}
	if binary.LittleEndian.Uint32(b[symAt+18:symAt+22]) != 4 {
		t.Fatalf("string table length = %d, want 4", binary.LittleEndian.Uint32(b[symAt+18:symAt+22]))
	}
	if nreloc != 8 {
		t.Fatalf("relocations = %d, want 8 (7 icons + group)", nreloc)
	}
	raw := b[rawAt : rawAt+rawSize]
	for i := 0; i < nreloc; i++ {
		at := int(relocAt) + 10*i
		sym := binary.LittleEndian.Uint32(b[at+4 : at+8])
		kind := binary.LittleEndian.Uint16(b[at+8 : at+10])
		if sym != 0 || kind != 0x0003 {
			t.Fatalf("reloc %d symbol=%d type=%#x, want 0 and IMAGE_REL_AMD64_ADDR32NB", i, sym, kind)
		}
	}
	group := resourcePayload(t, raw, rtGroupIcon, 1)
	if len(group) != 6+14*7 {
		t.Fatalf("RT_GROUP_ICON length = %d, want %d", len(group), 6+14*7)
	}
	if binary.LittleEndian.Uint16(group[0:2]) != 0 || binary.LittleEndian.Uint16(group[2:4]) != 1 {
		t.Fatalf("group reserved/type = %d/%d", binary.LittleEndian.Uint16(group[0:2]), binary.LittleEndian.Uint16(group[2:4]))
	}
	if got := binary.LittleEndian.Uint16(group[4:6]); got != 7 {
		t.Fatalf("RT_GROUP_ICON frames = %d, want 7", got)
	}
	for i := 0; i < 7; i++ {
		e := group[6+14*i : 6+14*(i+1)]
		id := binary.LittleEndian.Uint16(e[12:14])
		size := binary.LittleEndian.Uint32(e[8:12])
		if id != uint16(i+1) || size == 0 {
			t.Fatalf("group entry %d id=%d bytes=%d", i+1, id, size)
		}
	}
}

func resourcePayload(t *testing.T, raw []byte, wantType, wantID uint16) []byte {
	t.Helper()
	if len(raw) < 16 {
		t.Fatal("resource section too small")
	}
	nTypes := int(binary.LittleEndian.Uint16(raw[14:16]))
	var typeOff uint32
	for i := 0; i < nTypes; i++ {
		id := binary.LittleEndian.Uint16(raw[16+8*i : 18+8*i])
		off := binary.LittleEndian.Uint32(raw[20+8*i : 24+8*i])
		if id == wantType {
			typeOff = off
			break
		}
	}
	if typeOff&0x80000000 == 0 {
		t.Fatalf("resource type %d not found", wantType)
	}
	idDir := int(typeOff & 0x7fffffff)
	nIDs := int(binary.LittleEndian.Uint16(raw[idDir+14 : idDir+16]))
	var idOff uint32
	for i := 0; i < nIDs; i++ {
		id := binary.LittleEndian.Uint16(raw[idDir+16+8*i : idDir+18+8*i])
		off := binary.LittleEndian.Uint32(raw[idDir+20+8*i : idDir+24+8*i])
		if id == wantID {
			idOff = off
			break
		}
	}
	if idOff&0x80000000 == 0 {
		t.Fatalf("resource id %d not found", wantID)
	}
	langDir := int(idOff & 0x7fffffff)
	if lang := binary.LittleEndian.Uint32(raw[langDir+16 : langDir+20]); lang != langENUS {
		t.Fatalf("language = %#x, want %#x", lang, langENUS)
	}
	langOff := binary.LittleEndian.Uint32(raw[langDir+20 : langDir+24])
	if langOff&0x80000000 != 0 {
		t.Fatal("language entry is not a data entry")
	}
	dataAt := int(langOff)
	payloadAt := int(binary.LittleEndian.Uint32(raw[dataAt : dataAt+4]))
	size := int(binary.LittleEndian.Uint32(raw[dataAt+4 : dataAt+8]))
	if payloadAt < 0 || size < 0 || payloadAt+size > len(raw) {
		t.Fatalf("payload out of range at=%d size=%d", payloadAt, size)
	}
	return raw[payloadAt : payloadAt+size]
}
