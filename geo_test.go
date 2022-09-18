package main

import (
	"fmt"
	"math"
	"testing"
)

func TestMercatorMeters(t *testing.T) {
	testCases := []struct {
		lat, lon float64
	}{
		{0, 0},
		{0, -180},
		{0, 180},
		{45, 0},
		{-45, 0},
		{80, 0},
		{-80, 0},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			x, y := mercatorMeters(newPointFromDegrees(testCase.lat, testCase.lon))
			if math.IsNaN(x) {
				t.Fatal("expected x number")
			}
			if math.IsNaN(y) {
				t.Fatal("expected y number")
			}
		})
	}
}
