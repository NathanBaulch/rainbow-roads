package parse

import (
	"bytes"
	"testing"
)

func TestGPXStravaTypeCodes(t *testing.T) {
	if acts, err := parseGPX(bytes.NewBufferString(`
		<gpx creator="StravaGPX iPhone">
		  <trk>
		    <type>1</type>
		    <trkseg>
		      <trkpt lat="7.61969" lon="22.30989">
		        <time>2022-02-13T00:07:06Z</time>
		      </trkpt>
		      <trkpt lat="7.61968" lon="22.30988">
		        <time>2022-02-13T00:07:07Z</time>
		      </trkpt>
		    </trkseg>
		  </trk>
		  <trk>
		    <type>9</type>
		    <trkseg>
		      <trkpt lat="7.61969" lon="22.30989">
		        <time>2022-02-13T00:07:06Z</time>
		      </trkpt>
		      <trkpt lat="7.61968" lon="22.30988">
		        <time>2022-02-13T00:07:07Z</time>
		      </trkpt>
		    </trkseg>
		  </trk>
		</gpx>`), &Selector{Sports: []string{"running"}}); err != nil {
		t.Fatal(err)
	} else if len(acts) != 1 {
		t.Fatal("expected 1 activity")
	}
}
