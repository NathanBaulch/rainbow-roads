package worms

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/NathanBaulch/rainbow-roads/img"
	"github.com/NathanBaulch/rainbow-roads/parse"
	"github.com/NathanBaulch/rainbow-roads/scan"
	"github.com/StephaneBunel/bresenham"
	"github.com/kettek/apng"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/project"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	o          *Options
	fullTitle  string
	en         = message.NewPrinter(language.English)
	files      []*scan.File
	activities []*parse.Activity
	maxDur     time.Duration
	extent     orb.Bound
	images     []*image.Paletted
)

type Options struct {
	Title       string
	Version     string
	Input       []string
	Output      string
	Width       uint
	Frames      uint
	FPS         uint
	Format      string
	Colors      img.ColorGradient
	ColorDepth  uint
	Speed       float64
	Loop        bool
	NoWatermark bool
	Selector    parse.Selector
}

func Run(opts *Options) error {
	o = opts

	fullTitle = "NathanBaulch/" + o.Title
	if o.Version != "" {
		fullTitle += " " + o.Version
	}

	if len(o.Input) == 0 {
		o.Input = []string{"."}
	}

	if fi, err := os.Stat(o.Output); err != nil {
		var perr *fs.PathError
		if !errors.As(err, &perr) {
			return err
		}
	} else if fi.IsDir() {
		o.Output = filepath.Join(o.Output, "out")
	}
	ext := filepath.Ext(o.Output)
	if ext != "" {
		ext = ext[1:]
		if o.Format == "" {
			o.Format = ext[1:]
		}
	}
	if o.Format == "" {
		o.Format = "gif"
	}
	if !strings.EqualFold(ext, o.Format) {
		o.Output += "." + o.Format
	}

	for _, step := range []func() error{scanStep, parseStep, renderStep, saveStep} {
		if err := step(); err != nil {
			return err
		}
	}

	return nil
}

func scanStep() error {
	if f, err := scan.Scan(o.Input); err != nil {
		return err
	} else {
		files = f
		en.Println("files:        ", len(files))
		return nil
	}
}

func parseStep() error {
	if a, stats, err := parse.Parse(files, &o.Selector); err != nil {
		return err
	} else {
		activities = a
		extent = stats.Extent
		maxDur = stats.MaxDuration
		stats.Print(en)
		return nil
	}
}

func renderStep() error {
	if o.Loop {
		sort.Slice(activities, func(i, j int) bool {
			return activities[i].Records[0].Timestamp.Before(activities[j].Records[0].Timestamp)
		})
	}

	proj := project.WGS84.ToMercator
	ext := project.Bound(extent, proj)
	dX, dY := ext.Right()-ext.Left(), ext.Top()-ext.Bottom()
	scale := float64(o.Width) / dX
	height := uint(dY * scale)
	scale *= 0.9
	ext.Min[0] -= 0.05 * dX
	ext.Max[1] += 0.05 * dY
	tScale := 1 / (o.Speed * float64(maxDur))
	for i, act := range activities {
		ts0 := act.Records[0].Timestamp
		tOffset := 0.0
		if o.Loop {
			tOffset = float64(i) / float64(len(activities))
		}
		for _, r := range act.Records {
			p := project.Point(r.Position, proj)
			r.X = int((p.X() - ext.Left()) * scale)
			r.Y = int((ext.Top() - p.Y()) * scale)
			r.Percent = tOffset + float64(r.Timestamp.Sub(ts0))*tScale
		}
	}

	pal := color.Palette(make([]color.Color, 1<<o.ColorDepth))
	for i := 0; i < len(pal)-2; i++ {
		pal[i] = o.Colors.GetColorAt(float64(i) / float64(len(pal)-3))
	}
	pal[len(pal)-2] = color.Black
	pal[len(pal)-1] = color.Transparent

	images = make([]*image.Paletted, o.Frames)
	for i := range images {
		im := image.NewPaletted(image.Rect(0, 0, int(o.Width), int(height)), pal)
		if i == 0 {
			drawFill(im, uint8(len(pal)-2))
			if !o.NoWatermark {
				img.DrawWatermark(im, fullTitle, pal[len(pal)/2])
			}
		} else {
			copy(im.Pix, images[0].Pix)
		}
		images[i] = im
	}

	wg := &sync.WaitGroup{}
	wg.Add(int(o.Frames))
	for f := uint(0); f < o.Frames; f++ {
		go func() {
			fpc := float64(f+1) / float64(o.Frames)
			gp := &glowPlotter{images[f]}
			for _, act := range activities {
				var rPrev *parse.Record
				for _, r := range act.Records {
					pc := fpc - r.Percent
					if pc < 0 {
						if !o.Loop {
							break
						}
						pc++
					}
					if rPrev != nil && (r.X != rPrev.X || r.Y != rPrev.Y) {
						ci := uint8(len(pal) - 3)
						if pc >= 0 && pc < 1 {
							ci = uint8(math.Sqrt(pc) * float64(len(pal)-2))
						}
						bresenham.DrawLine(gp, rPrev.X, rPrev.Y, r.X, r.Y, grays[ci])
					}
					rPrev = r
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	return nil
}

func saveStep() error {
	if dir := filepath.Dir(o.Output); dir != "." {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}

	out, err := os.Create(o.Output)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	switch o.Format {
	case "gif":
		return saveGIF(out)
	case "png":
		return savePNG(out)
	case "zip":
		return saveZIP(out)
	default:
		return nil
	}
}

func saveGIF(w io.Writer) error {
	optimizeFrames(images)
	g := &gif.GIF{
		Image:    images,
		Delay:    make([]int, len(images)),
		Disposal: make([]byte, len(images)),
		Config: image.Config{
			ColorModel: images[0].Palette,
			Width:      images[0].Rect.Max.X,
			Height:     images[0].Rect.Max.Y,
		},
	}
	d := int(math.Round(100 / float64(o.FPS)))
	for i := range images {
		g.Disposal[i] = gif.DisposalNone
		g.Delay[i] = d
	}
	return gif.EncodeAll(&gifWriter{Writer: bufio.NewWriter(w), Comment: fullTitle}, g)
}

func savePNG(w io.Writer) error {
	optimizeFrames(images)
	a := apng.APNG{Frames: make([]apng.Frame, len(images))}
	for i, im := range images {
		a.Frames[i].Image = im
		a.Frames[i].XOffset = im.Rect.Min.X
		a.Frames[i].YOffset = im.Rect.Min.Y
		a.Frames[i].BlendOp = apng.BLEND_OP_OVER
		a.Frames[i].DelayNumerator = 1
		a.Frames[i].DelayDenominator = uint16(o.FPS)
	}
	return apng.Encode(&pngWriter{Writer: w, Text: fullTitle}, a)
}

func saveZIP(w io.Writer) error {
	z := zip.NewWriter(w)
	defer func() {
		if err := z.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	for i, im := range images {
		if w, err := z.Create(fmt.Sprintf("%d.gif", i)); err != nil {
			return err
		} else if err = gif.Encode(w, im, nil); err != nil {
			return err
		}
	}
	return nil
}
