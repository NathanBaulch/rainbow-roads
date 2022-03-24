package main

import (
	"image"
	"image/color"

	"github.com/StephaneBunel/bresenham"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func drawFill(im *image.Paletted, ci uint8) {
	if len(im.Pix) > 0 {
		im.Pix[0] = ci
		for i := 1; i < len(im.Pix); i *= 2 {
			copy(im.Pix[i:], im.Pix[:i])
		}
	}
}

var grays = make([]color.Color, 0x100)

func drawLine(p bresenham.Plotter, x1, y1, x2, y2 int, ci uint8) {
	c := grays[ci]
	if c == nil {
		c = color.Gray{Y: ci}
		grays[ci] = c
	}
	bresenham.Bresenham(p, x1, y1, x2, y2, c)
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
