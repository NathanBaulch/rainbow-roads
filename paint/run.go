package paint

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/NathanBaulch/rainbow-roads/img"
	"github.com/NathanBaulch/rainbow-roads/parse"
	"github.com/NathanBaulch/rainbow-roads/scan"
	"github.com/expr-lang/expr"
	"github.com/fogleman/gg"
	"golang.org/x/image/colornames"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	o          *Options
	fullTitle  string
	en         = message.NewPrinter(language.English)
	files      []*scan.File
	activities []*parse.Activity
	roads      []*way
	im         image.Image

	backCol    = colornames.Black
	donePriCol = colornames.Lime
	doneSecCol = colornames.Green
	pendPriCol = colornames.Red
	pendSecCol = colornames.Darkred
	actCol     = colornames.Blue
	queryExpr  = "is_tag(highway)" +
		" and highway not in ['proposed','corridor','construction','footway','steps','busway','elevator','services']" +
		" and service not in ['driveway','parking_aisle']" +
		" and area != 'yes'"
	primaryExpr = mustCompile(
		"highway in ['cycleway','primary','residential','secondary','tertiary','trunk','living_street','unclassified']"+
			" and access not in ['private','customers','no']"+
			" and surface not in ['cobblestone','sett']", expr.AsBool())
)

type Options struct {
	Title       string
	Version     string
	Input       []string
	Output      string
	Width       uint
	Region      geo.Circle
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
	if filepath.Ext(o.Output) == "" {
		o.Output += ".png"
	}

	for _, step := range []func() error{scanStep, parseStep, fetchStep, renderStep, saveStep} {
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
		stats.Print(en)
		return nil
	}
}

func fetchStep() error {
	query, err := buildQuery(o.Region.Grow(1/0.9), queryExpr)
	if err != nil {
		return err
	}

	roads, err = osmLookup(query)
	return err
}

func renderStep() error {
	oX, oY := o.Region.Origin.MercatorProjection()
	scale := math.Cos(o.Region.Origin.Lat) * 0.9 * float64(o.Width) / (2 * o.Region.Radius)

	drawLine := func(gc *gg.Context, pt geo.Point) {
		x, y := pt.MercatorProjection()
		x = float64(o.Width)/2 + (x-oX)*scale
		y = float64(o.Width)/2 - (y-oY)*scale
		gc.LineTo(x, y)
	}
	drawActs := func(gc *gg.Context, lineWidth float64) {
		gc.SetLineWidth(1.3 * lineWidth * scale)
		for _, a := range activities {
			for _, r := range a.Records {
				drawLine(gc, r.Position)
			}
			gc.Stroke()
		}
	}

	gc := gg.NewContext(int(o.Width), int(o.Width))
	gc.SetFillStyle(gg.NewSolidPattern(backCol))
	gc.DrawRectangle(0, 0, float64(o.Width), float64(o.Width))
	gc.Fill()

	gc.SetStrokeStyle(gg.NewSolidPattern(actCol))
	drawActs(gc, 10)

	drawWays := func(primary bool, strokeColor color.Color) {
		gc.SetStrokeStyle(gg.NewSolidPattern(strokeColor))

		for _, w := range roads {
			env := map[string]string{
				"highway": w.Highway,
				"access":  w.Access,
				"surface": w.Surface,
			}
			if !primary || mustRun(primaryExpr, env).(bool) {
				lineWidth := 10.0
				switch w.Highway {
				case "motorway", "trunk", "primary", "secondary", "tertiary":
					lineWidth *= 3.6
				case "motorway_link", "trunk_link", "primary_link", "secondary_link", "tertiary_link", "residential", "living_street":
					lineWidth *= 2.4
				case "pedestrian", "footway", "cycleway", "track":
					lineWidth *= 1.4
				}
				gc.SetLineWidth(lineWidth * scale)
				for _, pt := range w.Geometry {
					drawLine(gc, pt)
				}
				gc.Stroke()
			}
		}
	}

	maskGC := gg.NewContext(int(o.Width), int(o.Width))
	drawActs(maskGC, 50)
	actMask := maskGC.AsMask()

	_ = gc.SetMask(actMask)
	drawWays(false, doneSecCol)
	gc.InvertMask()
	drawWays(false, pendSecCol)

	_ = maskGC.SetMask(actMask)
	maskGC.SetColor(color.Transparent)
	maskGC.Clear()
	maskGC.SetColor(color.Black)
	maskGC.DrawCircle(float64(o.Width)/2, float64(o.Width)/2, 0.9*float64(o.Width)/2)
	maskGC.Fill()
	_ = gc.SetMask(maskGC.AsMask())
	drawWays(true, pendPriCol)

	maskGC.InvertMask()
	maskGC.SetColor(color.Transparent)
	maskGC.Clear()
	maskGC.SetColor(color.Black)
	maskGC.DrawCircle(float64(o.Width)/2, float64(o.Width)/2, 0.9*float64(o.Width)/2)
	maskGC.Fill()
	_ = gc.SetMask(maskGC.AsMask())
	drawWays(true, donePriCol)

	if !o.NoWatermark {
		img.DrawWatermark(gc.Image(), fullTitle, pendSecCol)
	}

	done, pend := 0, 0
	rect := gc.Image().Bounds()
	for y := rect.Min.Y; y <= rect.Max.Y; y++ {
		for x := rect.Min.X; x <= rect.Max.X; x++ {
			switch gc.Image().At(x, y) {
			case donePriCol:
				done++
			case pendPriCol:
				pend++
			}
		}
	}
	if done == 0 && pend == 0 {
		pend = 1
	}
	en.Printf("progress:      %.2f%%\n", 100*float64(done)/float64(done+pend))

	im = gc.Image()
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

	return png.Encode(out, im)
}
