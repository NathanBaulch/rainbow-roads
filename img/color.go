package img

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/NathanBaulch/rainbow-roads/conv"
	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/image/colornames"
)

type ColorGradient []struct {
	colorful.Color
	pos float64
}

func (c *ColorGradient) Parse(str string) error {
	if str == "" {
		return errors.New("unexpected empty value")
	}

	parts := strings.Split(str, ",")
	*c = make(ColorGradient, len(parts))
	var err error
	missingAt := 0

	for i, part := range parts {
		if part == "" {
			return errors.New("unexpected empty entry")
		}

		e := &(*c)[i]
		if pos := strings.Index(part, "@"); pos >= 0 {
			if strings.HasSuffix(part, "%") {
				if p, err := strconv.ParseFloat(part[pos+1:len(part)-1], 64); err != nil {
					return fmt.Errorf("position %q not recognized", part[pos+1:])
				} else {
					e.pos = p / 100
				}
			} else {
				if e.pos, err = strconv.ParseFloat(part[pos+1:], 64); err != nil {
					return fmt.Errorf("position %q not recognized", part[pos+1:])
				}
			}
			if e.pos < 0 || e.pos > 1 {
				return fmt.Errorf("position %q not within range", part[pos+1:])
			}
			part = part[:pos]
		} else if i == 0 {
			e.pos = 0
		} else if i == len(parts)-1 {
			e.pos = 1
		} else {
			e.pos = math.NaN()
			if missingAt == 0 {
				missingAt = i
			}
		}
		if !math.IsNaN(e.pos) && missingAt > 0 {
			p := (*c)[missingAt-1].pos
			step := (e.pos - p) / float64(i+1-missingAt)
			for j := missingAt; j < i; j++ {
				p += step
				(*c)[j].pos = p
			}
			missingAt = 0
		}

		if e.Color, err = colorful.Hex(part); err != nil {
			if col, ok := colornames.Map[strings.ToLower(part)]; !ok {
				return fmt.Errorf("color %q not recognized", part)
			} else {
				e.Color, _ = colorful.MakeColor(col)
			}
		}
	}

	return nil
}

func (c *ColorGradient) String() string {
	parts := make([]string, len(*c))
	for i, e := range *c {
		var hex string
		if r, g, b := e.Color.RGB255(); r>>4 == r&0xf && g>>4 == g&0xf && b>>4 == b&0xf {
			hex = fmt.Sprintf("#%1x%1x%1x", r&0xf, g&0xf, b&0xf)
		} else {
			hex = fmt.Sprintf("#%02x%02x%02x", r, g, b)
		}
		if (i == 0 && e.pos == 0) || (i == len(*c)-1 && e.pos == 1) {
			parts[i] = hex
		} else {
			parts[i] = fmt.Sprintf("%s@%s", hex, conv.FormatFloat(e.pos))
		}
	}
	return strings.Join(parts, ",")
}

func (c *ColorGradient) GetColorAt(p float64) color.Color {
	last := len(*c) - 1
	for i := 0; i < last; i++ {
		if e0, e1 := (*c)[i], (*c)[i+1]; e0.pos <= p && p <= e1.pos {
			return e0.Color.BlendHcl(e1.Color, (p-e0.pos)/(e1.pos-e0.pos)).Clamped()
		}
	}
	return (*c)[last].Color
}
