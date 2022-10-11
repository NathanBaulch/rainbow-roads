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

func (f *FormatFlag) Type() string {
	return "format"
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

func (c *ColorsFlag) Type() string {
	return "colors"
}

func (c *ColorsFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}

	parts := strings.Split(str, ",")
	*c = make(ColorsFlag, len(parts))
	var err error
	missingAt := 0

	for i, part := range parts {
		if part == "" {
			return errors.New("unexpected empty entry")
		}

		e := &(*c)[i]
		if pos := strings.Index(part, "@"); pos >= 0 {
			if strings.HasSuffix(part, "%") {
				if p, err := strconv.ParseFloat(part[pos+1:len(part)-1], 64); err != nil {
					return fmt.Errorf("position %q not recognized", part[pos+1:])
				} else {
					e.pos = p / 100
				}
			} else {
				if e.pos, err = strconv.ParseFloat(part[pos+1:], 64); err != nil {
					return fmt.Errorf("position %q not recognized", part[pos+1:])
				}
			}
			if e.pos < 0 || e.pos > 1 {
				return fmt.Errorf("position %q not within range", part[pos+1:])
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
			p := (*c)[missingAt-1].pos
			step := (e.pos - p) / float64(i+1-missingAt)
			for j := missingAt; j < i; j++ {
				p += step
				(*c)[j].pos = p
			}
			missingAt = 0
		}

		if e.Color, err = colorful.Hex(part); err != nil {
			if col, ok := colornames.Map[strings.ToLower(part)]; !ok {
				return fmt.Errorf("color %q not recognized", part)
			} else {
				e.Color, _ = colorful.MakeColor(col)
			}
		}
		i++
	}

	return nil
}

func (c *ColorsFlag) String() string {
	parts := make([]string, len(*c))
	for i, e := range *c {
		var hex string
		if r, g, b := e.Color.RGB255(); r>>4 == r&0xf && g>>4 == g&0xf && b>>4 == b&0xf {
			hex = fmt.Sprintf("#%1x%1x%1x", r&0xf, g&0xf, b&0xf)
		} else {
			hex = fmt.Sprintf("#%02x%02x%02x", r, g, b)
		}
		if (i == 0 && e.pos == 0) || (i == len(*c)-1 && e.pos == 1) {
			parts[i] = hex
		} else {
			parts[i] = fmt.Sprintf("%s@%s", hex, formatFloat(e.pos))
		}
	}
	return strings.Join(parts, ",")
}

func (c *ColorsFlag) GetColorAt(p float64) color.Color {
	last := len(*c) - 1
	for i := 0; i < last; i++ {
		if e0, e1 := (*c)[i], (*c)[i+1]; e0.pos <= p && p <= e1.pos {
			return e0.Color.BlendHcl(e1.Color, (p-e0.pos)/(e1.pos-e0.pos)).Clamped()
		}
	}
	return (*c)[last].Color
}

type SportsFlag []string

func (s *SportsFlag) Type() string {
	return "sports"
}

func (s *SportsFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}
	for _, str = range strings.Split(str, ",") {
		*s = append(*s, str)
	}
	return nil
}

func (s *SportsFlag) String() string {
	sort.Strings(*s)
	return strings.Join(*s, ",")
}

type DateFlag time.Time

func (d *DateFlag) Type() string {
	return "date"
}

func (d *DateFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}
	if val, err := dateparse.ParseIn(str, time.UTC); err != nil {
		return errors.New("date not recognized")
	} else {
		*d = DateFlag(val)
		return nil
	}
}

func (d *DateFlag) String() string {
	if d == nil || time.Time(*d).IsZero() {
		return ""
	}
	return time.Time(*d).String()
}

type DurationFlag time.Duration

func (d *DurationFlag) Type() string {
	return "duration"
}

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
	*d = DurationFlag(val)
	return nil
}

func (d *DurationFlag) String() string {
	if d == nil || *d == 0 {
		return ""
	}
	return time.Duration(*d).String()
}

type DistanceFlag float64

func (d *DistanceFlag) Type() string {
	return "distance"
}

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

type PaceFlag time.Duration

func (p *PaceFlag) Type() string {
	return "pace"
}

var paceRE = regexp.MustCompile(`^([^/]+)(/([^/]+))?$`)

func (p *PaceFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}
	if m := paceRE.FindStringSubmatch(str); len(m) != 4 {
		return errors.New("format not recognized")
	} else if d, err := time.ParseDuration(m[1]); err != nil {
		return fmt.Errorf("duration %q not recognized", m[1])
	} else if d <= 0 {
		return errors.New("must be positive")
	} else if m[3] == "" || strings.EqualFold(m[3], units.Meter.Symbol) {
		*p = PaceFlag(d)
	} else if u, err := units.Find(m[3]); err != nil {
		return fmt.Errorf("unit %q not recognized", m[3])
	} else if v, err := units.ConvertFloat(float64(d), units.Meter, u); err != nil {
		return fmt.Errorf("unit %q not a distance", m[3])
	} else {
		*p = PaceFlag(v.Float())
	}
	return nil
}

func (p *PaceFlag) String() string {
	if p == nil || *p == 0 {
		return ""
	}
	return time.Duration(*p).String()
}

type CircleFlag Circle

func (c *CircleFlag) Type() string {
	return "circle"
}

func (c *CircleFlag) Set(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}
	if parts := strings.Split(str, ","); len(parts) < 2 || len(parts) > 3 {
		return errors.New("invalid number of parts")
	} else if lat, err := strconv.ParseFloat(parts[0], 64); err != nil {
		return fmt.Errorf("latitude %q not recognized", parts[0])
	} else if lon, err := strconv.ParseFloat(parts[1], 64); err != nil {
		return fmt.Errorf("longitude %q not recognized", parts[1])
	} else if lat < -85 || lat > 85 {
		return fmt.Errorf("latitude %q not within range", formatFloat(lat))
	} else if lon < -180 || lon > 180 {
		return fmt.Errorf("longitude %q not within range", formatFloat(lon))
	} else {
		*c = CircleFlag{
			origin: newPointFromDegrees(lat, lon),
			radius: 100,
		}
		if len(parts) == 3 {
			if c.radius, err = parseDistance(parts[2]); err != nil {
				return errors.New("radius " + err.Error())
			}
		}
		return nil
	}
}

func (c *CircleFlag) String() string {
	if c == nil || Circle(*c).IsZero() {
		return ""
	}
	return Circle(*c).String()
}

var distanceRE = regexp.MustCompile(`^(.*\d)\s?(\w+)?$`)

func parseDistance(str string) (float64, error) {
	if m := distanceRE.FindStringSubmatch(str); len(m) != 3 {
		return 0, errors.New("format not recognized")
	} else if f, err := strconv.ParseFloat(m[1], 64); err != nil {
		return 0, fmt.Errorf("number %q not recognized", m[1])
	} else if f < 0 {
		return 0, errors.New("must be positive")
	} else if m[2] == "" || strings.EqualFold(m[2], units.Meter.Symbol) {
		return f, nil
	} else if u, err := units.Find(m[2]); err != nil {
		return 0, fmt.Errorf("unit %q not recognized", m[2])
	} else if v, err := units.ConvertFloat(f, u, units.Meter); err != nil {
		return 0, fmt.Errorf("unit %q not a distance", m[2])
	} else {
		return v.Float(), nil
	}
}
