package tools

import (
	"testing"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/mdb"
)

func strptr(value string) *string {
	return &value
}

func TestParameterTypeAtPathResolvesArrayAggregateLeafType(t *testing.T) {
	rpmType := &mdb.ParameterTypeInfo{
		EngType: strptr("integer"),
		UnitSet: []*mdb.UnitInfo{
			{Unit: strptr("rpm")},
		},
	}
	motorType := &mdb.ParameterTypeInfo{
		EngType: strptr("aggregate"),
		Member: []*mdb.MemberInfo{
			{Name: strptr("rpm"), Type: rpmType},
		},
	}
	motorsType := &mdb.ParameterTypeInfo{
		EngType: strptr("aggregate[]"),
		ArrayInfo: &mdb.ArrayInfo{
			Type: motorType,
		},
	}

	got := ParameterTypeAtPath(motorsType, []string{"[0]", "rpm"})
	if got == nil {
		t.Fatal("expected leaf type, got nil")
	}
	if got.GetEngType() != "integer" {
		t.Fatalf("expected integer leaf type, got %q", got.GetEngType())
	}
	if got.GetUnitSet()[0].GetUnit() != "rpm" {
		t.Fatalf("expected rpm leaf unit, got %q", got.GetUnitSet()[0].GetUnit())
	}
}

func TestParameterTypeAtPathResolvesPlainArrayLeafType(t *testing.T) {
	// Regression case: a plain numeric array (not array-of-aggregates), the
	// shape that exposed the original bug - unit/thresholds for
	// "/drone/BatteryCellVoltages[0]" live on the array's element type, not
	// on the array parameter's own top-level type.
	elementType := &mdb.ParameterTypeInfo{
		EngType: strptr("float"),
		UnitSet: []*mdb.UnitInfo{
			{Unit: strptr("V")},
		},
	}
	arrayType := &mdb.ParameterTypeInfo{
		EngType: strptr("float[]"),
		ArrayInfo: &mdb.ArrayInfo{
			Type: elementType,
		},
	}

	got := ParameterTypeAtPath(arrayType, []string{"[0]"})
	if got == nil {
		t.Fatal("expected the array's element type, got nil")
	}
	if got.GetUnitSet()[0].GetUnit() != "V" {
		t.Fatalf("expected element unit V, got %q", got.GetUnitSet()[0].GetUnit())
	}
}

func TestParameterTypeAtPathResolvesFromYamcsOwnParsedPath(t *testing.T) {
	// End-to-end of the actual bug: mdb.ParameterInfo.GetPath() is what
	// Yamcs's own GetParameter endpoint returns after parsing a combined
	// "name[index].member" request name - confirmed live against a real
	// Yamcs instance to return e.g. []string{"[0]"} for
	// "/drone/BatteryCellVoltages[0]". Given that path and the array
	// parameter's own type (with unit/thresholds nested one level down in
	// ArrayInfo.Type, not on the top level), this must resolve to the leaf
	// type, not the array's own (unit-less) top-level type.
	elementType := &mdb.ParameterTypeInfo{
		EngType: strptr("float"),
		UnitSet: []*mdb.UnitInfo{{Unit: strptr("V")}},
	}
	arrayType := &mdb.ParameterTypeInfo{
		EngType:   strptr("float[]"),
		ArrayInfo: &mdb.ArrayInfo{Type: elementType},
	}

	resolved := ParameterTypeAtPath(arrayType, []string{"[0]"})

	if resolved == nil {
		t.Fatal("expected the resolved leaf type, got nil")
	}
	if len(resolved.GetUnitSet()) == 0 || resolved.GetUnitSet()[0].GetUnit() != "V" {
		t.Fatalf("expected leaf unit V, got %v", resolved.GetUnitSet())
	}
}
