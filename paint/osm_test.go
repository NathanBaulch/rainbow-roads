package paint

import (
	"testing"

	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/paulmach/orb"
	"github.com/serjvanilla/go-overpass"
	"github.com/stretchr/testify/require"
)

func TestPackUnpackWays(t *testing.T) {
	is := require.New(t)

	in := map[int64]*overpass.Way{
		0: {
			Meta: overpass.Meta{
				Tags: map[string]string{
					"highway": "primary",
					"access":  "public",
					"surface": "paved",
				},
			},
			Geometry: []overpass.Point{{Lat: 1, Lon: 2}},
		},
	}
	b, err := packWays(in)
	is.NoError(err)
	out, err := unpackWays(b)
	is.NoError(err)

	is.Len(out, 1)
	is.Len(out[0].Geometry, 1)
	is.True((geo.Circle{Origin: out[0].Geometry[0], Radius: 0.002}).Contains(orb.Point{2, 1}))
	is.Equal("primary", out[0].Highway)
	is.Equal("public", out[0].Access)
	is.Equal("paved", out[0].Surface)
}
