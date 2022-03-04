package main

import (
	"bytes"
	"testing"
)

func TestTCXNoPosition(t *testing.T) {
	sports = sports[:0]
	activities = activities[:0]
	if err := parseTCX(bytes.NewBufferString(`
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
	}
	if len(activities) > 0 {
		t.Fatal("expected no activities")
	}
}
