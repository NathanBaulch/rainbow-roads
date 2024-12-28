package parse

import (
	"errors"
	"io"
	"strings"

	"github.com/paulmach/orb"
	"github.com/tormoder/fit"
)

func parseFIT(r io.Reader, selector *Selector) ([]*Activity, error) {
	f, err := fit.Decode(r)
	if err != nil {
		var ferr fit.FormatError
		if errors.As(err, &ferr) {
			return nil, nil
		}
		return nil, err
	}

	if a, err := f.Activity(); err != nil {
		if strings.HasPrefix(err.Error(), "fit file type is ") {
			return nil, nil
		}
		return nil, err
	} else if len(a.Records) == 0 {
		return nil, nil
	} else {
		act := &Activity{
			Sport:    a.Sessions[0].Sport.String(),
			Distance: a.Sessions[0].GetTotalDistanceScaled(),
		}
		r0, r1 := a.Records[0], a.Records[len(a.Records)-1]
		dur := r1.Timestamp.Sub(r0.Timestamp)
		if !selector.Sport(act.Sport) ||
			!selector.Timestamp(r0.Timestamp, r1.Timestamp) ||
			!selector.Duration(dur) ||
			!selector.Distance(act.Distance) ||
			!selector.Pace(dur, act.Distance) {
			return nil, nil
		}

		act.Records = make([]*Record, 0, len(a.Records))
		for _, rec := range a.Records {
			if !rec.PositionLat.Invalid() && !rec.PositionLong.Invalid() {
				act.Records = append(act.Records, &Record{
					Timestamp: rec.Timestamp,
					Position:  orb.Point{rec.PositionLong.Degrees(), rec.PositionLat.Degrees()},
				})
			}
		}
		if len(act.Records) == 0 {
			return nil, nil
		}
		return []*Activity{act}, nil
	}
}
