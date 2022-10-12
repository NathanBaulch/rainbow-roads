package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
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

	"github.com/StephaneBunel/bresenham"
	"github.com/kettek/apng"
	"github.com/schollz/progressbar"
	"github.com/spf13/pflag"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	Version    string
	fullTitle  string
	shortTitle string

	input       []string
	output      string
	width       uint
	frames      uint
	fps         uint
	format      = NewFormatFlag("gif", "png", "zip")
	colors      ColorsFlag
	colorDepth  uint
	speed       float64
	loop        bool
	noWatermark bool

	sports        SportsFlag
	after         time.Time
	before        time.Time
	minDuration   time.Duration
	maxDuration   time.Duration
	minDistance   float64
	maxDistance   float64
	minPace       time.Duration
	maxPace       time.Duration
	boundedBy     Circle
	startsNear    Circle
	endsNear      Circle
	passesThrough Circle

	en         = message.NewPrinter(language.English)
	files      []*file
	activities []*activity
	maxDur     time.Duration
	box        Box
	images     []*image.Paletted
)

func init() {
	shortTitle = "rainbow-roads"
	if Version != "" {
		shortTitle += " " + Version
	}
	fullTitle = "NathanBaulch/" + shortTitle

	_ = colors.Set("#fff,#ff8,#911,#414,#007@.5,#003")
}

func main() {
	general := &pflag.FlagSet{}
	general.StringVar(&output, "output", "out", "optional path of the generated file")
	general.Var(&format, "format", "output file format string, supports gif, png, zip")
	general.VisitAll(func(f *pflag.Flag) { pflag.Var(f.Value, f.Name, f.Usage) })

	rendering := &pflag.FlagSet{}
	rendering.UintVar(&frames, "frames", 200, "number of animation frames")
	rendering.UintVar(&fps, "fps", 20, "animation frame rate")
	rendering.UintVar(&width, "width", 500, "width of the generated image in pixels")
	rendering.Var(&colors, "colors", "CSS linear-colors inspired color scheme string, eg red,yellow,green,blue,black")
	rendering.UintVar(&colorDepth, "color_depth", 5, "number of bits per color in the image palette")
	rendering.Float64Var(&speed, "speed", 1.25, "how quickly activities should progress")
	rendering.BoolVar(&loop, "loop", false, "start each activity sequentially and animate continuously")
	rendering.BoolVar(&noWatermark, "no_watermark", false, "suppress the embedded project name and version string")
	rendering.VisitAll(func(f *pflag.Flag) { pflag.Var(f.Value, f.Name, f.Usage) })

	filters := &pflag.FlagSet{}
	filters.Var(&sports, "sport", "sports to include, can be specified multiple times, eg running, cycling")
	filters.Var((*DateFlag)(&after), "after", "date from which activities should be included")
	filters.Var((*DateFlag)(&before), "before", "date prior to which activities should be included")
	filters.Var((*DurationFlag)(&minDuration), "min_duration", "shortest duration of included activities, eg 15m")
	filters.Var((*DurationFlag)(&maxDuration), "max_duration", "longest duration of included activities, eg 1h")
	filters.Var((*DistanceFlag)(&minDistance), "min_distance", "shortest distance of included activities, eg 2km")
	filters.Var((*DistanceFlag)(&maxDistance), "max_distance", "greatest distance of included activities, eg 10mi")
	filters.Var((*PaceFlag)(&minPace), "min_pace", "slowest pace of included activities, eg 8km/h")
	filters.Var((*PaceFlag)(&maxPace), "max_pace", "fastest pace of included activities, eg 10min/mi")
	filters.Var((*CircleFlag)(&boundedBy), "bounded_by", "region that activities must be fully contained within, eg -37.8,144.9,10km")
	filters.Var((*CircleFlag)(&startsNear), "starts_near", "region that activities must start from, eg 51.53,-0.21,1km")
	filters.Var((*CircleFlag)(&endsNear), "ends_near", "region that activities must end in, eg 30.06,31.22,1km")
	filters.Var((*CircleFlag)(&passesThrough), "passes_through", "region that activities must pass through, eg 40.69,-74.12,10mi")
	filters.VisitAll(func(f *pflag.Flag) { pflag.Var(f.Value, f.Name, f.Usage) })

	pflag.CommandLine.Init("", pflag.ContinueOnError)
	pflag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage of", shortTitle+":")
		general.PrintDefaults()
		fmt.Fprintln(os.Stderr, "Filtering:")
		filters.PrintDefaults()
		fmt.Fprintln(os.Stderr, "Rendering:")
		rendering.PrintDefaults()
	}
	if err := pflag.CommandLine.Parse(os.Args[1:]); err != nil {
		if err == pflag.ErrHelp {
			return
		}
		fmt.Println(err)
		os.Exit(2)
	}

	invalidFlag := func(name string, reason string) {
		fmt.Fprintf(os.Stderr, "invalid value %q for flag --%s: %s\n", pflag.Lookup(name).Value, name, reason)
		pflag.Usage()
		os.Exit(2)
	}
	if frames == 0 {
		invalidFlag("frames", "must be positive")
	}
	if fps == 0 {
		invalidFlag("fps", "must be positive")
	}
	if width == 0 {
		invalidFlag("width", "must be positive")
	}
	if colorDepth == 0 {
		invalidFlag("color_depth", "must be positive")
	}
	if speed < 1 {
		invalidFlag("speed", "must be greater than or equal to 1")
	}

	en.Println(shortTitle)

	input = pflag.Args()
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
					return fmt.Errorf("input path pattern %q malformed", in)
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
					return fmt.Errorf("input path %q not found", path)
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

	en.Println("activity files:", len(files))
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
				if b, err := io.ReadAll(f); err != nil {
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
		var parser func(io.Reader) ([]*activity, error)
		if strings.EqualFold(ext, ".fit") {
			parser = parseFIT
		} else if strings.EqualFold(ext, ".gpx") {
			parser = parseGPX
		} else if strings.EqualFold(ext, ".tcx") {
			parser = parseTCX
		} else {
			return nil
		}
		parse := func() ([]*activity, error) {
			var r io.Reader
			var err error
			if r, err = fsys.Open(path); err != nil {
				return nil, err
			} else if gz {
				if r, err = gzip.NewReader(r); err != nil {
					return nil, err
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
	parse func() ([]*activity, error)
}

func parse() error {
	pb := progressbar.New(len(files))
	pb.SetWriter(os.Stderr)
	wg := sync.WaitGroup{}
	wg.Add(len(files))
	res := make([]struct {
		acts []*activity
		err  error
	}, len(files))
	for i := range files {
		i := i
		go func() {
			res[i].acts, res[i].err = files[i].parse()
			_ = pb.Add(1)
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Fprintln(os.Stderr)
	activities = make([]*activity, 0, len(files))
	for _, r := range res {
		if r.err != nil {
			fmt.Fprintln(os.Stderr, "WARN:", r.err)
		} else {
			activities = append(activities, r.acts...)
		}
	}
	if len(activities) == 0 {
		return errors.New("no matching activities found")
	}

	sportStats := make(map[string]int)
	minDur := time.Duration(math.MaxInt64)
	var minDate, maxDate time.Time
	minDist, maxDist := math.MaxFloat64, 0.0
	minP, maxP := time.Duration(math.MaxInt64), time.Duration(0)
	sumRec := 0
	var sumDur time.Duration
	sumDist := 0.0
	var startBox, endBox Box
	for i := len(activities) - 1; i >= 0; i-- {
		act := activities[i]
		include := passesThrough.IsZero()
		exclude := false
		for j, r := range act.records {
			if j == 0 && !startsNear.IsZero() && !startsNear.Contains(r.pt) {
				exclude = true
				break
			}
			if j == len(act.records)-1 && !endsNear.IsZero() && !endsNear.Contains(r.pt) {
				exclude = true
				break
			}
			if !boundedBy.IsZero() && !boundedBy.Contains(r.pt) {
				exclude = true
				break
			}
			if !include && passesThrough.Contains(r.pt) {
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
		ts0, ts1 := act.records[0].ts, act.records[len(act.records)-1].ts
		if minDate.IsZero() || ts0.Before(minDate) {
			minDate = ts0
		}
		if maxDate.IsZero() || ts1.After(maxDate) {
			maxDate = ts1
		}
		dur := ts1.Sub(ts0)
		if dur < minDur {
			minDur = dur
		}
		if dur > maxDur {
			maxDur = dur
		}
		if act.distance < minDist {
			minDist = act.distance
		}
		if act.distance > maxDist {
			maxDist = act.distance
		}
		pace := time.Duration(float64(dur) / act.distance)
		if pace < minP {
			minP = pace
		}
		if pace > maxP {
			maxP = pace
		}

		sumRec += len(act.records)
		sumDur += dur
		sumDist += act.distance

		for _, r := range act.records {
			box = box.Enclose(r.pt)
		}
		startBox = startBox.Enclose(act.records[0].pt)
		endBox = endBox.Enclose(act.records[len(act.records)-1].pt)
	}

	if len(activities) == 0 {
		return errors.New("no matching activities found")
	}

	bounds := Circle{origin: box.Center()}
	starts := Circle{origin: startBox.Center()}
	ends := Circle{origin: endBox.Center()}
	for _, act := range activities {
		for _, r := range act.records {
			bounds = bounds.Enclose(r.pt)
		}
		starts = starts.Enclose(act.records[0].pt)
		ends = ends.Enclose(act.records[len(act.records)-1].pt)
	}

	en.Printf("activities:    %d\n", len(activities))
	en.Printf("records:       %d\n", sumRec)
	en.Printf("sports:        %s\n", sprintSportStats(en, sportStats))
	en.Printf("period:        %s\n", sprintPeriod(en, minDate, maxDate))
	en.Printf("duration:      %s to %s, average %s, total %s\n", minDur, maxDur, (sumDur / time.Duration(len(activities))).Truncate(time.Second), sumDur)
	en.Printf("distance:      %.1fkm to %.1fkm, average %.1fkm, total %.1fkm\n", minDist/1000, maxDist/1000, sumDist/float64(len(activities))/1000, sumDist/1000)
	en.Printf("pace:          %s/km to %s/km, average %s/km\n", (minP * 1000).Truncate(time.Second), (maxP * 1000).Truncate(time.Second), (sumDur * 1000 / time.Duration(sumDist)).Truncate(time.Second))
	en.Printf("bounds:        %s\n", bounds)
	en.Printf("starts within: %s\n", starts)
	en.Printf("ends within:   %s\n", ends)
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

func includeTimestamp(from, to time.Time) bool {
	if !after.IsZero() && after.After(from) {
		return false
	}
	if !before.IsZero() && before.Before(to) {
		return false
	}
	return true
}

func includeDuration(duration time.Duration) bool {
	if duration == 0 {
		return false
	}
	if minDuration != 0 && duration < minDuration {
		return false
	}
	if maxDuration != 0 && duration > maxDuration {
		return false
	}
	return true
}

func includeDistance(distance float64) bool {
	if distance == 0 {
		return false
	}
	if minDistance != 0 && distance < float64(minDistance) {
		return false
	}
	if maxDistance != 0 && distance > float64(maxDistance) {
		return false
	}
	return true
}

func includePace(duration time.Duration, distance float64) bool {
	pace := time.Duration(float64(duration) / distance)
	if pace == 0 {
		return false
	}
	if minPace != 0 && pace < minPace {
		return false
	}
	if maxPace != 0 && pace > maxPace {
		return false
	}
	return true
}

type activity struct {
	sport    string
	distance float64
	records  []*record
}

type record struct {
	ts   time.Time
	pt   Point
	x, y int
	pc   float64
}

func render() error {
	if loop {
		sort.Slice(activities, func(i, j int) bool { return activities[i].records[0].ts.Before(activities[j].records[0].ts) })
	}

	minX, minY := mercatorMeters(box.min)
	maxX, maxY := mercatorMeters(box.max)
	dX, dY := maxX-minX, maxY-minY
	scale := float64(width) / dX
	height := uint(dY * scale)
	scale *= 0.9
	minX -= 0.05 * dX
	maxY += 0.05 * dY
	tScale := 1 / (speed * float64(maxDur))
	for i, act := range activities {
		ts0 := act.records[0].ts
		tOffset := 0.0
		if loop {
			tOffset = float64(i) / float64(len(activities))
		}
		for _, r := range act.records {
			x, y := mercatorMeters(r.pt)
			r.x = int((x - minX) * scale)
			r.y = int((maxY - y) * scale)
			r.pc = tOffset + float64(r.ts.Sub(ts0))*tScale
		}
	}

	pal := color.Palette(make([]color.Color, 1<<colorDepth))
	for i := 0; i < len(pal)-2; i++ {
		pal[i] = colors.GetColorAt(float64(i) / float64(len(pal)-3))
	}
	pal[len(pal)-2] = color.Black
	pal[len(pal)-1] = color.Transparent

	images = make([]*image.Paletted, frames)
	for i := range images {
		im := image.NewPaletted(image.Rect(0, 0, int(width), int(height)), pal)
		if i == 0 {
			drawFill(im, uint8(len(pal)-2))
			if !noWatermark {
				drawString(im, fullTitle, uint8(len(pal)/2))
			}
		} else {
			copy(im.Pix, images[0].Pix)
		}
		images[i] = im
	}

	wg := &sync.WaitGroup{}
	wg.Add(int(frames))
	for f := uint(0); f < frames; f++ {
		f := f
		go func() {
			fpc := float64(f+1) / float64(frames)
			gp := &glowPlotter{images[f]}
			for _, act := range activities {
				var rPrev *record
				for _, r := range act.records {
					pc := fpc - r.pc
					if pc < 0 {
						if !loop {
							break
						}
						pc++
					}
					if rPrev != nil && (r.x != rPrev.x || r.y != rPrev.y) {
						ci := uint8(len(pal) - 3)
						if pc >= 0 && pc < 1 {
							ci = uint8(math.Sqrt(pc) * float64(len(pal)-2))
						}
						bresenham.DrawLine(gp, rPrev.x, rPrev.y, r.x, r.y, grays[ci])
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

type glowPlotter struct {
	*image.Paletted
}

func (p *glowPlotter) Set(x, y int, c color.Color) {
	p.SetColorIndex(x, y, c.(color.Gray).Y)
}

func (p *glowPlotter) SetColorIndex(x, y int, ci uint8) {
	if p.setPixIfLower(x, y, ci) {
		const sqrt2 = 1.414213562
		if i := float64(ci) * sqrt2; i < float64(len(p.Palette)-2) {
			ci = uint8(i)
			p.setPixIfLower(x-1, y, ci)
			p.setPixIfLower(x, y-1, ci)
			p.setPixIfLower(x+1, y, ci)
			p.setPixIfLower(x, y+1, ci)
		}
		if i := float64(ci) * sqrt2; i < float64(len(p.Palette)-2) {
			ci = uint8(i)
			p.setPixIfLower(x-1, y-1, ci)
			p.setPixIfLower(x-1, y+1, ci)
			p.setPixIfLower(x+1, y-1, ci)
			p.setPixIfLower(x+1, y+1, ci)
		}
	}
}

func (p *glowPlotter) setPixIfLower(x, y int, ci uint8) bool {
	if (image.Point{X: x, Y: y}.In(p.Rect)) {
		i := p.PixOffset(x, y)
		if p.Pix[i] > ci {
			p.Pix[i] = ci
			return true
		}
	}
	return false
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
	defer func() {
		if err := out.Close(); err != nil {
			log.Fatal(err)
		}
	}()

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
	d := int(math.Round(100 / float64(fps)))
	for i := range images {
		g.Disposal[i] = gif.DisposalNone
		g.Delay[i] = d
	}
	return gif.EncodeAll(&gifWriter{Writer: bufio.NewWriter(w)}, g)
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
		a.Frames[i].DelayDenominator = uint16(fps)
	}
	return apng.Encode(&pngWriter{Writer: w}, a)
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
