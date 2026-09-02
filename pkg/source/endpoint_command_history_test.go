package source

import (
	"fmt"
	"testing"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
)

func TestCommandHistoryListenerBuffersBurstsWithoutReceiver(t *testing.T) {
	ring := types.NewRing[*commanding.CommandHistoryEntry](BroadcastRingCapacity)
	endpoint := &YamcsEndpoint{
		CommandHistoryRing: ring,
		CommandHistorySignals: map[string]*BroadcastStreamDemand[*commanding.CommandHistoryEntry]{
			"req/commands": newBroadcastStreamDemand(ring),
		},
	}

	listener := endpoint.getCommandHistoryListener()
	for i := 0; i < 8; i++ {
		commandID := fmt.Sprintf("cmd-%d", i)
		if err := listener(&commanding.CommandHistoryEntry{Id: &commandID}); err != nil {
			t.Fatalf("listener returned error: %v", err)
		}
	}

	if got := len(endpoint.DrainCommandHistoryStream("req/commands")); got != 8 {
		t.Fatalf("expected 8 buffered command history entries, got %d", got)
	}
}
