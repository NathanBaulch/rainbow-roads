package main

import (
	"io"
	"time"

	"github.com/llehouerou/go-tcx"
)

func parseTCX(r io.Reader) error {
	f, err := tcx.Parse(r)
	if err != nil {
		return err
	}

	for _, a := range f.Activities {
		if !includeSport(a.Sport) {
			continue
		}

		for _, l := range a.Laps {
			if len(l.Track) == 0 ||
				!includeDate(l.StartTime) ||
				!includeDuration(time.Duration(l.TotalTimeInSeconds)*time.Second) ||
				!includeDistance(l.DistanceInMeters) {
				continue
			}

			act := &activity{
				date:     l.StartTime,
				duration: time.Duration(l.TotalTimeInSeconds) * time.Second,
				distance: l.DistanceInMeters,
				records:  make([]*record, 0, len(l.Track)),
			}
			for _, t := range l.Track {
				if t.LatitudeInDegrees != 0 && t.LongitudeInDegrees != 0 {
					act.records = append(act.records, &record{
						ts:  t.Time,
						lat: degreesToRadians(t.LatitudeInDegrees),
						lon: degreesToRadians(t.LongitudeInDegrees),
					})
				}
			}
			if len(act.records) > 0 {
				activities = append(activities, act)
			}
		}
	}

	return nil
}
