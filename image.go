package main

import (
	"bufio"
	"encoding/binary"
	"hash/crc32"
	"image"
	"io"
)

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

type gifWriter struct {
	*bufio.Writer
	done bool
}

func (w *gifWriter) Write(p []byte) (nn int, err error) {
	var n = 0
	if !w.done {
		// intercept application extension
		if len(p) == 3 && p[0] == 0x21 && p[1] == 0xff && p[2] == 0x0b {
			if n, err = w.writeExtension([]byte(fullTitle), 0xfe); err != nil {
				return
			} else {
				nn += n
			}
			w.done = true
		}
	}
	if n, err = w.Writer.Write(p); err != nil {
		return
	} else {
		nn += n
	}
	return
}

func (w *gifWriter) writeExtension(b []byte, e byte) (nn int, err error) {
	var n = 0
	if n, err = w.Writer.Write([]byte{0x21, e, byte(len(b))}); err != nil {
		return
	} else {
		nn += n
	}
	if n, err = w.Writer.Write(b); err != nil {
		return
	} else {
		nn += n
	}
	if err = w.Writer.WriteByte(0); err != nil {
		return
	} else {
		nn++
	}
	return
}

type pngWriter struct {
	io.Writer
	done bool
}

func (w *pngWriter) Write(p []byte) (nn int, err error) {
	n := 0
	if !w.done {
		// intercept first data chunk
		if len(p) >= 8 && string(p[4:8]) == "IDAT" {
			if n, err = w.writeChunk([]byte(fullTitle), "tEXt"); err != nil {
				return
			} else {
				nn += n
			}
			w.done = true
		}
	}
	if n, err = w.Writer.Write(p); err != nil {
		return
	} else {
		nn += n
	}
	return
}

func (w *pngWriter) writeChunk(b []byte, name string) (nn int, err error) {
	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header, uint32(len(b)))
	copy(header[4:], name)
	crc := crc32.NewIEEE()
	_, _ = crc.Write(header[4:8])
	_, _ = crc.Write(b)
	footer := make([]byte, 4)
	binary.BigEndian.PutUint32(footer, crc.Sum32())

	n := 0
	if n, err = w.Writer.Write(header); err != nil {
		return
	} else {
		nn += n
	}
	if n, err = w.Writer.Write(b); err != nil {
		return
	} else {
		nn += n
	}
	if n, err = w.Writer.Write(footer); err != nil {
		return
	} else {
		nn += n
	}
	return
}
