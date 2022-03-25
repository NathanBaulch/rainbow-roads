package main

import "image"

func optimizeFrames(ims []*image.Paletted) {
	if len(ims) == 0 {
		return
	}

	buf := image.NewPaletted(ims[0].Rect, ims[0].Palette)
	trans := []uint8{uint8(len(ims[0].Palette) - 1)}
	for i, im := range ims {
		if i == 0 {
			copy(buf.Pix, im.Pix)
		} else {
			var same bool
			var j0, x0, y0 int
			var crop image.Rectangle
			for j := 0; j <= len(im.Pix); j++ {
				if j == 0 {
					same = buf.Pix[j] == im.Pix[j]
				} else if j == len(im.Pix) || (buf.Pix[j] == im.Pix[j]) != same {
					x := j % im.Stride
					y := j / im.Stride
					if same {
						for len(trans) < j-j0 {
							trans = append(trans, trans...)
						}
						copy(im.Pix[j0:j], trans[:j-j0])
					} else {
						copy(buf.Pix[j0:j], im.Pix[j0:j])
						var r image.Rectangle
						if y > y0 {
							r = image.Rect(0, y0, im.Stride, y+1)
						} else {
							r = image.Rect(x0, y0, x, y+1)
						}
						if crop.Empty() {
							crop = r
						} else {
							crop = crop.Union(r)
						}
					}
					same = !same
					j0, x0, y0 = j, x, y
				}
			}
			if crop.Empty() {
				crop = image.Rect(0, 0, 1, 1)
			}
			ims[i] = im.SubImage(crop).(*image.Paletted)
		}
	}
}
