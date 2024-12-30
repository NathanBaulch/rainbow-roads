package worms

import (
	"bufio"
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageOptimizeFrames(t *testing.T) {
	is := require.New(t)

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
		is.Equal(expect.size, ims[i].Rect)
		pix := make([]uint8, -ims[i].PixOffset(0, 0))
		for j := range pix {
			pix[j] = 2
		}
		pix = append(pix, ims[i].Pix...)
		is.Equal(expect.pix, pix)
	}
}

func TestImageGifWriter(t *testing.T) {
	is := require.New(t)

	b := &bytes.Buffer{}
	w := &gifWriter{Writer: bufio.NewWriter(b), Comment: "foo"}
	n, err := w.Write([]byte{0x21, 0xff, 0x0b})
	is.NoError(err)
	is.Equal(10, n)
	is.NoError(w.Flush())
	is.True(bytes.Contains(b.Bytes(), []byte("foo")))
}

func TestImagePngWriter(t *testing.T) {
	is := require.New(t)

	b := &bytes.Buffer{}
	w := &pngWriter{Writer: b, Text: "foo"}
	n, err := w.Write([]byte("    IDAT"))
	is.NoError(err)
	is.Equal(23, n)
	is.True(bytes.Contains(b.Bytes(), []byte("foo")))
}
