package source

import (
	"fmt"
	"testing"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/commanding"
)

func TestCommandHistoryListenerBuffersBurstsWithoutReceiver(t *testing.T) {
	signal := make(chan *commanding.CommandHistoryEntry, StreamSignalBufferSize)
	endpoint := &YamcsEndpoint{
		CommandHistorySignals: map[string]CommandHistorySignal{
			"req/commands": signal,
		},
	}

	listener := endpoint.getCommandHistoryListener()
	for i := 0; i < 8; i++ {
		commandID := fmt.Sprintf("cmd-%d", i)
		if err := listener(&commanding.CommandHistoryEntry{Id: &commandID}); err != nil {
			t.Fatalf("listener returned error: %v", err)
		}
	}

	if got := len(signal); got != 8 {
		t.Fatalf("expected 8 buffered command history entries, got %d", got)
	}
}
