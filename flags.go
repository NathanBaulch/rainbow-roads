package main

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NathanBaulch/rainbow-roads/conv"
	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/NathanBaulch/rainbow-roads/img"
	"github.com/NathanBaulch/rainbow-roads/parse"
	"github.com/araddon/dateparse"
	"github.com/bcicen/go-units"
	"github.com/spf13/pflag"
)

func filterFlagSet(selector *parse.Selector) *pflag.FlagSet {
	fs := &pflag.FlagSet{}
	fs.Var((*SportsFlag)(&selector.Sports), "sport", "sports to include, can be specified multiple times, eg running, cycling")
	fs.Var((*DateFlag)(&selector.After), "after", "date from which activities should be included")
	fs.Var((*DateFlag)(&selector.Before), "before", "date prior to which activities should be included")
	fs.Var((*DurationFlag)(&selector.MinDuration), "min_duration", "shortest duration of included activities, eg 15m")
	fs.Var((*DurationFlag)(&selector.MaxDuration), "max_duration", "longest duration of included activities, eg 1h")
	fs.Var((*DistanceFlag)(&selector.MinDistance), "min_distance", "shortest distance of included activities, eg 2km")
	fs.Var((*DistanceFlag)(&selector.MaxDistance), "max_distance", "greatest distance of included activities, eg 10mi")
	fs.Var((*PaceFlag)(&selector.MinPace), "min_pace", "slowest pace of included activities, eg 8km/h")
	fs.Var((*PaceFlag)(&selector.MaxPace), "max_pace", "fastest pace of included activities, eg 10min/mi")
	fs.Var((*CircleFlag)(&selector.BoundedBy), "bounded_by", "region that activities must be fully contained within, eg -37.8,144.9,10km")
	fs.Var((*CircleFlag)(&selector.StartsNear), "starts_near", "region that activities must start from, eg 51.53,-0.21,1km")
	fs.Var((*CircleFlag)(&selector.EndsNear), "ends_near", "region that activities must end in, eg 30.06,31.22,1km")
	fs.Var((*CircleFlag)(&selector.PassesThrough), "passes_through", "region that activities must pass through, eg 40.69,-74.12,10mi")
	return fs
}

func flagError(name string, value any, reason string) error {
	return fmt.Errorf("invalid value %q for flag --%s: %s", value, name, reason)
}

type ColorsFlag img.ColorGradient

func (c *ColorsFlag) Type() string {
	return "colors"
}

func (c *ColorsFlag) Set(str string) error {
	return (*img.ColorGradient)(c).Parse(str)
}

func (c *ColorsFlag) String() string {
	if c == nil {
		return ""
	}
	return (*img.ColorGradient)(c).String()
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
	return conv.FormatFloat(float64(*d))
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

type CircleFlag geo.Circle

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
		return fmt.Errorf("latitude %q not within range", conv.FormatFloat(lat))
	} else if lon < -180 || lon > 180 {
		return fmt.Errorf("longitude %q not within range", conv.FormatFloat(lon))
	} else {
		*c = CircleFlag{
			Origin: geo.NewPointFromDegrees(lat, lon),
			Radius: 1000,
		}
		if len(parts) == 3 {
			if c.Radius, err = parseDistance(parts[2]); err != nil {
				return errors.New("radius " + err.Error())
			}
		}
		return nil
	}
}

func (c *CircleFlag) String() string {
	if c == nil || geo.Circle(*c).IsZero() {
		return ""
	}
	return geo.Circle(*c).String()
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
