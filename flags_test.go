package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/lucasb-eyer/go-colorful"
)

func TestFormatSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect interface{}
	}{
		{"gif", "gif"},
		{"GIF", "gif"},
		{"", errors.New("unexpected empty value")},
		{"foo", errors.New("invalid value")},
	}

	for i, testCase := range testCases {
		f := NewFormatFlag("gif")
		if err := f.Set(testCase.set); err != nil {
			if expectErr, ok := testCase.expect.(error); !ok {
				t.Fatal("test case", i, "error:", err)
			} else if !strings.Contains(err.Error(), expectErr.Error()) {
				t.Fatal("test case", i, "failed:", err, "!=", testCase.expect)
			} else {
				continue
			}
		}
		actual := f.String()
		if actual != testCase.expect {
			t.Fatal("test case", i, "failed:", actual, "!=", testCase.expect)
		}
	}
}

func TestColorsSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect interface{}
	}{
		{"#fff", "#fff"},
		{"#fff,#000", "#fff,#000"},
		{"#123456,#789abc", "#123456,#789abc"},
		{"#fff,#888,#000", "#fff,#888@0.5,#000"},
		{"#fff,#ccc,#666,#333,#000", "#fff,#ccc@0.25,#666@0.5,#333@0.75,#000"},
		{"#fff,#999@.7,#888,#777,#000", "#fff,#999@0.7,#888@0.8,#777@0.9,#000"},
		{"#fff,#aaa,#999@.7,#888,#777,#000", "#fff,#aaa@0.35,#999@0.7,#888@0.8,#777@0.9,#000"},
		{"#fff@.1,#000@.9", "#fff@0.1,#000@0.9"},
		{"#ABCDEF", "#abcdef"},
		{"red,green,blue", "#f00,#008000@0.5,#00f"},
		{"red@11.1%", "#f00@0.111"},
		{"", errors.New("unexpected empty value")},
		{",#fff", errors.New("unexpected empty entry")},
		{"#fff,", errors.New("unexpected empty entry")},
		{"foo", errors.New(`color "foo" not recognized`)},
		{"#fff@foo", errors.New(`position "foo" not recognized`)},
		{"#fff@foo%", errors.New(`position "foo%" not recognized`)},
		{"#fff@9e9", errors.New(`position "9e9" not within range`)},
		{"#fff@101%", errors.New(`position "101%" not within range`)},
	}

	for i, testCase := range testCases {
		g := &ColorsFlag{}
		if err := g.Set(testCase.set); err != nil {
			if expectErr, ok := testCase.expect.(error); !ok {
				t.Fatal("test case", i, "error:", err)
			} else if !strings.Contains(err.Error(), expectErr.Error()) {
				t.Fatal("test case", i, "failed:", err, "!=", testCase.expect)
			} else {
				continue
			}
		}
		actual := g.String()
		if actual != testCase.expect {
			t.Fatal("test case", i, "failed:", actual, "!=", testCase.expect)
		}
	}
}

func TestColorsAt(t *testing.T) {
	g := &ColorsFlag{}
	if err := g.Set("#fff,#ccc,#888,#444,#222,#000"); err != nil {
		t.Fatal(err)
	}
	for p, expect := range map[float64]string{
		0.0: "#ffffff",
		0.1: "#e5e5e5",
		0.2: "#cccccc",
		0.3: "#a9a9a9",
		0.4: "#888888",
		0.5: "#656565",
		0.6: "#444444",
		0.7: "#333333",
		0.8: "#222222",
		0.9: "#151515",
		1.0: "#000000",
	} {
		actual := g.GetColorAt(p).(colorful.Color).Hex()
		if actual != expect {
			t.Fatal("palette color at ", p, ":", actual, "!=", expect)
		}
	}
}

func TestSportsSet(t *testing.T) {
	testCases := []struct {
		sets   []string
		expect interface{}
	}{
		{[]string{"Running"}, "Running"},
		{[]string{"RUNNING"}, "RUNNING"},
		{[]string{"Cycling", "Running"}, "Cycling,Running"},
		{[]string{"Running,Cycling", "Swimming"}, "Cycling,Running,Swimming"},
		{[]string{""}, errors.New("unexpected empty value")},
	}

	for i, testCase := range testCases {
		s := &SportsFlag{}
		for _, set := range testCase.sets {
			if err := s.Set(set); err != nil {
				if expectErr, ok := testCase.expect.(error); !ok {
					t.Fatal("test case", i, "error:", err)
				} else if !strings.Contains(err.Error(), expectErr.Error()) {
					t.Fatal("test case", i, "failed:", err, "!=", testCase.expect)
				} else {
					s = nil
					break
				}
			}
		}
		if s == nil {
			continue
		}
		actual := s.String()
		if actual != testCase.expect {
			t.Fatal("test case", i, "failed:", actual, "!=", testCase.expect)
		}
	}
}

func TestTimeSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect interface{}
	}{
		{"19 Jan 2022", "2022-01-19 00:00:00 +0000 UTC"},
		{"1645228800", "2022-02-19 00:00:00 +0000 UTC"},
		{"03/19/2022", "2022-03-19 00:00:00 +0000 UTC"},
		{"", errors.New("unexpected empty value")},
		{"foo", errors.New("date not recognized")},
	}

	for i, testCase := range testCases {
		f := &DateFlag{}
		if err := f.Set(testCase.set); err != nil {
			if expectErr, ok := testCase.expect.(error); !ok {
				t.Fatal("test case", i, "error:", err)
			} else if !strings.Contains(err.Error(), expectErr.Error()) {
				t.Fatal("test case", i, "failed:", err, "!=", testCase.expect)
			} else {
				continue
			}
		}
		actual := f.String()
		if actual != testCase.expect {
			t.Fatal("test case", i, "failed:", actual, "!=", testCase.expect)
		}
	}
}

func TestDurationSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect interface{}
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
		d := &DurationFlag{}
		if err := d.Set(testCase.set); err != nil {
			if expectErr, ok := testCase.expect.(error); !ok {
				t.Fatal("test case", i, "error:", err)
			} else if !strings.Contains(err.Error(), expectErr.Error()) {
				t.Fatal("test case", i, "failed:", err, "!=", testCase.expect)
			} else {
				continue
			}
		}
		actual := d.String()
		if actual != testCase.expect {
			t.Fatal("test case", i, "failed:", actual, "!=", testCase.expect)
		}
	}
}

func TestDistanceSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect interface{}
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
		var f DistanceFlag
		d := &f
		if err := d.Set(testCase.set); err != nil {
			if expectErr, ok := testCase.expect.(error); !ok {
				t.Fatal("test case", i, "error:", err)
			} else if !strings.Contains(err.Error(), expectErr.Error()) {
				t.Fatal("test case", i, "failed:", err, "!=", testCase.expect)
			} else {
				continue
			}
		}
		actual := d.String()
		if actual != testCase.expect {
			t.Fatal("test case", i, "failed:", actual, "!=", testCase.expect)
		}
	}
}

func TestPaceFlag(t *testing.T) {
	testCases := []struct {
		set    string
		expect interface{}
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
		d := &PaceFlag{}
		if err := d.Set(testCase.set); err != nil {
			if expectErr, ok := testCase.expect.(error); !ok {
				t.Fatal("test case", i, "error:", err)
			} else if !strings.Contains(err.Error(), expectErr.Error()) {
				t.Fatal("test case", i, "failed:", err, "!=", testCase.expect)
			} else {
				continue
			}
		}
		actual := d.String()
		if actual != testCase.expect {
			t.Fatal("test case", i, "failed:", actual, "!=", testCase.expect)
		}
	}
}

func TestRegionSet(t *testing.T) {
	testCases := []struct {
		set    string
		expect interface{}
	}{
		{"1,2", "1,2,100"},
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
		r := &RegionFlag{}
		if err := r.Set(testCase.set); err != nil {
			if expectErr, ok := testCase.expect.(error); !ok {
				t.Fatal("test case", i, "error:", err)
			} else if !strings.Contains(err.Error(), expectErr.Error()) {
				t.Fatal("test case", i, "failed:", err, "!=", testCase.expect)
			} else if !r.IsZero() {
				t.Fatal("test case", i, "expected zero")
			} else {
				continue
			}
		}
		actual := r.String()
		if actual != testCase.expect {
			t.Fatal("test case", i, "failed:", actual, "!=", testCase.expect)
		} else if r.IsZero() {
			t.Fatal("test case", i, "expected not zero")
		}
	}
}

func TestRegionContains(t *testing.T) {
	r := &RegionFlag{}
	if err := r.Set("1,2,3"); err != nil {
		t.Fatal(err)
	}
	if !r.Contains(0.0174536, 0.0349068) {
		t.Fatal("expected contains")
	}
	if r.Contains(0.0174537, 0.0349069) {
		t.Fatal("expected not contains")
	}
}
