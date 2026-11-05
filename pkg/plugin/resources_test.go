package plugin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
)

func TestActionRoutesDoNotRestrictViewerRole(t *testing.T) {
	router := mux.NewRouter()
	(&Datasource{multiplexer: &source.Multiplexer{Endpoints: map[string]*source.YamcsEndpoint{}}}).registerRoutes(router)
	actionPaths := []string{
		"/endpoint/e1/command/issue",
		"/endpoint/e1/alarm/acknowledge",
		"/endpoint/e1/alarm/clear",
		"/endpoint/e1/alarm/shelve",
		"/endpoint/e1/alarm/unshelve",
		"/endpoint/e1/links/link/enable",
		"/endpoint/e1/links/link/disable",
		"/endpoint/e1/links/link/reset",
		"/endpoint/e1/links/link/action/action",
	}
	for _, path := range actionPaths {
		for _, role := range []string{"", "Viewer"} {
			t.Run(path+"/"+role, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, path, nil)
				if role != "" {
					req = req.WithContext(backend.WithUser(context.Background(), &backend.User{Role: role}))
				}
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code == http.StatusForbidden {
					t.Fatalf("%s role was unexpectedly denied", role)
				}
			})
		}
	}
}

func TestParameterPathValueFormatsAggregateAndArrayMembers(t *testing.T) {
	got := parameterPathValue("/drone/Motors", []string{"[0]", "rpm"})
	want := "/drone/Motors[0].rpm"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
