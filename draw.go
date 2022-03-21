package main

import (
	"image"
	"image/color"

	"github.com/StephaneBunel/bresenham"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func drawLine(im *image.Paletted, ci uint8, x1, y1, x2, y2 int) {
	setPixIfLower := func(x, y int, ci uint8) bool {
		if (image.Point{X: x, Y: y}.In(im.Rect)) {
			i := im.PixOffset(x, y)
			if im.Pix[i] > ci {
				im.Pix[i] = ci
				return true
			}
		}
		return false
	}
	setPix := func(x, y int, c color.Color) {
		ci := c.(color.Gray).Y
		if !setPixIfLower(x, y, ci) {
			return
		}
		const sqrt2 = 1.414213562
		if i := float64(ci) * sqrt2; i < float64(len(im.Palette)) {
			ci = uint8(i)
			setPixIfLower(x-1, y, ci)
			setPixIfLower(x, y-1, ci)
			setPixIfLower(x+1, y, ci)
			setPixIfLower(x, y+1, ci)
		}
		if i := float64(ci) * sqrt2; i < float64(len(im.Palette)) {
			ci = uint8(i)
			setPixIfLower(x-1, y-1, ci)
			setPixIfLower(x-1, y+1, ci)
			setPixIfLower(x+1, y-1, ci)
			setPixIfLower(x+1, y+1, ci)
		}
	}
	bresenham.Bresenham(plotterFunc(setPix), x1, y1, x2, y2, color.Gray{Y: ci})
}

type plotterFunc func(x, y int, c color.Color)

func (f plotterFunc) Set(x, y int, c color.Color) {
	f(x, y, c)
}

func drawString(im *image.Paletted, ci uint8, text string) {
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
