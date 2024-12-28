package paint

import (
	"testing"

	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/serjvanilla/go-overpass"
)

func TestPackUnpackWays(t *testing.T) {
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
	if b, err := packWays(in); err != nil {
		t.Fatal(err)
	} else if out, err := unpackWays(b); err != nil {
		t.Fatal(err)
	} else if len(out) != 1 {
		t.Fatalf("ways len %d != %d", len(out), 1)
	} else if len(out[0].Geometry) != 1 {
		t.Fatalf("geometry len %d != %d", len(out[0].Geometry), 1)
	} else if !(geo.Circle{Origin: out[0].Geometry[0], Radius: 0.002}).Contains(geo.NewPointFromDegrees(1, 2)) {
		t.Fatalf("geometry %+v != %+v", out[0].Geometry[0], geo.NewPointFromDegrees(1, 2))
	} else if out[0].Highway != "primary" {
		t.Fatalf("highway %s != %s", out[0].Highway, "primary")
	} else if out[0].Access != "public" {
		t.Fatalf("access %s != %s", out[0].Access, "public")
	} else if out[0].Surface != "paved" {
		t.Fatalf("surface %s != %s", out[0].Surface, "paved")
	}
}
