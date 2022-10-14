package parse

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/tormoder/fit"
)

func TestFITShortDistance(t *testing.T) {
	w := &bytes.Buffer{}
	if f, err := fit.NewFile(fit.FileTypeActivity, fit.NewHeader(fit.V20, false)); err != nil {
		t.Fatal(err)
	} else {
		a, _ := f.Activity()
		a.Sessions = append(a.Sessions, &fit.SessionMsg{TotalDistance: 1})
		a.Records = append(a.Records, &fit.RecordMsg{Timestamp: time.Now()}, &fit.RecordMsg{Timestamp: time.Now().Add(time.Second)})
		if err := fit.Encode(w, f, binary.BigEndian); err != nil {
			t.Fatal(err)
		}
	}
	if acts, err := parseFIT(w, &Selector{}); err != nil {
		t.Fatal(err)
	} else if len(acts) != 1 {
		t.Fatal("expected 1 activity")
	}
}
