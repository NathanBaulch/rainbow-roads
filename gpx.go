package main

import (
	"io"
	"io/ioutil"

	"github.com/tkrajina/gpxgo/gpx"
)

func parseGPX(r io.Reader) error {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	g, err := gpx.ParseBytes(buf)
	if err != nil {
		return err
	}

	if !includeDate(g.Time) {
		return nil
	}

	for _, t := range g.Tracks {
		if !includeSport(t.Type) {
			continue
		}

		for _, s := range t.Segments {
			if len(s.Points) == 0 {
				continue
			}

			p0, p1 := s.Points[0], s.Points[len(s.Points)-1]
			act := &activity{
				date:     g.Time,
				duration: p1.Timestamp.Sub(p0.Timestamp),
				records:  make([]*record, len(s.Points)),
			}
			if !includeDuration(act.duration) {
				continue
			}

			for i, p := range s.Points {
				act.records[i] = &record{
					ts:  p.Timestamp,
					lat: degreesToRadians(p.Latitude),
					lon: degreesToRadians(p.Longitude),
				}
				if i > 0 {
					r0, r1 := act.records[i-1], act.records[i]
					act.distance += haversineDistance(r0.lat, r0.lon, r1.lat, r1.lon)
				}
			}
			if !includeDistance(act.distance) {
				continue
			}
			activities = append(activities, act)
		}
	}

	return nil
}
