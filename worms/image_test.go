package worms

import (
	"bufio"
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestImageOptimizeFrames(t *testing.T) {
	rect := image.Rect(0, 0, 3, 3)
	pal := color.Palette([]color.Color{color.Black, color.White, color.Transparent})
	ims := make([]*image.Paletted, 7)
	for i := range ims {
		ims[i] = image.NewPaletted(rect, pal)
	}
	ims[2].Pix = []byte{1, 1, 1, 1, 1, 1, 1, 1, 1}
	ims[3].Pix = []byte{1, 1, 1, 1, 0, 1, 1, 1, 1}
	ims[4].Pix = []byte{0, 1, 0, 1, 0, 1, 0, 1, 0}
	ims[5].Pix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0}
	ims[6].Pix = []byte{0, 0, 1, 1, 0, 0, 0, 0, 0}
	optimizeFrames(ims)
	expects := []struct {
		size image.Rectangle
		pix  []uint8
	}{
		{size: image.Rect(0, 0, 3, 3), pix: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{size: image.Rect(0, 0, 1, 1), pix: []byte{2, 2, 2, 2, 2, 2, 2, 2, 2}},
		{size: image.Rect(0, 0, 3, 3), pix: []byte{1, 1, 1, 1, 1, 1, 1, 1, 1}},
		{size: image.Rect(1, 1, 2, 2), pix: []byte{2, 2, 2, 2, 0, 2, 2, 2, 2}},
		{size: image.Rect(0, 0, 3, 3), pix: []byte{0, 2, 0, 2, 2, 2, 0, 2, 0}},
		{size: image.Rect(0, 0, 3, 3), pix: []byte{2, 0, 2, 0, 2, 0, 2, 0, 2}},
		{size: image.Rect(0, 0, 3, 2), pix: []byte{2, 2, 1, 1, 2, 2, 2, 2, 2}},
	}
	for i, expect := range expects {
		if ims[i].Rect != expect.size {
			t.Fatal("unexpected frame", i, "size: ", ims[i].Rect, "!=", expect.size)
		}
		pix := make([]uint8, -ims[i].PixOffset(0, 0))
		for j := range pix {
			pix[j] = 2
		}
		pix = append(pix, ims[i].Pix...)
		if !bytes.Equal(pix, expect.pix) {
			t.Fatal("unexpected frame", i, "pixels: ", pix, "!=", expect.pix)
		}
	}
}

func TestImageGifWriter(t *testing.T) {
	b := &bytes.Buffer{}
	w := &gifWriter{Writer: bufio.NewWriter(b), Comment: "foo"}
	if n, err := w.Write([]byte{0x21, 0xff, 0x0b}); err != nil {
		t.Fatal(err)
	} else if n != 10 {
		t.Fatal("number of bytes written:", n, "!=", 10)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b.Bytes(), []byte("foo")) {
		t.Fatal("metadata text not found")
	}
}

func TestImagePngWriter(t *testing.T) {
	b := &bytes.Buffer{}
	w := &pngWriter{Writer: b, Text: "foo"}
	if n, err := w.Write([]byte("    IDAT")); err != nil {
		t.Fatal(err)
	} else if n != 23 {
		t.Fatal("number of bytes written:", n, "!=", 23)
	}
	if !bytes.Contains(b.Bytes(), []byte("foo")) {
		t.Fatal("metadata text not found")
	}
}
