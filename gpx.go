package main

import (
	"io"
	"io/ioutil"
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

func parseGPX(r io.Reader) error {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	g, err := gpx.ParseBytes(buf)
	if err != nil {
		return err
	}

	if g.Time != nil && !includeDate(*g.Time) {
		return nil
	}

	for _, t := range g.Tracks {
		sport := t.Type
		if strings.Contains(g.Creator, "Strava") {
			if s, ok := stravaTypeCodes[sport]; ok {
				sport = s
			}
		}
		if !includeSport(sport) {
			continue
		}

		for _, s := range t.Segments {
			if len(s.Points) == 0 {
				continue
			}

			p0, p1 := s.Points[0], s.Points[len(s.Points)-1]
			act := &activity{
				sport:    sport,
				duration: p1.Timestamp.Sub(p0.Timestamp),
			}
			if g.Time != nil {
				act.date = *g.Time
			} else {
				act.date = s.Points[0].Timestamp
			}
			if !includeDate(act.date) ||
				!includeDuration(act.duration) {
				continue
			}

			act.records = make([]*record, len(s.Points))
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
