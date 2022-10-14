package img

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/lucasb-eyer/go-colorful"
)

func TestColorGradientParse(t *testing.T) {
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
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			g := &ColorGradient{}
			if err := g.Parse(testCase.set); err != nil {
				if expectErr, ok := testCase.expect.(error); !ok {
					t.Fatal(err)
				} else if !strings.Contains(err.Error(), expectErr.Error()) {
					t.Fatal(err, "!=", testCase.expect)
				} else {
					return
				}
			}
			actual := g.String()
			if actual != testCase.expect {
				t.Fatal(actual, "!=", testCase.expect)
			}
		})
	}
}

func TestColorGradientColorAt(t *testing.T) {
	g := &ColorGradient{}
	if err := g.Parse("#fff,#ccc,#888,#444,#222,#000"); err != nil {
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
