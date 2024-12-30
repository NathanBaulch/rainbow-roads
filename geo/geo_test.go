package geo

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"
)

func TestCircleString(t *testing.T) {
	is := require.New(t)

	c := Circle{Origin: orb.Point{1, 2}, Radius: 3000}
	is.Equal("circle(2,1,3000)", c.String())
}

func TestCircleContains(t *testing.T) {
	is := require.New(t)

	c := &Circle{Origin: orb.Point{1, 2}, Radius: 3000}
	is.True(c.Contains(orb.Point{1.01, 2.01}))
	is.False(c.Contains(orb.Point{1.02, 2.02}))
}

func TestCircleBound(t *testing.T) {
	is := require.New(t)

	c := Circle{Origin: orb.Point{1, 2}, Radius: 3000}
	is.Equal(orb.Bound{
		Min: orb.Point{0.9730341145864024, 1.9730505414764143},
		Max: orb.Point{1.0269658854135977, 2.026949458523586},
	}, c.Bound())
}

func TestCircleGrow(t *testing.T) {
	is := require.New(t)

	c := Circle{Origin: orb.Point{1, 2}, Radius: 3000}
	is.Equal("circle(2,1,3300)", c.Grow(1.1).String())
}

func TestCircleExtend(t *testing.T) {
	is := require.New(t)

	c := Circle{Origin: orb.Point{1, 2}, Radius: 3000}
	is.Equal("circle(2,1,157177.55181)", c.Extend(orb.Point{2, 3}).String())
}

func TestSquareString(t *testing.T) {
	is := require.New(t)

	c := Square{Origin: orb.Point{1, 2}, Size: 3000, Angle: 4}
	is.Equal("square(2,1,3000,4)", c.String())
}

func TestSquareContains(t *testing.T) {
	is := require.New(t)

	s := NewSquare(orb.Point{1, 2}, 3000, 4)
	is.True(s.Contains(orb.Point{1.01, 2.01}))
	is.False(s.Contains(orb.Point{1.02, 2.02}))
}

func TestSquareBound(t *testing.T) {
	is := require.New(t)

	s := NewSquare(orb.Point{1, 2}, 3000, 4)
	is.Equal(orb.Bound{
		Min: orb.Point{0.9856093785257114, 1.985618144903492},
		Max: orb.Point{1.0143906214742886, 2.014381855096508},
	}, s.Bound())
}

func TestSquareRing(t *testing.T) {
	is := require.New(t)

	s := NewSquare(orb.Point{1, 2}, 3000, 4)
	is.Equal(orb.Ring{
		{0.9856093785257114, 1.9874980440994108},
		{0.9874904236034675, 2.014381855096508},
		{1.0143906214742886, 2.0125019559005892},
		{1.0125095763965324, 1.985618144903492},
	}, s.Ring())
}

func TestSquareGrow(t *testing.T) {
	is := require.New(t)

	s := NewSquare(orb.Point{1, 2}, 3000, 4)
	is.Equal("square(2,1,3300,4)", s.Grow(1.1).String())
}
