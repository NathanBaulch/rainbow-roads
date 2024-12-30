package geo

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"
)

func TestCircleString(t *testing.T) {
	is := require.New(t)

	c := Circle{Origin: orb.Point{1, 2}, Radius: 3000}
	is.Equal("2,1,3000", c.String())
}

func TestCircleContains(t *testing.T) {
	is := require.New(t)

	c := &Circle{Origin: orb.Point{1, 2}, Radius: 3000}
	is.True(c.Contains(orb.Point{1.01, 2.01}))
	is.False(c.Contains(orb.Point{1.02, 2.02}))
}

func TestCircleExtend(t *testing.T) {
	is := require.New(t)

	c := Circle{Origin: orb.Point{1, 2}, Radius: 3000}
	is.Equal("2,1,157177.55181", c.Extend(orb.Point{2, 3}).String())
}

func TestCircleGrow(t *testing.T) {
	is := require.New(t)

	c := Circle{Origin: orb.Point{1, 2}, Radius: 3000}
	is.Equal("2,1,3300", c.Grow(1.1).String())
}
