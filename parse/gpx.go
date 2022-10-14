package parse

import (
	"io"
	"strings"

	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/tkrajina/gpxgo/gpx"
)

var stravaTypeCodes = map[string]string{
	"1":  "Cycling",
	"2":  "AlpineSkiing",
	"3":  "BackcountrySkiing",
	"4":  "Hiking",
	"5":  "IceSkating",
	"6":  "InlineSkating",
	"7":  "CrossCountrySkiing",
	"8":  "RollerSkiing",
	"9":  "Running",
	"10": "Walking",
	"11": "Workout",
	"12": "Snowboarding",
	"13": "Snowshoeing",
	"14": "Kitesurfing",
	"15": "Windsurfing",
	"16": "Swimming",
	"17": "VirtualBiking",
	"18": "EBiking",
	"19": "Velomobile",
	"21": "Paddling",
	"22": "Kayaking",
	"23": "Rowing",
	"24": "StandUpPaddling",
	"25": "Surfing",
	"26": "Crossfit",
	"27": "Elliptical",
	"28": "RockClimbing",
	"29": "StairStepper",
	"30": "WeightTraining",
	"31": "Yoga",
	"51": "Handcycling",
	"52": "Wheelchair",
	"53": "VirtualRunning",
}

func parseGPX(r io.Reader, selector *Selector) ([]*Activity, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	g, err := gpx.ParseBytes(buf)
	if err != nil {
		return nil, err
	}

	acts := make([]*Activity, 0, len(g.Tracks))

	for _, t := range g.Tracks {
		sport := t.Type
		if strings.Contains(g.Creator, "Strava") {
			if s, ok := stravaTypeCodes[sport]; ok {
				sport = s
			}
		}
		if len(t.Segments) == 0 || !selector.Sport(sport) {
			continue
		}

		act := &Activity{
			Sport:   sport,
			Records: make([]*Record, 0, len(t.Segments[0].Points)),
		}

		var p0, p1 gpx.GPXPoint
		for _, s := range t.Segments {
			if len(s.Points) == 0 {
				continue
			}

			for i, p := range s.Points {
				if len(act.Records) == 0 {
					p0 = p
				}
				p1 = p
				act.Records = append(act.Records, &Record{
					Timestamp: p.Timestamp,
					Position:  geo.NewPointFromDegrees(p.Latitude, p.Longitude),
				})
				if i > 0 {
					act.Distance += act.Records[i-1].Position.DistanceTo(act.Records[i].Position)
				}
			}
		}

		dur := p1.Timestamp.Sub(p0.Timestamp)
		if len(act.Records) == 0 ||
			!selector.Timestamp(p0.Timestamp, p1.Timestamp) ||
			!selector.Duration(dur) ||
			!selector.Distance(act.Distance) ||
			!selector.Pace(dur, act.Distance) {
			continue
		}

		acts = append(acts, act)
	}

	return acts, nil
}
