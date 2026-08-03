package client

import (
	"testing"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/processing"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestHandleParameterMessageUsesSnapshotMappingForValues(t *testing.T) {
	paramName := "/myproject/Battery1_Voltage"
	numericID := uint32(42)
	delivered := make([]string, 0, 1)

	subscription := &ParameterSubscription{
		subscriptionID:      7,
		parameterIDToName:   map[int]string{},
		ActiveSubscriptions: types.NewSet[string](),
		valueChangeListener: func(parameter string, newValue *pvalue.ParameterValue) error {
			delivered = append(delivered, parameter)
			if newValue.GetNumericId() != numericID {
				t.Fatalf("expected numeric ID %d, got %d", numericID, newValue.GetNumericId())
			}
			return nil
		},
	}
	c := &YamcsClient{
		ParameterSubscriptions: map[int32]*ParameterSubscription{
			subscription.subscriptionID: subscription,
		},
	}

	payload, err := anypb.New(&processing.SubscribeParametersData{
		Mapping: map[uint32]*protobuf.NamedObjectId{
			numericID: {Name: &paramName},
		},
		Values: []*pvalue.ParameterValue{
			{NumericId: &numericID, AcquisitionStatus: pvalue.AcquisitionStatus_ACQUIRED.Enum()},
		},
	})
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	c.HandleParameterMessage(&api.ServerMessage{
		Type: "parameters",
		Call: int32(subscription.subscriptionID),
		Data: payload,
	})

	if got := subscription.parameterIDToName[int(numericID)]; got != paramName {
		t.Fatalf("expected mapped parameter %q, got %q", paramName, got)
	}
	if len(delivered) != 1 || delivered[0] != paramName {
		t.Fatalf("expected one delivered parameter %q, got %#v", paramName, delivered)
	}
}

func TestHandleParameterMessageIgnoresUnknownSubscriptionCall(t *testing.T) {
	numericID := uint32(1)
	paramName := "/myproject/Battery1_Voltage"
	payload, err := anypb.New(&processing.SubscribeParametersData{
		Mapping: map[uint32]*protobuf.NamedObjectId{
			numericID: {Name: &paramName},
		},
	})
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	c := &YamcsClient{ParameterSubscriptions: map[int32]*ParameterSubscription{}}
	c.HandleParameterMessage(&api.ServerMessage{Type: "parameters", Call: 99, Data: payload})

	if len(c.ParameterSubscriptions) != 0 {
		t.Fatalf("expected unknown call to leave registry unchanged")
	}
}
