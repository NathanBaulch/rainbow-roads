package main

import (
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
		x, y := mercatorMeters(degreesToRadians(testCase.lat), degreesToRadians(testCase.lon))
		if math.IsNaN(x) {
			t.Fatal("test case", i, "failed: expected x number")
		}
		if math.IsNaN(y) {
			t.Fatal("test case", i, "failed: expected y number")
		}
	}
}
