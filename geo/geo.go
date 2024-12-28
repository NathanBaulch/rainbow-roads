package geo

import (
	"fmt"
	"math"

	"github.com/NathanBaulch/rainbow-roads/conv"
	"github.com/paulmach/orb"
)

func DegreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

func DistanceHaversine(p1, p2 orb.Point) float64 {
	sinLat := math.Sin(DegreesToRadians(p1[1]-p2[1]) / 2)
	sinLon := math.Sin(DegreesToRadians(p1[0]-p2[0]) / 2)
	a := sinLat*sinLat + math.Cos(DegreesToRadians(p2[1]))*math.Cos(DegreesToRadians(p1[1]))*sinLon*sinLon
	return 2.0 * 6_371_000 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

type Circle struct {
	Origin orb.Point
	Radius float64
}

func (c Circle) String() string {
	return fmt.Sprintf("%s,%s,%s", conv.FormatFloat(c.Origin.Lat()), conv.FormatFloat(c.Origin.Lon()), conv.FormatFloat(c.Radius))
}

func (c Circle) IsZero() bool {
	return c.Radius == 0
}

func (c Circle) Contains(pt orb.Point) bool {
	return DistanceHaversine(c.Origin, pt) < c.Radius
}

func (c Circle) Extend(pt orb.Point) Circle {
	c.Radius = math.Max(c.Radius, DistanceHaversine(c.Origin, pt))
	return c
}

func (c Circle) Grow(factor float64) Circle {
	c.Radius *= factor
	return c
}
