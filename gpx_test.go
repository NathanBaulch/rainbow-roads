package main

import (
	"bytes"
	"testing"
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
