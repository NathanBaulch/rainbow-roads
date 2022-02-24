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

func radiansToDegrees(d float64) float64 {
	return d * 180 / math.Pi
}

func semicirclesToRadians(s int32) float64 {
	return float64(s) * math.Pi / math.MaxInt32
}

func mercatorMeters(lat, lon float64) (float64, float64) {
	x := 6_378_137 * lon
	y := 6_378_137 * math.Log(math.Tan(lat+(math.Pi/4))) / (2 * math.Pi)
	return x, y
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	sinLat := math.Sin((lat2 - lat1) / 2)
	sinLon := math.Sin((lon2 - lon1) / 2)
	a := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 12_742_000 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

type Region struct {
	lat, lon, radius float64
}

func (r *Region) String() string {
	return fmt.Sprintf("%s,%s,%s", formatFloat(radiansToDegrees(r.lat)), formatFloat(radiansToDegrees(r.lon)), formatFloat(r.radius))
}

func (r *Region) Contains(lat, lon float64) bool {
	return haversineDistance(lat, lon, r.lat, r.lon) < r.radius
}

func (r *Region) IsZero() bool {
	return r.radius == 0
}

func formatFloat(val float64) string {
	str := strconv.FormatFloat(val, 'f', 5, 64)
	return strings.TrimRight(strings.TrimRight(str, "0"), ".")
}
