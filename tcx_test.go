package main

import (
	"bytes"
	"testing"
)

func TestTCXNoPosition(t *testing.T) {
	sports = sports[:0]
	if acts, err := parseTCX(bytes.NewBufferString(`
		<TrainingCenterDatabase>
		  <Activities>
		    <Activity>
		      <Lap>
		        <Track>
		          <Trackpoint>
		            <Position/>
		          </Trackpoint>
		        </Track>
		      </Lap>
		    </Activity>
		  </Activities>
		</TrainingCenterDatabase>`)); err != nil {
		t.Fatal(err)
	} else if len(acts) > 0 {
		t.Fatal("expected no activities")
	}
}
