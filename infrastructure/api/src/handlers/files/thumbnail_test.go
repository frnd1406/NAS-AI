package files

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestBuildImageThumbnail_ResizesNormalImage(t *testing.T) {
	src := encodePNG(t, 800, 600)

	data, contentType, ok, err := buildImageThumbnail(bytes.NewReader(src), 256)
	if err != nil || !ok {
		t.Fatalf("buildImageThumbnail: ok=%v err=%v", ok, err)
	}
	if contentType != "image/jpeg" {
		t.Errorf("contentType = %q, want image/jpeg", contentType)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}
	if cfg.Width > 256 || cfg.Height > 256 {
		t.Errorf("thumbnail %dx%d exceeds 256", cfg.Width, cfg.Height)
	}
}

// A decompression bomb declares huge dimensions in a tiny file. It must be
// rejected from the header alone — decoding it would allocate gigabytes.
func TestBuildImageThumbnail_RejectsOversizedSource(t *testing.T) {
	// 40000x40000 = 1.6 G pixels, but the encoded PNG stays small.
	bomb := encodePNG(t, 1, 1)
	// Patch the IHDR width/height to 40000x40000 (bytes 16..23 of a PNG).
	if len(bomb) < 24 {
		t.Fatal("unexpected png layout")
	}
	huge := []byte{0x00, 0x00, 0x9c, 0x40} // 40000
	copy(bomb[16:20], huge)
	copy(bomb[20:24], huge)

	_, _, ok, err := buildImageThumbnail(bytes.NewReader(bomb), 256)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("oversized source was accepted; expected ok=false")
	}
}

func TestParseThumbMax_Clamps(t *testing.T) {
	cases := map[string]int{
		"":        256,
		"garbage": 256,
		"256":     256,
		"1":       minThumbMaxPixel,
		"-9000":   minThumbMaxPixel,
		"100000":  maxThumbMaxPixel,
	}
	for raw, want := range cases {
		if got := parseThumbMax(raw); got != want {
			t.Errorf("parseThumbMax(%q) = %d, want %d", raw, got, want)
		}
	}
}
