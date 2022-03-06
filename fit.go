package main

import (
	"io"

	"github.com/tormoder/fit"
)

func parseFIT(r io.Reader) error {
	f, err := fit.Decode(r)
	if err != nil {
		if _, ok := err.(fit.FormatError); ok {
			return nil
		}
		return err
	}

	if a, err := f.Activity(); err != nil || len(a.Records) == 0 {
		return nil
	} else {
		act := &activity{
			sport:    a.Sessions[0].Sport.String(),
			distance: a.Sessions[0].GetTotalDistanceScaled(),
		}
		r0, r1 := a.Records[0], a.Records[len(a.Records)-1]
		if !includeSport(act.sport) ||
			!includeTimestamp(r0.Timestamp, r1.Timestamp) ||
			!includeDuration(r1.Timestamp.Sub(r0.Timestamp)) ||
			!includeDistance(act.distance) {
			return nil
		}
		act.records = make([]*record, 0, len(a.Records))
		for _, r := range a.Records {
			if !r.PositionLat.Invalid() && !r.PositionLong.Invalid() {
				act.records = append(act.records, &record{
					ts:  r.Timestamp,
					lat: semicirclesToRadians(r.PositionLat.Semicircles()),
					lon: semicirclesToRadians(r.PositionLong.Semicircles()),
				})
			}
		}
		if len(act.records) > 0 {
			activities = append(activities, act)
		}
		return nil
	}
}
