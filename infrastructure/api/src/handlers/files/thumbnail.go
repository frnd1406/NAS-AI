package files

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"strconv"
	"strings"

	_ "image/gif"
	_ "image/png"
)

// Decoding happens before resizing, so bound source dimensions to prevent a
// tiny compressed image from forcing a multi-gigabyte allocation.
const maxThumbnailSourcePixels = 50_000_000

// Gallery requests arrive in bursts; serialize the memory-heavy decode step so
// several otherwise valid high-resolution images cannot exhaust memory together.
var thumbnailDecodeSlot = make(chan struct{}, 1)

// buildImageThumbnail decodes a common image format and returns a JPEG thumbnail.
// Returns ok=false when the format cannot be decoded (caller should fall back to full file).
func buildImageThumbnail(r io.Reader, maxPixel int) (data []byte, contentType string, ok bool, err error) {
	if maxPixel < 32 {
		maxPixel = 32
	}
	if maxPixel > 1024 {
		maxPixel = 1024
	}

	var header bytes.Buffer
	cfg, _, err := image.DecodeConfig(io.TeeReader(r, &header))
	if err != nil {
		return nil, "", false, nil
	}
	if cfg.Width <= 0 || cfg.Height <= 0 ||
		cfg.Width > maxThumbnailSourcePixels/cfg.Height {
		return nil, "", false, fmt.Errorf(
			"image dimensions %dx%d exceed thumbnail safety limit",
			cfg.Width,
			cfg.Height,
		)
	}

	thumbnailDecodeSlot <- struct{}{}
	defer func() { <-thumbnailDecodeSlot }()

	src, _, err := image.Decode(io.MultiReader(bytes.NewReader(header.Bytes()), r))
	if err != nil {
		return nil, "", false, nil
	}

	dst := resizeMax(src, maxPixel)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 72}); err != nil {
		return nil, "", false, fmt.Errorf("jpeg encode: %w", err)
	}
	return buf.Bytes(), "image/jpeg", true, nil
}

func resizeMax(src image.Image, maxPixel int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return src
	}
	if w <= maxPixel && h <= maxPixel {
		return src
	}
	scale := math.Min(float64(maxPixel)/float64(w), float64(maxPixel)/float64(h))
	nw := int(math.Max(1, math.Round(float64(w)*scale)))
	nh := int(math.Max(1, math.Round(float64(h)*scale)))

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := b.Min.Y + y*h/nh
		for x := 0; x < nw; x++ {
			sx := b.Min.X + x*w/nw
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func parseThumbMax(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 256
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 256
	}
	return n
}
