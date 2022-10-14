package parse

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/NathanBaulch/rainbow-roads/scan"
	"golang.org/x/exp/slices"
	"golang.org/x/text/message"
)

func Parse(files []*scan.File, selector *Selector) ([]*Activity, *Stats, error) {
	wg := sync.WaitGroup{}
	wg.Add(len(files))
	res := make([]struct {
		acts []*Activity
		err  error
	}, len(files))
	for i := range files {
		i := i
		go func() {
			defer wg.Done()
			var parser func(io.Reader, *Selector) ([]*Activity, error)
			switch files[i].Ext {
			case ".fit":
				parser = parseFIT
			case ".gpx":
				parser = parseGPX
			case ".tcx":
				parser = parseTCX
			default:
				return
			}
			if r, err := files[i].Opener(); err != nil {
				res[i].err = err
			} else {
				res[i].acts, res[i].err = parser(r, selector)
			}
		}()
	}
	wg.Wait()

	activities := make([]*Activity, 0, len(files))
	for _, r := range res {
		if r.err != nil {
			fmt.Fprintln(os.Stderr, "WARN:", r.err)
		} else {
			activities = append(activities, r.acts...)
		}
	}
	if len(activities) == 0 {
		return nil, nil, errors.New("no matching activities found")
	}

	stats := &Stats{
		SportCounts: make(map[string]int),
		After:       time.UnixMilli(math.MaxInt64),
		MinDuration: time.Duration(math.MaxInt64),
		MinDistance: math.MaxFloat64,
		MinPace:     time.Duration(math.MaxInt64),
	}
	var startExtent, endExtent geo.Box

	for i := len(activities) - 1; i >= 0; i-- {
		act := activities[i]
		include := selector.PassesThrough.IsZero()
		exclude := false
		for j, r := range act.Records {
			if !selector.Bounded(r.Position) {
				exclude = true
				break
			}
			if j == 0 && !selector.Starts(r.Position) {
				exclude = true
				break
			}
			if j == len(act.Records)-1 && !selector.Ends(r.Position) {
				exclude = true
				break
			}
			if !include && selector.Passes(r.Position) {
				include = true
			}
		}
		if exclude || !include {
			j := len(activities) - 1
			activities[i] = activities[j]
			activities = activities[:j]
			continue
		}

		if act.Sport == "" {
			stats.SportCounts["unknown"]++
		} else {
			stats.SportCounts[strings.ToLower(act.Sport)]++
		}
		ts0, ts1 := act.Records[0].Timestamp, act.Records[len(act.Records)-1].Timestamp
		if ts0.Before(stats.After) {
			stats.After = ts0
		}
		if ts1.After(stats.Before) {
			stats.Before = ts1
		}
		dur := ts1.Sub(ts0)
		if dur < stats.MinDuration {
			stats.MinDuration = dur
		}
		if dur > stats.MaxDuration {
			stats.MaxDuration = dur
		}
		if act.Distance < stats.MinDistance {
			stats.MinDistance = act.Distance
		}
		if act.Distance > stats.MaxDistance {
			stats.MaxDistance = act.Distance
		}
		pace := time.Duration(float64(dur) / act.Distance)
		if pace < stats.MinPace {
			stats.MinPace = pace
		}
		if pace > stats.MaxPace {
			stats.MaxPace = pace
		}

		stats.CountRecords += len(act.Records)
		stats.SumDuration += dur
		stats.SumDistance += act.Distance

		for _, r := range act.Records {
			stats.Extent = stats.Extent.Enclose(r.Position)
		}
		startExtent = startExtent.Enclose(act.Records[0].Position)
		endExtent = endExtent.Enclose(act.Records[len(act.Records)-1].Position)
	}

	if len(activities) == 0 {
		return nil, nil, errors.New("no matching activities found")
	}

	stats.CountActivities = len(activities)
	stats.BoundedBy = geo.Circle{Origin: stats.Extent.Center()}
	stats.StartsNear = geo.Circle{Origin: startExtent.Center()}
	stats.EndsNear = geo.Circle{Origin: endExtent.Center()}
	for _, act := range activities {
		for _, r := range act.Records {
			stats.BoundedBy = stats.BoundedBy.Enclose(r.Position)
		}
		stats.StartsNear = stats.StartsNear.Enclose(act.Records[0].Position)
		stats.EndsNear = stats.EndsNear.Enclose(act.Records[len(act.Records)-1].Position)
	}

	return activities, stats, nil
}

type Selector struct {
	Sports                                         []string
	After, Before                                  time.Time
	MinDuration, MaxDuration                       time.Duration
	MinDistance, MaxDistance                       float64
	MinPace, MaxPace                               time.Duration
	BoundedBy, StartsNear, EndsNear, PassesThrough geo.Circle
}

func (s *Selector) Sport(sport string) bool {
	return len(s.Sports) == 0 || slices.IndexFunc(s.Sports, func(s string) bool { return strings.EqualFold(s, sport) }) >= 0
}

func (s *Selector) Timestamp(from, to time.Time) bool {
	return (s.After.IsZero() || s.After.Before(from)) && (s.Before.IsZero() || s.Before.After(to))
}

func (s *Selector) Duration(duration time.Duration) bool {
	return duration > 0 &&
		(s.MinDuration == 0 || duration > s.MinDuration) &&
		(s.MaxDuration == 0 || duration < s.MaxDuration)
}

func (s *Selector) Distance(distance float64) bool {
	return distance > 0 &&
		(s.MinDistance == 0 || distance > s.MinDistance) &&
		(s.MaxDistance == 0 || distance < s.MaxDistance)
}

func (s *Selector) Pace(duration time.Duration, distance float64) bool {
	pace := time.Duration(float64(duration) / distance)
	return pace > 0 &&
		(s.MinPace == 0 || pace > s.MinPace) &&
		(s.MaxPace == 0 || pace < s.MaxPace)
}

func (s *Selector) Bounded(pt geo.Point) bool {
	return s.BoundedBy.IsZero() || s.BoundedBy.Contains(pt)
}

func (s *Selector) Starts(pt geo.Point) bool {
	return s.StartsNear.IsZero() || s.StartsNear.Contains(pt)
}

func (s *Selector) Ends(pt geo.Point) bool {
	return s.EndsNear.IsZero() || s.EndsNear.Contains(pt)
}

func (s *Selector) Passes(pt geo.Point) bool {
	return s.PassesThrough.IsZero() || s.PassesThrough.Contains(pt)
}

type Activity struct {
	Sport    string
	Distance float64
	Records  []*Record
}

type Record struct {
	Timestamp time.Time
	Position  geo.Point
	X, Y      int
	Percent   float64
}

type Stats struct {
	CountActivities, CountRecords         int
	SportCounts                           map[string]int
	After, Before                         time.Time
	MinDuration, MaxDuration, SumDuration time.Duration
	MinDistance, MaxDistance, SumDistance float64
	MinPace, MaxPace                      time.Duration
	BoundedBy, StartsNear, EndsNear       geo.Circle
	Extent                                geo.Box
}

func (s *Stats) Print(p *message.Printer) {
	avgDur := s.SumDuration / time.Duration(s.CountActivities)
	avgDist := s.SumDistance / float64(s.CountActivities)
	avgPace := s.SumDuration / time.Duration(s.SumDistance)

	p.Printf("activities:    %d\n", s.CountActivities)
	p.Printf("records:       %d\n", s.CountRecords)
	p.Printf("sports:        %s\n", sprintSportStats(p, s.SportCounts))
	p.Printf("period:        %s\n", sprintPeriod(p, s.After, s.Before))
	p.Printf("duration:      %s to %s, average %s, total %s\n", sprintDuration(p, s.MinDuration), sprintDuration(p, s.MaxDuration), sprintDuration(p, avgDur), sprintDuration(p, s.SumDuration))
	p.Printf("distance:      %s to %s, average %s, total %s\n", sprintDistance(p, s.MinDistance), sprintDistance(p, s.MaxDistance), sprintDistance(p, avgDist), sprintDistance(p, s.SumDistance))
	p.Printf("pace:          %s to %s, average %s\n", sprintPace(p, s.MinPace), sprintPace(p, s.MaxPace), sprintPace(p, avgPace))
	p.Printf("bounds:        %s\n", s.BoundedBy)
	p.Printf("starts within: %s\n", s.StartsNear)
	p.Printf("ends within:   %s\n", s.EndsNear)
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

func sprintDuration(p *message.Printer, dur time.Duration) string {
	return p.Sprintf("%s", dur.Truncate(time.Second))
}

func sprintDistance(p *message.Printer, dist float64) string {
	return p.Sprintf("%.1fkm", dist/1000)
}

func sprintPace(p *message.Printer, pace time.Duration) string {
	return p.Sprintf("%s/km", (pace * 1000).Truncate(time.Second))
}
