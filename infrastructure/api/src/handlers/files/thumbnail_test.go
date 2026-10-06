package files

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestBuildImageThumbnailRejectsOversizedSource(t *testing.T) {
	data := pngHeader(100_000, 100_000)

	thumb, contentType, ok, err := buildImageThumbnail(bytes.NewReader(data), 256)
	if err == nil || !strings.Contains(err.Error(), "exceed thumbnail safety limit") {
		t.Fatalf("expected dimension limit error, got %v", err)
	}
	if ok || thumb != nil || contentType != "" {
		t.Fatalf("oversized source unexpectedly produced a thumbnail")
	}
}

func TestBuildImageThumbnailDecodesAfterConfigCheck(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, src); err != nil {
		t.Fatal(err)
	}

	thumb, contentType, ok, err := buildImageThumbnail(bytes.NewReader(encoded.Bytes()), 256)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(thumb) == 0 {
		t.Fatal("valid source did not produce a thumbnail")
	}
	if contentType != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", contentType)
	}
	if _, _, err := image.Decode(bytes.NewReader(thumb)); err != nil {
		t.Fatalf("thumbnail is not a valid image: %v", err)
	}
}

func pngHeader(width, height uint32) []byte {
	var data bytes.Buffer
	data.WriteString("\x89PNG\r\n\x1a\n")
	_ = binary.Write(&data, binary.BigEndian, uint32(13))
	data.WriteString("IHDR")

	var ihdr bytes.Buffer
	_ = binary.Write(&ihdr, binary.BigEndian, width)
	_ = binary.Write(&ihdr, binary.BigEndian, height)
	ihdr.Write([]byte{8, 6, 0, 0, 0})
	data.Write(ihdr.Bytes())

	crcInput := append([]byte("IHDR"), ihdr.Bytes()...)
	_ = binary.Write(&data, binary.BigEndian, crc32.ChecksumIEEE(crcInput))
	return data.Bytes()
}
