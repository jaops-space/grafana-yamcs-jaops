package tools

import (
	"strings"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/mdb"
)

// ParameterTypeAtPath walks parameterType through path (a sequence of
// "[index]", "member", or combined "member[index]" tokens, e.g.
// []string{"battery", "temperatures[0]"}) and returns the
// ParameterTypeInfo actually found at that path, or nil if any step along
// the way doesn't resolve. Shared by the parameter-options search endpoint
// (path segments from Yamcs's own member search) and
// getOrCreateParameterDemand (path segments from mdb.ParameterInfo.GetPath(),
// which Yamcs's GetParameter endpoint already parses out of a combined
// "name[index].member" request name) so both agree on the exact same
// member-type resolution.
func ParameterTypeAtPath(parameterType *mdb.ParameterTypeInfo, path []string) *mdb.ParameterTypeInfo {
	current := parameterType
	for _, part := range path {
		if current == nil {
			return nil
		}
		name, _, indexed := strings.Cut(part, "[")
		if name != "" || !indexed {
			var memberType *mdb.ParameterTypeInfo
			for _, member := range current.GetMember() {
				if member.GetName() == name {
					memberType = member.GetType()
					break
				}
			}
			current = memberType
		}
		if indexed {
			// One ArrayInfo describes all dimensions, including "[0][1]".
			current = current.GetArrayInfo().GetType()
		}
	}
	return current
}
