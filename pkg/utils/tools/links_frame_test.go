package tools

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/actions"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/links"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConvertLinksToFrameUsesColumnarFields(t *testing.T) {
	instance := "sim"
	name := "tm"
	linkType := "UdpTmFrameLink"
	disabled := true
	status := "UNAVAIL"
	dataInCount := int64(12)
	dataOutCount := int64(3)
	detailedStatus := "disabled by operator"
	parentName := "parent"
	actionID := "reset"
	actionLabel := "Reset"
	actionStyle := "PUSH_BUTTON"
	actionEnabled := true
	actionChecked := false
	extra, err := structpb.NewStruct(map[string]any{"vc": "tm"})
	if err != nil {
		t.Fatal(err)
	}

	frame, err := ConvertLinksToFrame([]*links.LinkInfo{{
		Instance:       &instance,
		Name:           &name,
		Type:           &linkType,
		Disabled:       &disabled,
		Status:         &status,
		DataInCount:    &dataInCount,
		DataOutCount:   &dataOutCount,
		DetailedStatus: &detailedStatus,
		ParentName:     &parentName,
		Actions: []*actions.ActionInfo{{
			Id:      &actionID,
			Label:   &actionLabel,
			Style:   &actionStyle,
			Enabled: &actionEnabled,
			Checked: &actionChecked,
		}},
		Extra: extra,
	}})
	if err != nil {
		t.Fatal(err)
	}

	if frame.Name != "links" {
		t.Fatalf("expected frame name links, got %q", frame.Name)
	}
	if frame.Fields[0].Name == "linksJson" {
		t.Fatal("links frame must not use a JSON blob field")
	}
	if frame.Fields[0].Len() != 1 {
		t.Fatalf("expected one row, got %d", frame.Fields[0].Len())
	}

	assertFieldValue(t, frame, "name", 0, name)
	assertFieldValue(t, frame, "disabled", 0, disabled)
	assertFieldValue(t, frame, "status", 0, status)
	assertFieldValue(t, frame, "dataInCount", 0, dataInCount)
	assertFieldValue(t, frame, "dataOutCount", 0, dataOutCount)
	assertFieldValue(t, frame, "detailedStatus", 0, detailedStatus)
	assertFieldValue(t, frame, "parentName", 0, parentName)
}

func assertFieldValue(t *testing.T, frame *data.Frame, name string, row int, want any) {
	t.Helper()
	field, _ := frame.FieldByName(name)
	if field == nil {
		t.Fatalf("expected field %q", name)
	}
	got := field.At(row)
	if got != want {
		t.Fatalf("expected %s[%d] = %#v, got %#v", name, row, want, got)
	}
}
