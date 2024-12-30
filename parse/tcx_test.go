package parse

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTCXNoPosition(t *testing.T) {
	is := require.New(t)

	acts, err := parseTCX(bytes.NewBufferString(`
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
		</TrainingCenterDatabase>`), &Selector{})
	is.NoError(err)
	is.Empty(acts)
}
