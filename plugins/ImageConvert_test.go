package plugins

import (
	"encoding/binary"
	"testing"
)

func avifHeader(brand string, compatible ...string) []byte {
	size := 16 + 4*len(compatible)
	header := make([]byte, size)
	binary.BigEndian.PutUint32(header[0:4], uint32(size))
	copy(header[4:8], "ftyp")
	copy(header[8:12], brand)
	for i, b := range compatible {
		copy(header[16+i*4:20+i*4], b)
	}
	return header
}

func TestIsAVIFHeaderAcceptsAVIFBrands(t *testing.T) {
	if !isAVIFHeader(avifHeader("avif")) {
		t.Fatal("avif brand should be accepted")
	}
	if !isAVIFHeader(avifHeader("avis")) {
		t.Fatal("avis brand should be accepted")
	}
	if !isAVIFHeader(avifHeader("mif1", "avif")) {
		t.Fatal("mif1 with avif compatible brand should be accepted")
	}
}

func TestIsAVIFHeaderRejectsJPEGAndPNG(t *testing.T) {
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01}
	if isAVIFHeader(jpeg) {
		t.Fatal("jpeg should be rejected")
	}
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}
	if isAVIFHeader(png) {
		t.Fatal("png should be rejected")
	}
}
