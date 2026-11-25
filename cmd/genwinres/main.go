// Command genwinres writes an AMD64 COFF object that embeds a Windows icon
// for the Go linker. It has no third-party dependencies.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
)

const (
	rtIcon      = 3
	rtGroupIcon = 14
	langENUS    = 0x0409
)

func main() {
	ico := flag.String("ico", "", "input multi-resolution .ico")
	out := flag.String("out", "", "output AMD64 Windows COFF .syso")
	flag.Parse()
	if *ico == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: genwinres -ico <path> -out <path>")
		os.Exit(2)
	}
	if err := writeSyso(*ico, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeSyso(icoPath, outPath string) error {
	b, err := buildSyso(icoPath)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0o644)
}

func buildSyso(icoPath string) ([]byte, error) {
	src, err := os.ReadFile(icoPath)
	if err != nil {
		return nil, err
	}
	images, group, err := parseICO(src)
	if err != nil {
		return nil, err
	}
	res := map[uint16]map[uint16][]byte{
		rtIcon:      {},
		rtGroupIcon: {1: group},
	}
	for i, img := range images {
		res[rtIcon][uint16(i+1)] = img
	}
	raw, relocs := resourceSection(res, langENUS)
	return coffAMD64(raw, relocs), nil
}

// parseICO reads ICONDIR and each 16-byte ICONDIRENTRY. The returned group
// blob is RT_GROUP_ICON: a 6-byte directory plus one 14-byte GRPICONDIRENTRY
// per frame (the file offset is replaced by the resource id).
func parseICO(src []byte) (images [][]byte, group []byte, err error) {
	if len(src) < 6 {
		return nil, nil, fmt.Errorf("ico: shorter than ICONDIR")
	}
	reserved := binary.LittleEndian.Uint16(src[0:2])
	kind := binary.LittleEndian.Uint16(src[2:4])
	count := int(binary.LittleEndian.Uint16(src[4:6]))
	if reserved != 0 || kind != 1 {
		return nil, nil, fmt.Errorf("ico: reserved=%d type=%d, want icon", reserved, kind)
	}
	if count <= 0 {
		return nil, nil, fmt.Errorf("ico: no frames")
	}
	if len(src) < 6+16*count {
		return nil, nil, fmt.Errorf("ico: directory truncated")
	}
	var g bytes.Buffer
	_ = binary.Write(&g, binary.LittleEndian, [3]uint16{0, 1, uint16(count)})
	images = make([][]byte, count)
	for i := 0; i < count; i++ {
		e := src[6+16*i : 6+16*(i+1)]
		size := binary.LittleEndian.Uint32(e[8:12])
		off := binary.LittleEndian.Uint32(e[12:16])
		if size == 0 || uint64(off)+uint64(size) > uint64(len(src)) {
			return nil, nil, fmt.Errorf("ico: frame %d out of range", i+1)
		}
		images[i] = append([]byte(nil), src[off:off+size]...)
		// Width through dwBytesInRes is 12 bytes. nID replaces dwImageOffset.
		g.Write(e[:12])
		_ = binary.Write(&g, binary.LittleEndian, uint16(i+1))
	}
	return images, g.Bytes(), nil
}

func sortedIDs(m map[uint16][]byte) []uint16 {
	ids := make([]uint16, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedTypes(m map[uint16]map[uint16][]byte) []uint16 {
	ids := make([]uint16, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// resourceSection builds a .rsrc tree: type, id, language, then 8-byte
// aligned payloads. Relocation offsets point at each data entry's RVA field.
func resourceSection(res map[uint16]map[uint16][]byte, lang uint16) ([]byte, []uint32) {
	types := sortedTypes(res)
	dirSize := func(n int) int { return 16 + 8*n }
	size := dirSize(len(types))
	count := 0
	for _, t := range types {
		size += dirSize(len(res[t])) + len(res[t])*dirSize(1)
		count += len(res[t])
	}
	entriesAt := size
	dataAt := entriesAt + 16*count

	raw := make([]byte, dataAt)
	le := binary.LittleEndian
	putDirCount := func(at, n int) {
		le.PutUint16(raw[at+14:], uint16(n))
	}
	var relocs []uint32
	var data bytes.Buffer
	putDirCount(0, len(types))
	next := dirSize(len(types))
	entry := entriesAt
	for i, t := range types {
		le.PutUint32(raw[16+8*i:], uint32(t))
		le.PutUint32(raw[16+8*i+4:], uint32(next)|0x80000000)
		ids := sortedIDs(res[t])
		idDir := next
		putDirCount(idDir, len(ids))
		next += dirSize(len(ids))
		for j, id := range ids {
			le.PutUint32(raw[idDir+16+8*j:], uint32(id))
			le.PutUint32(raw[idDir+16+8*j+4:], uint32(next)|0x80000000)
			putDirCount(next, 1)
			le.PutUint32(raw[next+16:], uint32(lang))
			le.PutUint32(raw[next+20:], uint32(entry))
			next += dirSize(1)

			for data.Len()%8 != 0 {
				data.WriteByte(0)
			}
			le.PutUint32(raw[entry:], uint32(dataAt+data.Len()))
			le.PutUint32(raw[entry+4:], uint32(len(res[t][id])))
			relocs = append(relocs, uint32(entry))
			entry += 16
			data.Write(res[t][id])
		}
	}
	raw = append(raw, data.Bytes()...)
	for len(raw)%4 != 0 {
		raw = append(raw, 0)
	}
	return raw, relocs
}

// coffAMD64 wraps raw .rsrc bytes in an AMD64 COFF object.
// The single symbol is the 18-byte .rsrc section symbol. The string table
// is only its 4-byte length, because the symbol name is stored inline.
// Each relocation is 10 bytes and uses IMAGE_REL_AMD64_ADDR32NB (3),
// symbol 0.
func coffAMD64(raw []byte, relocs []uint32) []byte {
	const (
		headerSize        = 20
		sectionHeaderSize = 40
		machineAMD64      = 0x8664
		relADDR32NB       = 0x0003
		sectionFlags      = 0x40000040
	)
	rawAt := headerSize + sectionHeaderSize
	relocAt := rawAt + len(raw)
	symbolsAt := relocAt + 10*len(relocs)

	var out bytes.Buffer
	le := binary.LittleEndian
	write16 := func(v uint16) { _ = binary.Write(&out, le, v) }
	write32 := func(v uint32) { _ = binary.Write(&out, le, v) }

	write16(machineAMD64)
	write16(1)
	write32(0)
	write32(uint32(symbolsAt))
	write32(1)
	write16(0)
	write16(0)

	out.Write([]byte{'.', 'r', 's', 'r', 'c', 0, 0, 0})
	write32(0)
	write32(0)
	write32(uint32(len(raw)))
	write32(uint32(rawAt))
	write32(uint32(relocAt))
	write32(0)
	write16(uint16(len(relocs)))
	write16(0)
	write32(sectionFlags)

	out.Write(raw)
	for _, r := range relocs {
		write32(r)
		write32(0)
		write16(relADDR32NB)
	}

	out.Write([]byte{'.', 'r', 's', 'r', 'c', 0, 0, 0})
	write32(0)
	write16(1)
	write16(0)
	out.Write([]byte{3, 0})
	write32(4)
	return out.Bytes()
}
