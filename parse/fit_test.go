package parse

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tormoder/fit"
)

func TestFITShortDistance(t *testing.T) {
	is := require.New(t)

	w := &bytes.Buffer{}
	f, err := fit.NewFile(fit.FileTypeActivity, fit.NewHeader(fit.V20, false))
	is.NoError(err)
	a, _ := f.Activity()
	a.Sessions = append(a.Sessions, &fit.SessionMsg{TotalDistance: 1})
	a.Records = append(a.Records, &fit.RecordMsg{Timestamp: time.Now()}, &fit.RecordMsg{Timestamp: time.Now().Add(time.Second)})
	is.NoError(fit.Encode(w, f, binary.BigEndian))
	acts, err := parseFIT(w, &Selector{})
	is.NoError(err)
	is.Len(acts, 1)
}
