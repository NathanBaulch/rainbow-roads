package img

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func DrawWatermark(im image.Image, text string, c color.Color) {
	d := &font.Drawer{
		Dst:  im.(draw.Image),
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
	}
	b, _ := d.BoundString(text)
	b = b.Sub(b.Min)
	if b.In(fixed.R(0, 0, im.Bounds().Max.X-10, im.Bounds().Max.Y-10)) {
		d.Dot = fixed.P(im.Bounds().Max.X, im.Bounds().Max.Y).
			Sub(b.Max.Sub(fixed.P(0, basicfont.Face7x13.Height))).
			Sub(fixed.P(5, 5))
		d.DrawString(text)
	}
}
