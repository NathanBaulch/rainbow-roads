package paint

import (
	"errors"
	"hash/fnv"
	"log"
	"math"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/NathanBaulch/rainbow-roads/geo"
	"github.com/serjvanilla/go-overpass"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/exp/slices"
)

type way struct {
	Geometry []geo.Point
	Highway  string
	Access   string
	Surface  string
}

const ttl = 168 * time.Hour

func osmLookup(query string) ([]*way, error) {
	h := fnv.New64()
	_, _ = h.Write([]byte(query))
	name := path.Join(os.TempDir(), "rainbow-roads")
	if err := os.MkdirAll(name, 0o777); err != nil {
		return nil, err
	}
	name = path.Join(name, big.NewInt(0).SetBytes(h.Sum(nil)).Text(62))

	if f, err := os.Stat(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	} else if err == nil && time.Since(f.ModTime()) < ttl {
		if data, err := os.ReadFile(name); err != nil {
			log.Println("WARN:", err)
		} else if ways, err := unpackWays(data); err != nil {
			log.Println("WARN:", err)
		} else {
			return ways, nil
		}
	}

	if res, err := overpass.Query(query); err != nil {
		return nil, err
	} else if data, err := packWays(res.Ways); err != nil {
		return nil, err
	} else if err := os.WriteFile(name, data, 0o777); err != nil {
		return nil, err
	} else {
		return unpackWays(data)
	}
}

func packWays(ways map[int64]*overpass.Way) ([]byte, error) {
	d := doc{Ways: make([]elem, len(ways))}

	i := 0
	for _, w := range ways {
		d.Ways[i].Geometry = make([][2]float32, len(w.Geometry))
		for j, g := range w.Geometry {
			pt := geo.NewPointFromDegrees(g.Lat, g.Lon)
			d.Ways[i].Geometry[j][0] = float32(pt.Lat)
			d.Ways[i].Geometry[j][1] = float32(pt.Lon)
		}

		packTag := func(tag string, known *[]string) uint8 {
			if val, ok := w.Tags[tag]; ok {
				j := slices.Index(*known, val)
				if j < 0 {
					j = len(*known)
					*known = append(*known, val)
				}
				return uint8(j)
			}
			return math.MaxUint8
		}
		d.Ways[i].Highway = packTag("highway", &d.Highways)
		d.Ways[i].Access = packTag("access", &d.Accesses)
		d.Ways[i].Surface = packTag("surface", &d.Surfaces)

		i++
	}

	return msgpack.Marshal(d)
}

func unpackWays(data []byte) ([]*way, error) {
	d := &doc{}
	if err := msgpack.Unmarshal(data, d); err != nil {
		return nil, err
	}

	ways := make([]*way, len(d.Ways))
	for i, w := range d.Ways {
		ways[i] = &way{Geometry: make([]geo.Point, len(w.Geometry))}
		for j, p := range w.Geometry {
			ways[i].Geometry[j].Lat = float64(p[0])
			ways[i].Geometry[j].Lon = float64(p[1])
		}

		if w.Highway < math.MaxUint8 {
			if w.Highway >= uint8(len(d.Highways)) {
				return nil, errors.New("invalid cache data")
			}
			ways[i].Highway = d.Highways[w.Highway]
		}
		if w.Access < math.MaxUint8 {
			if w.Access >= uint8(len(d.Accesses)) {
				return nil, errors.New("invalid cache data")
			}
			ways[i].Access = d.Accesses[w.Access]
		}
		if w.Surface < math.MaxUint8 {
			if w.Surface >= uint8(len(d.Surfaces)) {
				return nil, errors.New("invalid cache data")
			}
			ways[i].Surface = d.Surfaces[w.Surface]
		}
	}

	return ways, nil
}

type doc struct {
	Ways     []elem   `msgpack:"w"`
	Highways []string `msgpack:"h"`
	Accesses []string `msgpack:"a"`
	Surfaces []string `msgpack:"s"`
}

type elem struct {
	Geometry [][2]float32 `msgpack:"g"`
	Highway  uint8        `msgpack:"h"`
	Access   uint8        `msgpack:"a"`
	Surface  uint8        `msgpack:"s"`
}
