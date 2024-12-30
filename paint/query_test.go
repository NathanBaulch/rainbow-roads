package paint

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"
)

func TestBuildQuery(t *testing.T) {
	testCases := []struct {
		origin       orb.Point
		radius       float64
		filter, want string
	}{
		{orb.Point{1, 2}, 3, "is_tag(highway)", `[out:json];(way(around:3,2,1)[highway];);out tags geom qt;`},
		{orb.Point{4, 5}, 6, "is_tag(highway) or is_tag(service)", `[out:json];(way(around:6,5,4)[highway];way(around:6,5,4)[service];);out tags geom qt;`},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			got, err := buildQuery(geo.Circle{Origin: tc.origin, Radius: tc.radius}, tc.filter)
			is.NoError(err)

			is.Equal(tc.want, got)
		})
	}
}

func TestBuildCriteria(t *testing.T) {
	testCases := []struct{ input, want string }{
		{"lit", `[lit="yes"]`},
		{"!lit", `[lit="no"]`},
		{"lit == true", `[lit="yes"]`},
		{"not(lit == false)", `[lit!="no"]`},
		{"highway == 'primary'", `[highway="primary"]`},
		{"maxspeed == 50", `[maxspeed=50]`},
		{"max(maxspeed == 50)", `(if:max(maxspeed==50))`},
		{"ref == 2.5", `[ref=2.5]`},
		{"public_transport == 'platform'", `["public_transport"="platform"]`},
		{"power == ''", `[power~"^$"]`},
		{"power != ''", `[power!~"^$"]`},
		{"is_tag(name)", `[name]`},
		{"is_tag('name')", `["name"]`},
		{"!is_tag(name)", `[!name]`},
		{"id() == 4", `(if:id()==4)`},
		{"lit ? 'light' : 'dark'", `(if:lit?"light":"dark")`},
		{"name contains 'Lane'", `[name~"Lane"]`},
		{"name startsWith 'Lane'", `[name~"^Lane"]`},
		{"name endsWith 'Lane'", `[name~"Lane$"]`},
		{"name matches '^L.n.$'", `[name~"^L.n.$"]`},
		{"not(name contains 'Lane')", `[name!~"Lane"]`},
		{"not(name startsWith 'Lane')", `[name!~"^Lane"]`},
		{"not(name endsWith 'Lane')", `[name!~"Lane$"]`},
		{"not(name matches '^L.n.$')", `[name!~"^L.n.$"]`},
		{"max(name contains 'Lane')", `(if:max(name~"Lane"))`},
		{"max(name startsWith 'Lane')", `(if:max(name~"^Lane"))`},
		{"max(name endsWith 'Lane')", `(if:max(name~"Lane$"))`},
		{"max(name matches '^L.n.$')", `(if:max(name~"^L.n.$"))`},
		{"maxspeed > 50", `(if:t["maxspeed"]>50)`},
		{"maxspeed in 50..60", `(if:t["maxspeed"]>=50&&t["maxspeed"]<=60)`},
		{"maxspeed not in 50..60", `(if:t["maxspeed"]<50||t["maxspeed"]>60)`},
		{"'proposed' == highway", `[highway="proposed"]`},
		{"max(!'primary')", `(if:max(!("primary")))`},
		{"!-id()", `(if:!-(id()))`},
		{"is_tag(highway) and maxspeed > 50", `[highway](if:t["maxspeed"]>50)`},
		{"maxspeed > 50 and is_tag(highway)", `(if:t["maxspeed"]>50)[highway]`},
		{"is_tag(highway) and maxspeed > 50 and service != 'driveway'", `[highway](if:t["maxspeed"]>50)[service!="driveway"]`},
		{"maxspeed > 50 and is_tag(highway) and maxspeed < 60", `(if:t["maxspeed"]>50)[highway](if:t["maxspeed"]<60)`},
		{"maxspeed > 50 and maxspeed < 60 and is_tag(highway)", `(if:t["maxspeed"]>50&&t["maxspeed"]<60)[highway]`},
		{"highway not in ['proposed','corridor'] and service != 'driveway'", `[highway!="proposed"][highway!="corridor"][service!="driveway"]`},
		{"highway in ['primary','secondary','tertiary']", `[highway="primary"];[highway="secondary"];[highway="tertiary"]`},
		{"highway in ['primary','secondary','tertiary'] and service == 'driveway'", `[highway="primary"][service="driveway"];[highway="secondary"][service="driveway"];[highway="tertiary"][service="driveway"]`},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			crits, err := buildCriteria(tc.input)
			is.NoError(err)

			is.Equal(tc.want, strings.Join(crits, ";"))
		})
	}
}

func TestBuildCriteriaUnsupported(t *testing.T) {
	testCases := []struct{ input, err string }{
		{"!5", `inverted integer not supported`},
		{"!3.14", `inverted float not supported`},
		{"!'foo'", `inverted string not supported`},
		{"nil", `nil not supported`},
		{"[]", `array not supported`},
		{"{}", `map not supported`},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			_, err := buildCriteria(tc.input)
			is.EqualError(err, tc.err)
		})
	}
}
