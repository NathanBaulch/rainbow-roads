package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

func radiansToDegrees(r float64) float64 {
	return r * 180 / math.Pi
}

func semicirclesToRadians(s int32) float64 {
	return float64(s) * math.Pi / math.MaxInt32
}

func mercatorMeters(pt Point) (float64, float64) {
	x := 6_378_137 * pt.lon
	y := 6_378_137 * math.Log(math.Tan((2*pt.lat+math.Pi)/4))
	return x, y
}

func haversineDistance(pt1, pt2 Point) float64 {
	sinLat := math.Sin((pt2.lat - pt1.lat) / 2)
	sinLon := math.Sin((pt2.lon - pt1.lon) / 2)
	a := sinLat*sinLat + math.Cos(pt1.lat)*math.Cos(pt2.lat)*sinLon*sinLon
	return 12_742_000 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func newPointFromDegrees(lat, lon float64) Point {
	return Point{lat: degreesToRadians(lat), lon: degreesToRadians(lon)}
}

func newPointFromSemicircles(lat, lon int32) Point {
	return Point{lat: semicirclesToRadians(lat), lon: semicirclesToRadians(lon)}
}

type Point struct {
	lat, lon float64
}

func (p Point) String() string {
	return fmt.Sprintf("%s,%s", formatFloat(radiansToDegrees(p.lat)), formatFloat(radiansToDegrees(p.lon)))
}

func (p Point) IsZero() bool {
	return p.lat == 0 && p.lon == 0
}

type Circle struct {
	center Point
	radius float64
}

func (c Circle) String() string {
	return fmt.Sprintf("%s,%s", c.center, formatFloat(c.radius))
}

func (c Circle) IsZero() bool {
	return c.radius == 0
}

func (c Circle) Contains(pt Point) bool {
	return haversineDistance(pt, c.center) < c.radius
}

func (c Circle) Enclose(pt Point) Circle {
	c.radius = math.Max(c.radius, haversineDistance(c.center, pt))
	return c
}

type Box struct {
	min, max Point
}

func (b Box) IsZero() bool {
	return b.min.IsZero() && b.max.IsZero()
}

func (b Box) Center() Point {
	return Point{lat: (b.max.lat + b.min.lat) / 2, lon: (b.max.lon + b.min.lon) / 2}
}

func (b Box) Enclose(pt Point) Box {
	if b.IsZero() {
		b.min = pt
		b.max = pt
	} else {
		b.min.lat = math.Min(b.min.lat, pt.lat)
		b.max.lat = math.Max(b.max.lat, pt.lat)
		b.min.lon = math.Min(b.min.lon, pt.lon)
		b.max.lon = math.Max(b.max.lon, pt.lon)
	}
	return b
}

func formatFloat(val float64) string {
	str := strconv.FormatFloat(val, 'f', 5, 64)
	return strings.TrimRight(strings.TrimRight(str, "0"), ".")
}
