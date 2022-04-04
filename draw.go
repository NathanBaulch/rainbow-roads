package main

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var grays = make([]color.Color, 0x100)

func init() {
	for i := range grays {
		grays[i] = color.Gray{Y: uint8(i)}
	}
}

func drawFill(im *image.Paletted, ci uint8) {
	if len(im.Pix) > 0 {
		im.Pix[0] = ci
		for i := 1; i < len(im.Pix); i *= 2 {
			copy(im.Pix[i:], im.Pix[:i])
		}
	}
}

func drawString(im *image.Paletted, text string, ci uint8) {
	d := &font.Drawer{
		Dst:  im,
		Src:  image.NewUniform(im.Palette[ci]),
		Face: basicfont.Face7x13,
	}
	b, _ := d.BoundString(text)
	b = b.Sub(b.Min)
	if b.In(fixed.R(0, 0, im.Bounds().Max.X-10, im.Rect.Max.Y-10)) {
		d.Dot = fixed.P(im.Rect.Max.X, im.Rect.Max.Y).
			Sub(b.Max.Sub(fixed.P(0, basicfont.Face7x13.Height))).
			Sub(fixed.P(5, 5))
		d.DrawString(text)
	}
}
