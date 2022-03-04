package main

import (
	"bytes"
	"testing"
	"time"
)

func TestGPXStravaTypeCodes(t *testing.T) {
	_ = sports.Set("running")
	activities = activities[:0]
	if err := parseGPX(bytes.NewBufferString(`
		<gpx creator="StravaGPX iPhone">
		  <trk>
		    <type>9</type>
		    <trkseg>
		      <trkpt lat="7.6196940" lon="22.3098920">
		        <time>2022-02-13T00:07:06Z</time>
		      </trkpt>
		    </trkseg>
		  </trk>
		</gpx>`)); err != nil {
		t.Fatal(err)
	}
	if len(activities) != 1 {
		t.Fatal("expected 1 activity")
	}
}

func TestGPXMissingMetadataTime(t *testing.T) {
	sports = sports[:0]
	activities = activities[:0]
	if err := parseGPX(bytes.NewBufferString(`
		<gpx>
		  <trk>
		    <trkseg>
		      <trkpt lat="7.6196940" lon="22.3098920">
		        <time>2022-02-13T00:07:06Z</time>
		      </trkpt>
		    </trkseg>
		  </trk>
		</gpx>`)); err != nil {
		t.Fatal(err)
	}
	if len(activities) != 1 {
		t.Fatal("expected 1 activity")
	}
	if expect := time.Date(2022, 2, 13, 0, 7, 6, 0, time.UTC); activities[0].date != expect {
		t.Fatal("activity date:", activities[0].date, "!=", expect)
	}
}
