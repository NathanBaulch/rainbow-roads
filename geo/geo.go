package geo

import (
	"fmt"
	"math"

	"github.com/NathanBaulch/rainbow-roads/conv"
)

const (
	mercatorRadius  = 6_378_137
	haversineRadius = 6_371_000
)

func DegreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

func RadiansToDegrees(r float64) float64 {
	return r * 180 / math.Pi
}

func SemicirclesToRadians(s int32) float64 {
	return float64(s) * math.Pi / math.MaxInt32
}

func NewPointFromDegrees(lat, lon float64) Point {
	return Point{Lat: DegreesToRadians(lat), Lon: DegreesToRadians(lon)}
}

func NewPointFromSemicircles(lat, lon int32) Point {
	return Point{Lat: SemicirclesToRadians(lat), Lon: SemicirclesToRadians(lon)}
}

type Point struct {
	Lat, Lon float64
}

func (p Point) String() string {
	return fmt.Sprintf("%s,%s", conv.FormatFloat(RadiansToDegrees(p.Lat)), conv.FormatFloat(RadiansToDegrees(p.Lon)))
}

func (p Point) IsZero() bool {
	return p.Lat == 0 && p.Lon == 0
}

func (p Point) DistanceTo(pt Point) float64 {
	sinLat := math.Sin((pt.Lat - p.Lat) / 2)
	sinLon := math.Sin((pt.Lon - p.Lon) / 2)
	a := sinLat*sinLat + math.Cos(p.Lat)*math.Cos(pt.Lat)*sinLon*sinLon
	return 2 * haversineRadius * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func (p Point) MercatorProjection() (float64, float64) {
	x := mercatorRadius * p.Lon
	y := mercatorRadius * math.Log(math.Tan((2*p.Lat+math.Pi)/4))
	return x, y
}

type Circle struct {
	Origin Point
	Radius float64
}

func (c Circle) String() string {
	return fmt.Sprintf("%s,%s", c.Origin, conv.FormatFloat(c.Radius))
}

func (c Circle) IsZero() bool {
	return c.Radius == 0
}

func (c Circle) Contains(pt Point) bool {
	return c.Origin.DistanceTo(pt) < c.Radius
}

func (c Circle) Enclose(pt Point) Circle {
	c.Radius = math.Max(c.Radius, c.Origin.DistanceTo(pt))
	return c
}

func (c Circle) Grow(factor float64) Circle {
	c.Radius *= factor
	return c
}

type Box struct {
	Min, Max Point
}

func (b Box) IsZero() bool {
	return b.Min.IsZero() && b.Max.IsZero()
}

func (b Box) Center() Point {
	return Point{Lat: (b.Max.Lat + b.Min.Lat) / 2, Lon: (b.Max.Lon + b.Min.Lon) / 2}
}

func (b Box) Enclose(pt Point) Box {
	if b.IsZero() {
		b.Min = pt
		b.Max = pt
	} else {
		b.Min.Lat = math.Min(b.Min.Lat, pt.Lat)
		b.Max.Lat = math.Max(b.Max.Lat, pt.Lat)
		b.Min.Lon = math.Min(b.Min.Lon, pt.Lon)
		b.Max.Lon = math.Max(b.Max.Lon, pt.Lon)
	}
	return b
}
