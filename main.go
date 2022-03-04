package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/StephaneBunel/bresenham"
	"github.com/kettek/apng"
	"github.com/schollz/progressbar"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	Version string

	input       []string
	output      string
	frames      uint
	width       uint
	height      uint
	format      = NewFormatFlag("gif", "png", "zip")
	colors      ColorsFlag
	noWatermark bool

	sports        SportsFlag
	after         DateFlag
	before        DateFlag
	minDuration   DurationFlag
	maxDuration   DurationFlag
	minDistance   DistanceFlag
	maxDistance   DistanceFlag
	startsNear    RegionFlag
	endsNear      RegionFlag
	passesThrough RegionFlag
	boundedBy     RegionFlag

	files          []*file
	activities     []*activity
	maxDur         time.Duration
	minLat, minLon = math.MaxFloat64, math.MaxFloat64
	maxLat, maxLon = -math.MaxFloat64, -math.MaxFloat64
	ims            []*image.Paletted
)

func init() {
	_ = colors.Set("#fff,#ff8,#911,#414,#007@.5,#001")

	flag.StringVar(&output, "output", "out", "optional path of the generated file")
	flag.UintVar(&frames, "frames", 100, "number of animation frames")
	flag.UintVar(&width, "width", 500, "width of the generated image in pixels")
	flag.Var(&format, "format", "output file format `string`, supports gif, png, zip")
	flag.Var(&colors, "colors", "CSS linear-colors inspired color scheme `string`, eg red,yellow,green,blue,black")
	flag.BoolVar(&noWatermark, "no_watermark", false, "suppress the embedded project name and version string")
	flag.Var(&sports, "sport", "sports to include, can be specified multiple times, eg running, cycling")
	flag.Var(&after, "after", "`date` from which activities should be included")
	flag.Var(&before, "before", "`date` prior to which activities should be included")
	flag.Var(&minDuration, "min_duration", "shortest `duration` of included activities, eg 15m")
	flag.Var(&maxDuration, "max_duration", "longest `duration` of included activities, eg 1h")
	flag.Var(&minDistance, "min_distance", "shortest `distance` of included activities, eg 2km")
	flag.Var(&maxDistance, "max_distance", "greatest `distance` of included activities, eg 10mi")
	flag.Var(&startsNear, "starts_near", "`region` that activities must start from, eg 51.53,-0.21,1km")
	flag.Var(&endsNear, "ends_near", "`region` that activities must end in, eg 30.06,31.22,1km")
	flag.Var(&passesThrough, "passes_through", "`region` that activities must pass through, eg 40.69,-74.12,10mi")
	flag.Var(&boundedBy, "bounded_by", "`region` that activities must be fully contained within, eg -37.8,144.9,10km")
}

func main() {
	flag.Usage = func() {
		header := "Usage of rainbow-roads"
		if Version != "" {
			header += " " + Version
		}
		fmt.Fprintln(flag.CommandLine.Output(), header+":")
		flag.PrintDefaults()
	}
	flag.Parse()
	input = flag.Args()
	if len(input) == 0 {
		input = []string{"."}
	}

	if fi, err := os.Stat(output); err != nil {
		if _, ok := err.(*fs.PathError); !ok {
			log.Fatalln(err)
		}
	} else if fi.IsDir() {
		output = filepath.Join(output, "out")
	}

	ext := filepath.Ext(output)
	if ext != "" {
		ext = ext[1:]
		if format.IsZero() {
			_ = format.Set(ext)
		}
	}
	if format.IsZero() {
		_ = format.Set("gif")
	}
	if !strings.EqualFold(ext, format.String()) {
		output += "." + format.String()
	}

	for _, step := range []func() error{scan, parse, render, save} {
		if err := step(); err != nil {
			log.Fatalln(err)
		}
	}
}

func scan() error {
	for _, in := range input {
		paths := []string{in}
		if strings.ContainsAny(in, "*?[") {
			var err error
			if paths, err = filepath.Glob(in); err != nil {
				if err == filepath.ErrBadPattern {
					return errors.New(fmt.Sprintf("input path pattern %q malformed", in))
				}
				return err
			}
		}

		for _, path := range paths {
			dir, name := filepath.Split(path)
			if dir == "" {
				dir = "."
			}
			fsys := os.DirFS(dir)
			if fi, err := os.Stat(path); err != nil {
				if _, ok := err.(*fs.PathError); ok {
					return errors.New(fmt.Sprintf("input path %q not found", path))
				}
				return err
			} else if fi.IsDir() {
				err := fs.WalkDir(fsys, name, func(path string, d fs.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return err
					} else {
						return scanFile(fsys, path)
					}
				})
				if err != nil {
					return err
				}
			} else if err := scanFile(fsys, name); err != nil {
				return err
			}
		}
	}

	p := message.NewPrinter(language.English)
	p.Println("activity files:", len(files))
	return nil
}

func scanFile(fsys fs.FS, path string) error {
	ext := filepath.Ext(path)
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		if f, err := fsys.Open(path); err != nil {
			return err
		} else if s, err := f.Stat(); err != nil {
			return err
		} else {
			r, ok := f.(io.ReaderAt)
			if !ok {
				if b, err := ioutil.ReadAll(f); err != nil {
					return err
				} else {
					r = bytes.NewReader(b)
				}
			}
			if fsys, err := zip.NewReader(r, s.Size()); err != nil {
				return err
			} else {
				return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return err
					} else {
						return scanFile(fsys, path)
					}
				})
			}
		}
	} else {
		gz := strings.EqualFold(ext, ".gz")
		if gz {
			ext = filepath.Ext(path[:len(path)-3])
		}
		var parser func(io.Reader) error
		if strings.EqualFold(ext, ".fit") {
			parser = parseFIT
		} else if strings.EqualFold(ext, ".gpx") {
			parser = parseGPX
		} else if strings.EqualFold(ext, ".tcx") {
			parser = parseTCX
		} else {
			return nil
		}
		parse := func() (err error) {
			var r io.Reader
			if r, err = fsys.Open(path); err != nil {
				return
			} else if gz {
				if r, err = gzip.NewReader(r); err != nil {
					return
				}
			}
			return parser(r)
		}
		files = append(files, &file{path, parse})
		return nil
	}
}

type file struct {
	path  string
	parse func() error
}

func parse() error {
	activities = make([]*activity, 0, len(files))
	pb := progressbar.New(len(files))
	var warnings []string
	for _, f := range files {
		_ = pb.Add(1)
		if err := f.parse(); err != nil {
			warnings = append(warnings, fmt.Sprint(f.path, ":", err))
		}
	}
	fmt.Println()
	for _, warning := range warnings {
		fmt.Println("WARN:", warning)
	}

	if len(activities) == 0 {
		return errors.New("no matching activities found")
	}

	sportStats := make(map[string]int)
	minDur := time.Duration(math.MaxInt64)
	var minDate, maxDate time.Time
	minDist, maxDist := math.MaxFloat64, 0.0
	sumPts := 0
	var sumDur time.Duration
	sumDist := 0.0
	for i := len(activities) - 1; i >= 0; i-- {
		act := activities[i]
		include := passesThrough.IsZero()
		exclude := false
		for j, r := range act.records {
			if j == 0 && !startsNear.IsZero() && !startsNear.Contains(r.lat, r.lon) {
				exclude = true
				break
			}
			if j == len(act.records)-1 && !endsNear.IsZero() && !endsNear.Contains(r.lat, r.lon) {
				exclude = true
				break
			}
			if !boundedBy.IsZero() && !boundedBy.Contains(r.lat, r.lon) {
				exclude = true
				break
			}
			if !include && passesThrough.Contains(r.lat, r.lon) {
				include = true
			}
		}
		if exclude || !include {
			j := len(activities) - 1
			activities[i] = activities[j]
			activities = activities[:j]
			continue
		}

		if act.sport == "" {
			sportStats["unknown"]++
		} else {
			sportStats[strings.ToLower(act.sport)]++
		}
		if minDate.IsZero() || act.date.Before(minDate) {
			minDate = act.date
		}
		if maxDate.IsZero() || act.date.After(maxDate) {
			maxDate = act.date
		}
		if act.duration < minDur {
			minDur = act.duration
		}
		if act.duration > maxDur {
			maxDur = act.duration
		}
		if act.distance < minDist {
			minDist = act.distance
		}
		if act.distance > maxDist {
			maxDist = act.distance
		}

		sumPts += len(act.records)
		sumDur += act.duration
		sumDist += act.distance

		for _, r := range act.records {
			minLat, minLon = math.Min(minLat, r.lat), math.Min(minLon, r.lon)
			maxLat, maxLon = math.Max(maxLat, r.lat), math.Max(maxLon, r.lon)
		}
	}

	if len(activities) == 0 {
		return errors.New("no matching activities found")
	}

	lat, lon := (maxLat+minLat)/2, (maxLon+minLon)/2
	radius := 0.0
	for _, act := range activities {
		for _, r := range act.records {
			radius = math.Max(radius, haversineDistance(lat, lon, r.lat, r.lon))
		}
	}

	p := message.NewPrinter(language.English)
	p.Printf("activities:     %d\n", len(activities))
	p.Printf("sports:         %s\n", sprintSportStats(p, sportStats))
	p.Printf("period:         %s\n", sprintPeriod(p, minDate, maxDate))
	p.Printf("duration range: %s to %s\n", minDur, maxDur)
	p.Printf("distance range: %.1fkm to %.1fkm\n", minDist/1000, maxDist/1000)
	p.Printf("bounds:         %s\n", &Region{lat, lon, radius})
	p.Printf("total points:   %d\n", sumPts)
	p.Printf("total duration: %s\n", sumDur)
	p.Printf("total distance: %.1fkm\n", sumDist/1000)
	return nil
}

func sprintSportStats(p *message.Printer, stats map[string]int) string {
	pairs := make([]struct {
		k string
		v int
	}, len(stats))
	i := 0
	for k, v := range stats {
		pairs[i].k = k
		pairs[i].v = v
		i++
	}
	sort.Slice(pairs, func(i, j int) bool {
		p0, p1 := pairs[i], pairs[j]
		return p0.v > p1.v || (p0.v == p1.v && p0.k < p1.k)
	})
	a := make([]interface{}, len(stats)*2)
	i = 0
	for _, kv := range pairs {
		a[i] = kv.k
		i++
		a[i] = kv.v
		i++
	}
	return p.Sprintf(strings.Repeat(", %s (%d)", len(stats))[2:], a...)
}

func sprintPeriod(p *message.Printer, minDate, maxDate time.Time) string {
	d := maxDate.Sub(minDate)
	var num float64
	var unit string
	switch {
	case d.Hours() >= 365.25*24:
		num, unit = d.Hours()/(365.25*24), "years"
	case d.Hours() >= 365.25*2:
		num, unit = d.Hours()/(365.25*2), "months"
	case d.Hours() >= 7*24:
		num, unit = d.Hours()/(7*24), "weeks"
	case d.Hours() >= 24:
		num, unit = d.Hours()/24, "days"
	case d.Hours() >= 1:
		num, unit = d.Hours(), "hours"
	case d.Minutes() >= 1:
		num, unit = d.Minutes(), "minutes"
	default:
		num, unit = d.Seconds(), "seconds"
	}
	return p.Sprintf("%.1f %s (%s to %s)", num, unit, minDate.Format("2006-01-02"), maxDate.Format("2006-01-02"))
}

func includeSport(sport string) bool {
	if len(sports) == 0 {
		return true
	}
	for _, s := range sports {
		if strings.EqualFold(s, sport) {
			return true
		}
	}
	return false
}

func includeDate(date time.Time) bool {
	if !after.Time.IsZero() && after.Time.After(date) {
		return false
	}
	if !before.Time.IsZero() && before.Time.Before(date) {
		return false
	}
	return true
}

func includeDuration(duration time.Duration) bool {
	if min := minDuration.Duration; min != 0 && duration < min {
		return false
	}
	if max := maxDuration.Duration; max != 0 && duration > max {
		return false
	}
	return true
}

func includeDistance(distance float64) bool {
	if minDistance != 0 && distance < float64(minDistance) {
		return false
	}
	if maxDistance != 0 && distance > float64(maxDistance) {
		return false
	}
	return true
}

type activity struct {
	sport    string
	date     time.Time
	duration time.Duration
	distance float64
	records  []*record
}

type record struct {
	ts       time.Time
	lat, lon float64
	x, y     int
	p        float64
}

func render() error {
	minX, minY := mercatorMeters(minLat, minLon)
	maxX, maxY := mercatorMeters(maxLat, maxLon)
	dX, dY := maxX-minX, maxY-minY
	scale := float64(width) / dX
	height = uint(dY * scale)
	scale *= 0.9
	minX -= 0.05 * dX
	maxY += 0.05 * dY
	for _, act := range activities {
		ts0 := act.records[0].ts
		for _, r := range act.records {
			x, y := mercatorMeters(r.lat, r.lon)
			r.x = int((x - minX) * scale)
			r.y = int((maxY - y) * scale)
			r.p = float64(r.ts.Sub(ts0)) / float64(maxDur)
		}
	}

	pal := color.Palette(make([]color.Color, 0x100))
	for i := 0; i < 0x100; i++ {
		pal[i] = colors.GetColorAt(math.Sqrt(float64(i) / float64(0xff)))
	}

	watermark := "NathanBaulch/rainbow-roads"
	if Version != "" {
		watermark += " " + Version
	}
	bg := image.NewPaletted(image.Rect(0, 0, int(width), int(height)), pal)
	for i := 0; i < len(bg.Pix); i += bg.Stride {
		if i == 0 {
			for j := 0; j < bg.Stride; j++ {
				bg.Pix[j] = 0xff
			}
		} else {
			copy(bg.Pix[i:i+bg.Stride], bg.Pix[0:bg.Stride])
		}
	}
	if !noWatermark {
		d := &font.Drawer{
			Dst:  bg,
			Src:  image.NewUniform(pal[0x80]),
			Face: basicfont.Face7x13,
		}
		b, _ := d.BoundString(watermark)
		b = b.Sub(b.Min)
		if b.In(fixed.R(0, 0, bg.Rect.Max.X-10, bg.Rect.Max.Y-10)) {
			d.Dot = fixed.P(bg.Rect.Max.X, bg.Rect.Max.Y).
				Sub(b.Max.Sub(fixed.P(0, basicfont.Face7x13.Height))).
				Sub(fixed.P(5, 5))
			d.DrawString(watermark)
		}
	}

	ims = make([]*image.Paletted, frames)

	for f := uint(0); f < frames; f++ {
		im := image.NewPaletted(bg.Rect, pal)
		copy(im.Pix, bg.Pix)
		fp := 1.2 * float64(f+1) / float64(frames)
		for _, act := range activities {
			var rprev *record
			for _, r := range act.records {
				pp := fp - r.p
				if pp < 0 {
					break
				} else if pp > 1 || (rprev != nil && r.x == rprev.x && r.y == rprev.y) {
					rprev = r
					continue
				}
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
				setPix := func(x, y int, _ color.Color) {
					ci := uint8(pp * 0xff)
					if !setPixIfLower(x, y, ci) {
						return
					}
					if ci < 0x80 {
						ci *= 2
						setPixIfLower(x-1, y, ci)
						setPixIfLower(x, y-1, ci)
						setPixIfLower(x+1, y, ci)
						setPixIfLower(x, y+1, ci)
					}
					if ci < 0x80 {
						ci *= 2
						setPixIfLower(x-1, y-1, ci)
						setPixIfLower(x-1, y+1, ci)
						setPixIfLower(x+1, y-1, ci)
						setPixIfLower(x+1, y+1, ci)
					}
				}
				if rprev != nil {
					if dx, dy := r.x-rprev.x, r.y-rprev.y; dx < -1 || dx > 1 || dy < -1 || dy > 1 {
						bresenham.Bresenham(plotterFunc(setPix), rprev.x, rprev.y, r.x, r.y, nil)
					} else {
						setPix(r.x, r.y, nil)
					}
				} else {
					setPix(r.x, r.y, nil)
				}
				rprev = r
			}
		}
		ims[f] = im
	}

	return nil
}

type plotterFunc func(x, y int, c color.Color)

func (f plotterFunc) Set(x, y int, c color.Color) {
	f(x, y, c)
}

func save() error {
	if dir := filepath.Dir(output); dir != "." {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}

	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	switch format.String() {
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
	g := &gif.GIF{Image: ims, Delay: make([]int, len(ims))}
	return gif.EncodeAll(w, g)
}

func savePNG(w io.Writer) error {
	a := apng.APNG{Frames: make([]apng.Frame, len(ims))}
	for i, img := range ims {
		a.Frames[i].Image = img
	}
	return apng.Encode(w, a)
}

func saveZIP(w io.Writer) error {
	z := zip.NewWriter(w)
	defer z.Close()
	for i, img := range ims {
		if w, err := z.Create(fmt.Sprintf("%d.jpg", i)); err != nil {
			return err
		} else if err := jpeg.Encode(w, img, nil); err != nil {
			return err
		}
	}
	return nil
}
