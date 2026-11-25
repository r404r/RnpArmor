package icons

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"testing"
)

func TestAppIconPNG(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(AppIconPNG))
	if err != nil {
		t.Fatalf("decode AppIconPNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 256 || b.Dy() != 256 {
		t.Fatalf("size = %dx%d, want 256x256", b.Dx(), b.Dy())
	}
	corners := [][2]int{
		{b.Min.X, b.Min.Y},
		{b.Max.X - 1, b.Min.Y},
		{b.Min.X, b.Max.Y - 1},
		{b.Max.X - 1, b.Max.Y - 1},
	}
	for _, p := range corners {
		_, _, _, a := img.At(p[0], p[1]).RGBA()
		if a != 0 {
			t.Fatalf("corner (%d,%d) alpha = %d, want 0", p[0], p[1], a)
		}
	}
	_, _, _, a := img.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2).RGBA()
	if a != 0xffff {
		t.Fatalf("center alpha = %d, want opaque", a)
	}
}

func TestICOFrameHeaders(t *testing.T) {
	b, err := os.ReadFile("rnparmor.ico")
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 6 {
		t.Fatal("ico shorter than ICONDIR")
	}
	if binary.LittleEndian.Uint16(b[0:2]) != 0 || binary.LittleEndian.Uint16(b[2:4]) != 1 {
		t.Fatalf("ICONDIR reserved/type = %d/%d, want 0/1", binary.LittleEndian.Uint16(b[0:2]), binary.LittleEndian.Uint16(b[2:4]))
	}
	count := int(binary.LittleEndian.Uint16(b[4:6]))
	if count != 7 {
		t.Fatalf("frame count = %d, want 7", count)
	}
	if len(b) < 6+16*count {
		t.Fatal("ico directory truncated")
	}
	for i := 0; i < count; i++ {
		e := b[6+16*i : 6+16*(i+1)]
		planes := binary.LittleEndian.Uint16(e[4:6])
		bits := binary.LittleEndian.Uint16(e[6:8])
		size := binary.LittleEndian.Uint32(e[8:12])
		off := binary.LittleEndian.Uint32(e[12:16])
		if planes == 0 || bits == 0 || size == 0 {
			t.Fatalf("frame %d header planes=%d bits=%d bytes=%d", i+1, planes, bits, size)
		}
		if uint64(off)+uint64(size) > uint64(len(b)) {
			t.Fatalf("frame %d image out of range off=%d size=%d len=%d", i+1, off, size, len(b))
		}
	}
}

func TestICNSMagicAndLength(t *testing.T) {
	b, err := os.ReadFile("RnpArmor.icns")
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 8 {
		t.Fatal("icns shorter than header")
	}
	if string(b[:4]) != "icns" {
		t.Fatalf("magic = %q, want icns", b[:4])
	}
	total := binary.BigEndian.Uint32(b[4:8])
	if int(total) != len(b) {
		t.Fatalf("icns length field = %d, file = %d", total, len(b))
	}
}
