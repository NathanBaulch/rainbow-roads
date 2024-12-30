package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSportsSet(t *testing.T) {
	testCases := []struct {
		sets   []string
		expect any
	}{
		{[]string{"Running"}, "Running"},
		{[]string{"RUNNING"}, "RUNNING"},
		{[]string{"Cycling", "Running"}, "Cycling,Running"},
		{[]string{"Running,Cycling", "Swimming"}, "Cycling,Running,Swimming"},
		{[]string{""}, errors.New("unexpected empty value")},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			var s SportsFlag
			for _, set := range testCase.sets {
				if err := s.Set(set); err != nil {
					if expectErr, ok := testCase.expect.(error); !ok {
						is.NoError(err)
					} else {
						is.EqualError(err, expectErr.Error())
						return
					}
				}
			}
			is.Equal(testCase.expect, s.String())
		})
	}
}

func TestTimeSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect any
	}{
		{"19 Jan 2022", "2022-01-19 00:00:00 +0000 UTC"},
		{"1645228800", "2022-02-19 00:00:00 +0000 UTC"},
		{"03/19/2022", "2022-03-19 00:00:00 +0000 UTC"},
		{"", errors.New("unexpected empty value")},
		{"foo", errors.New("date not recognized")},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			var f DateFlag
			if err := f.Set(testCase.set); err != nil {
				if expectErr, ok := testCase.expect.(error); !ok {
					is.NoError(err)
				} else {
					is.EqualError(err, expectErr.Error())
				}
			} else {
				is.Equal(testCase.expect, f.String())
			}
		})
	}
}

func TestDurationSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect any
	}{
		{"1h", "1h0m0s"},
		{"1h2m3s", "1h2m3s"},
		{"3600s", "1h0m0s"},
		{"3600", "1h0m0s"},
		{"", errors.New("unexpected empty value")},
		{"foo", errors.New("duration not recognized")},
		{"-1h", errors.New("must be positive")},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			var f DurationFlag
			if err := f.Set(testCase.set); err != nil {
				if expectErr, ok := testCase.expect.(error); !ok {
					is.NoError(err)
				} else {
					is.EqualError(err, expectErr.Error())
				}
			} else {
				is.Equal(testCase.expect, f.String())
			}
		})
	}
}

func TestDistanceSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect any
	}{
		{"3000", "3000"},
		{"3000m", "3000"},
		{"3000 m", "3000"},
		{"3000M", "3000"},
		{"3km", "3000"},
		{"3000ft", "914.4"},
		{"9e9", "9000000000"},
		{"", errors.New("unexpected empty value")},
		{"foo", errors.New("format not recognized")},
		{"f00", errors.New(`number "f00" not recognized`)},
		{"3000x", errors.New(`unit "x" not recognized`)},
		{"3000g", errors.New(`unit "g" not a distance`)},
		{"-3000", errors.New(`must be positive`)},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			var f DistanceFlag
			if err := f.Set(testCase.set); err != nil {
				if expectErr, ok := testCase.expect.(error); !ok {
					is.NoError(err)
				} else {
					is.EqualError(err, expectErr.Error())
				}
			} else {
				is.Equal(testCase.expect, f.String())
			}
		})
	}
}

func TestPaceFlag(t *testing.T) {
	testCases := []struct {
		set    string
		expect any
	}{
		{"1s", "1s"},
		{"1m", "1m0s"},
		{"1s/m", "1s"},
		{"5m/km", "300ms"},
		{"8m/mile", "298.258172ms"},
		{"", errors.New("unexpected empty value")},
		{"/", errors.New("format not recognized")},
		{"foo", errors.New(`duration "foo" not recognized`)},
		{"1s/x", errors.New(`unit "x" not recognized`)},
		{"1s/g", errors.New(`unit "g" not a distance`)},
		{"-1s", errors.New("must be positive")},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			var f PaceFlag
			if err := f.Set(testCase.set); err != nil {
				if expectErr, ok := testCase.expect.(error); !ok {
					is.NoError(err)
				} else {
					is.EqualError(err, expectErr.Error())
				}
			} else {
				is.Equal(testCase.expect, f.String())
			}
		})
	}
}

func TestRegionSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect any
	}{
		{"1,2", "1,2,1000"},
		{"1,2,3", "1,2,3"},
		{"-10.10101,-20.20202,30.30303", "-10.10101,-20.20202,30.30303"},
		{"1,2,3000ft", "1,2,914.4"},
		{"1,2,9e9", "1,2,9000000000"},
		{"", errors.New("unexpected empty value")},
		{"1", errors.New("invalid number of parts")},
		{"1,2,3,4", errors.New("invalid number of parts")},
		{"foo,1", errors.New(`latitude "foo" not recognized`)},
		{"1,foo", errors.New(`longitude "foo" not recognized`)},
		{"1,2,foo", errors.New(`radius format not recognized`)},
		{"1,2,f00", errors.New(`radius number "f00" not recognized`)},
		{"1,2,3000x", errors.New(`radius unit "x" not recognized`)},
		{"1,2,3000g", errors.New(`radius unit "g" not a distance`)},
		{"100,0", errors.New(`latitude "100" not within range`)},
		{"0,200", errors.New(`longitude "200" not within range`)},
		{"1,2,-3", errors.New(`radius must be positive`)},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			is := require.New(t)

			var f CircleFlag
			if err := f.Set(testCase.set); err != nil {
				if expectErr, ok := testCase.expect.(error); !ok {
					is.NoError(err)
				} else {
					is.EqualError(err, expectErr.Error())
				}
			} else {
				is.Equal(testCase.expect, f.String())
			}
		})
	}
}
