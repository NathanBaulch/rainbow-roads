package main

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/bcicen/go-units"
	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/image/colornames"
)

func NewFormatFlag(opts ...string) FormatFlag {
	return FormatFlag{opts: opts}
}

type FormatFlag struct {
	opts  []string
	index int
}

func (f *FormatFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}
	for i, opt := range f.opts {
		if strings.EqualFold(str, opt) {
			f.index = i + 1
			return nil
		}
	}
	return errors.New("invalid value")
}

func (f *FormatFlag) String() string {
	if f.index > 0 {
		return f.opts[f.index-1]
	}
	return ""
}

func (f *FormatFlag) IsZero() bool {
	return f.index == 0
}

type ColorsFlag []struct {
	colorful.Color
	pos float64
}

func (g *ColorsFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}

	parts := strings.Split(str, ",")
	*g = make(ColorsFlag, len(parts))
	var err error
	missingAt := 0

	for i, part := range parts {
		if part == "" {
			return errors.New("unexpected empty entry")
		}

		e := &(*g)[i]
		if pos := strings.Index(part, "@"); pos >= 0 {
			if strings.HasSuffix(part, "%") {
				if p, err := strconv.ParseFloat(part[pos+1:len(part)-1], 64); err != nil {
					return errors.New(fmt.Sprintf("position %q not recognized", part[pos+1:]))
				} else {
					e.pos = p / 100
				}
			} else {
				if e.pos, err = strconv.ParseFloat(part[pos+1:], 64); err != nil {
					return errors.New(fmt.Sprintf("position %q not recognized", part[pos+1:]))
				}
			}
			if e.pos < 0 || e.pos > 1 {
				return errors.New(fmt.Sprintf("position %q not within range", part[pos+1:]))
			}
			part = part[:pos]
		} else if i == 0 {
			e.pos = 0
		} else if i == len(parts)-1 {
			e.pos = 1
		} else {
			e.pos = math.NaN()
			if missingAt == 0 {
				missingAt = i
			}
		}
		if !math.IsNaN(e.pos) && missingAt > 0 {
			p := (*g)[missingAt-1].pos
			step := (e.pos - p) / float64(i+1-missingAt)
			for j := missingAt; j < i; j++ {
				p += step
				(*g)[j].pos = p
			}
			missingAt = 0
		}

		if e.Color, err = colorful.Hex(part); err != nil {
			if c, ok := colornames.Map[strings.ToLower(part)]; !ok {
				return errors.New(fmt.Sprintf("color %q not recognized", part))
			} else {
				e.Color, _ = colorful.MakeColor(c)
			}
		}
		i++
	}

	return nil
}

func (g *ColorsFlag) String() string {
	parts := make([]string, len(*g))
	for i, e := range *g {
		var hex string
		if r, g, b := e.Color.RGB255(); r>>4 == r&0xf && g>>4 == g&0xf && b>>4 == b&0xf {
			hex = fmt.Sprintf("#%1x%1x%1x", r&0xf, g&0xf, b&0xf)
		} else {
			hex = fmt.Sprintf("#%02x%02x%02x", r, g, b)
		}
		if (i == 0 && e.pos == 0) || (i == len(*g)-1 && e.pos == 1) {
			parts[i] = hex
		} else {
			parts[i] = fmt.Sprintf("%s@%s", hex, formatFloat(e.pos))
		}
	}
	return strings.Join(parts, ",")
}

func (g *ColorsFlag) GetColorAt(p float64) color.Color {
	last := len(*g) - 1
	for i := 0; i < last; i++ {
		if e0, e1 := (*g)[i], (*g)[i+1]; e0.pos <= p && p <= e1.pos {
			return e0.Color.BlendHcl(e1.Color, (p-e0.pos)/(e1.pos-e0.pos)).Clamped()
		}
	}
	return (*g)[last].Color
}

type SportsFlag []string

func (s *SportsFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}
	for _, str := range strings.Split(str, ",") {
		*s = append(*s, str)
	}
	return nil
}

func (s *SportsFlag) String() string {
	sort.Strings(*s)
	return strings.Join(*s, ",")
}

type DateFlag struct{ time.Time }

func (d *DateFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}
	if val, err := dateparse.ParseIn(str, time.UTC); err != nil {
		return errors.New("date not recognized")
	} else {
		*d = DateFlag{val}
		return nil
	}
}

type DurationFlag struct{ time.Duration }

func (d *DurationFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}

	var val time.Duration
	var err error
	if val, err = time.ParseDuration(str); err != nil {
		if i, err := strconv.ParseInt(str, 10, 64); err != nil {
			return errors.New("duration not recognized")
		} else {
			val = time.Duration(i) * time.Second
		}
	}
	if val <= 0 {
		return errors.New("must be positive")
	}
	*d = DurationFlag{val}
	return nil
}

type DistanceFlag float64

func (d *DistanceFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}
	if f, err := parseDistance(str); err != nil {
		return err
	} else {
		*d = DistanceFlag(f)
		return nil
	}
}

func (d *DistanceFlag) String() string {
	return formatFloat(float64(*d))
}

type RegionFlag struct{ Region }

func (r *RegionFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}
	if parts := strings.Split(str, ","); len(parts) < 2 || len(parts) > 3 {
		return errors.New("invalid number of parts")
	} else if lat, err := strconv.ParseFloat(parts[0], 64); err != nil {
		return errors.New(fmt.Sprintf("latitude %q not recognized", parts[0]))
	} else if lon, err := strconv.ParseFloat(parts[1], 64); err != nil {
		return errors.New(fmt.Sprintf("longitude %q not recognized", parts[1]))
	} else if lat < -85 || lat > 85 {
		return errors.New(fmt.Sprintf("latitude %q not within range", formatFloat(lat)))
	} else if lon < -180 || lon > 180 {
		return errors.New(fmt.Sprintf("longitude %q not within range", formatFloat(lon)))
	} else {
		r.lat, r.lon = degreesToRadians(lat), degreesToRadians(lon)
		radius := 100.0
		if len(parts) == 3 {
			if radius, err = parseDistance(parts[2]); err != nil {
				return errors.New("radius " + err.Error())
			} else if radius <= 0 {
				return errors.New(fmt.Sprintf("radius %q must be positive", formatFloat(radius)))
			}
		}
		r.radius = radius
		return nil
	}
}

var distanceRE = regexp.MustCompile(`^(.*\d)\s?(\w+)?$`)

func parseDistance(str string) (float64, error) {
	if m := distanceRE.FindStringSubmatch(str); len(m) != 3 {
		return 0, errors.New("format not recognized")
	} else if f, err := strconv.ParseFloat(m[1], 64); err != nil {
		return 0, errors.New(fmt.Sprintf("number %q not recognized", m[1]))
	} else if m[2] == "" || strings.EqualFold(m[2], units.Meter.Symbol) {
		return f, nil
	} else if u, err := units.Find(m[2]); err != nil {
		return 0, errors.New(fmt.Sprintf("unit %q not recognized", m[2]))
	} else if v, err := units.ConvertFloat(f, u, units.Meter); err != nil {
		return 0, errors.New(fmt.Sprintf("unit %q not a distance", m[2]))
	} else {
		return v.Float(), nil
	}
}
