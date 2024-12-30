package parse

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGPXStravaTypeCodes(t *testing.T) {
	is := require.New(t)

	acts, err := parseGPX(bytes.NewBufferString(`
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
		</gpx>`), &Selector{Sports: []string{"running"}})
	is.NoError(err)
	is.Len(acts, 1)
}
