package main

import (
	"io"

	"github.com/tormoder/fit"
)

func parseFIT(r io.Reader) ([]*activity, error) {
	f, err := fit.Decode(r)
	if err != nil {
		if _, ok := err.(fit.FormatError); ok {
			return nil, nil
		}
		return nil, err
	}

	if a, err := f.Activity(); err != nil || len(a.Records) == 0 {
		return nil, nil
	} else {
		act := &activity{
			sport:    a.Sessions[0].Sport.String(),
			distance: a.Sessions[0].GetTotalDistanceScaled(),
		}
		r0, r1 := a.Records[0], a.Records[len(a.Records)-1]
		dur := r1.Timestamp.Sub(r0.Timestamp)
		if !includeSport(act.sport) ||
			!includeTimestamp(r0.Timestamp, r1.Timestamp) ||
			!includeDuration(dur) ||
			!includeDistance(act.distance) ||
			!includePace(dur, act.distance) {
			return nil, nil
		}
		act.records = make([]*record, 0, len(a.Records))
		for _, rec := range a.Records {
			if !rec.PositionLat.Invalid() && !rec.PositionLong.Invalid() {
				act.records = append(act.records, &record{
					ts: rec.Timestamp,
					pt: newPointFromSemicircles(rec.PositionLat.Semicircles(), rec.PositionLong.Semicircles()),
				})
			}
		}
		if len(act.records) == 0 {
			return nil, nil
		}
		return []*activity{act}, nil
	}
}
