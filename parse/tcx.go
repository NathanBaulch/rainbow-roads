package parse

import (
	"io"

	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/llehouerou/go-tcx"
)

func parseTCX(r io.Reader, selector *Selector) ([]*Activity, error) {
	f, err := tcx.Parse(r)
	if err != nil {
		return nil, err
	}

	acts := make([]*Activity, 0, len(f.Activities))

	for _, a := range f.Activities {
		if len(a.Laps) == 0 || !selector.Sport(a.Sport) {
			continue
		}

		act := &Activity{
			Sport:   a.Sport,
			Records: make([]*Record, 0, len(a.Laps[0].Track)),
		}

		var t0, t1 tcx.Trackpoint
		for _, l := range a.Laps {
			if len(l.Track) == 0 {
				continue
			}

			act.Distance += l.DistanceInMeters

			for _, t := range l.Track {
				if t.LatitudeInDegrees == 0 || t.LongitudeInDegrees == 0 {
					continue
				}
				if len(act.Records) == 0 {
					t0 = t
				}
				t1 = t
				act.Records = append(act.Records, &Record{
					Timestamp: t.Time,
					Position:  geo.NewPointFromDegrees(t.LatitudeInDegrees, t.LongitudeInDegrees),
				})
			}
		}

		dur := t1.Time.Sub(t0.Time)
		if len(act.Records) == 0 ||
			!selector.Timestamp(t0.Time, t1.Time) ||
			!selector.Duration(dur) ||
			!selector.Distance(act.Distance) ||
			!selector.Pace(dur, act.Distance) {
			continue
		}

		acts = append(acts, act)
	}

	return acts, nil
}
