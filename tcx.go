package main

import (
	"io"
	"time"

	"github.com/llehouerou/go-tcx"
)

func parseTCX(r io.Reader) ([]*activity, error) {
	f, err := tcx.Parse(r)
	if err != nil {
		return nil, err
	}

	acts := make([]*activity, 0, len(f.Activities))

	for _, a := range f.Activities {
		if len(a.Laps) == 0 || !includeSport(a.Sport) {
			continue
		}

		act := &activity{
			sport:   a.Sport,
			records: make([]*record, 0, len(a.Laps[0].Track)),
		}

		var t0, t1 tcx.Trackpoint
		for _, l := range a.Laps {
			if len(l.Track) == 0 {
				continue
			}

			act.distance += l.DistanceInMeters

			for _, t := range l.Track {
				if t.LatitudeInDegrees == 0 || t.LongitudeInDegrees == 0 {
					continue
				}
				if len(act.records) == 0 {
					t0 = t
				}
				t1 = t
				act.records = append(act.records, &record{
					ts: t.Time,
					pt: newPointFromDegrees(t.LatitudeInDegrees, t.LongitudeInDegrees),
				})
			}
		}

		dur := t1.Time.Sub(t0.Time)
		if len(act.records) == 0 ||
			!includeTimestamp(t0.Time, t1.Time) ||
			!includeDuration(dur) ||
			!includeDistance(act.distance) ||
			!includePace(dur/time.Duration(act.distance)) {
			continue
		}

		acts = append(acts, act)
	}

	return acts, nil
}
