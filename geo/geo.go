package geo

import (
	"fmt"
	"math"

	"github.com/NathanBaulch/rainbow-roads/conv"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/orb/planar"
)

const (
	earthRadiusEquatorial = 6_378_137
	earthRadiusMean       = 6_371_000
)

func DegreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

func RadiansToDegrees(r float64) float64 {
	return r * 180 / math.Pi
}

func DistanceHaversine(p1, p2 orb.Point) float64 {
	sinLat := math.Sin(DegreesToRadians(p1[1]-p2[1]) / 2)
	sinLon := math.Sin(DegreesToRadians(p1[0]-p2[0]) / 2)
	a := sinLat*sinLat + math.Cos(DegreesToRadians(p2[1]))*math.Cos(DegreesToRadians(p1[1]))*sinLon*sinLon
	return 2.0 * earthRadiusMean * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

type Geometry interface {
	String() string
	Contains(pt orb.Point) bool
	Bound() orb.Bound
	Ring() orb.Ring
	Grow(factor float64) Geometry
}

type Circle struct {
	Origin orb.Point
	Radius float64
}

func (c Circle) String() string {
	return fmt.Sprintf("circle(%s,%s,%s)", conv.FormatFloat(c.Origin.Lat()), conv.FormatFloat(c.Origin.Lon()), conv.FormatFloat(c.Radius))
}

func (c Circle) Contains(pt orb.Point) bool {
	return DistanceHaversine(c.Origin, pt) < c.Radius
}

func (c Circle) Bound() orb.Bound {
	return geo.NewBoundAroundPoint(c.Origin, c.Radius)
}

func (c Circle) Ring() orb.Ring {
	return orb.Ring{}
}

func (c Circle) Grow(factor float64) Geometry {
	c.Radius *= factor
	return c
}

func (c Circle) Extend(pt orb.Point) Circle {
	c.Radius = math.Max(c.Radius, DistanceHaversine(c.Origin, pt))
	return c
}

func NewSquare(origin orb.Point, size, angle float64) Square {
	return (Square{
		Origin: origin,
		Size:   size,
		Angle:  angle,
	}).computeRing()
}

type Square struct {
	Origin orb.Point
	Size   float64
	Angle  float64
	ring   orb.Ring
}

func (s Square) String() string {
	return fmt.Sprintf("square(%s,%s,%s,%s)", conv.FormatFloat(s.Origin.Lat()), conv.FormatFloat(s.Origin.Lon()), conv.FormatFloat(s.Size), conv.FormatFloat(s.Angle))
}

func (s Square) Contains(pt orb.Point) bool {
	return planar.RingContains(s.ring, pt)
}

func (s Square) Bound() orb.Bound {
	return s.ring.Bound()
}

func (s Square) Ring() orb.Ring {
	return s.ring
}

func (s Square) Grow(factor float64) Geometry {
	s.Size *= factor
	return s.computeRing()
}

func (s Square) computeRing() Square {
	halfWidth := s.Size / 2
	s.ring = orb.Ring{
		{-halfWidth, -halfWidth},
		{-halfWidth, halfWidth},
		{halfWidth, halfWidth},
		{halfWidth, -halfWidth},
	}
	cosLat := math.Cos(DegreesToRadians(s.Origin.Lat()))
	angle := DegreesToRadians(-s.Angle)
	cosAngle := math.Cos(angle)
	sinAngle := math.Sin(angle)
	for i, pt := range s.ring {
		s.ring[i][0] = s.Origin.Lon() + RadiansToDegrees((pt[0]*cosAngle-pt[1]*sinAngle)/(earthRadiusEquatorial*cosLat))
		s.ring[i][1] = s.Origin.Lat() + RadiansToDegrees((pt[0]*sinAngle+pt[1]*cosAngle)/earthRadiusEquatorial)
	}
	return s
}

var BoundWidth = geo.BoundWidth
