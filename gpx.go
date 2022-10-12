package main

import (
	"io"
	"strings"

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

func parseGPX(r io.Reader) ([]*activity, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	g, err := gpx.ParseBytes(buf)
	if err != nil {
		return nil, err
	}

	acts := make([]*activity, 0, len(g.Tracks))

	for _, t := range g.Tracks {
		sport := t.Type
		if strings.Contains(g.Creator, "Strava") {
			if s, ok := stravaTypeCodes[sport]; ok {
				sport = s
			}
		}
		if len(t.Segments) == 0 || !includeSport(sport) {
			continue
		}

		act := &activity{
			sport:   sport,
			records: make([]*record, 0, len(t.Segments[0].Points)),
		}

		var p0, p1 gpx.GPXPoint
		for _, s := range t.Segments {
			if len(s.Points) == 0 {
				continue
			}

			for i, p := range s.Points {
				if len(act.records) == 0 {
					p0 = p
				}
				p1 = p
				act.records = append(act.records, &record{
					ts: p.Timestamp,
					pt: newPointFromDegrees(p.Latitude, p.Longitude),
				})
				if i > 0 {
					act.distance += haversineDistance(act.records[i-1].pt, act.records[i].pt)
				}
			}
		}

		dur := p1.Timestamp.Sub(p0.Timestamp)
		if len(act.records) == 0 ||
			!includeTimestamp(p0.Timestamp, p1.Timestamp) ||
			!includeDuration(dur) ||
			!includeDistance(act.distance) ||
			!includePace(dur, act.distance) {
			continue
		}

		acts = append(acts, act)
	}

	return acts, nil
}
