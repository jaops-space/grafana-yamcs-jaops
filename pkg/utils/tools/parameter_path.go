package tools

import (
	"strings"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/mdb"
)

// ParameterTypeAtPath walks parameterType through path (a sequence of
// "[index]" array-member and "member" aggregate-member tokens, e.g.
// []string{"[0]", "rpm"} for ".../Motors[0].rpm") and returns the
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
		if strings.HasPrefix(part, "[") {
			current = current.GetArrayInfo().GetType()
			continue
		}
		var memberType *mdb.ParameterTypeInfo
		for _, member := range current.GetMember() {
			if member.GetName() == part {
				memberType = member.GetType()
				break
			}
		}
		current = memberType
	}
	return current
}
