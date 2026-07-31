package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
)

// fakeAirflowDAG describes one DAG the fake Airflow should serve.
type fakeAirflowDAG struct {
	// State is the latest DAG run state; empty means the DAG has never run.
	State    string
	IsPaused bool
}

// startFakeAirflow serves canned Airflow REST responses for the given DAGs.
func startFakeAirflow(t *testing.T, dags map[string]fakeAirflowDAG) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/dags", func(w http.ResponseWriter, r *http.Request) {
		if !hasSessionCookie(r) {
			writeAirflowUnauthorized(w)
			return
		}

		list := map[string]any{"dags": []any{}, "total_entries": len(dags)}
		entries := make([]any, 0, len(dags))
		for dagID, dag := range dags {
			entries = append(entries, map[string]any{
				"dag_id":            dagID,
				"dag_display_name":  dagID,
				"is_active":         true,
				"is_paused":         dag.IsPaused,
				"has_import_errors": false,
				"schedule_interval": map[string]any{"__type": "CronExpression", "value": "0 * * * *"},
			})
		}
		list["dags"] = entries

		writeJSON(w, list)
	})

	// GET /api/v1/dags/{dagID}/dagRuns
	mux.HandleFunc("GET /api/v1/dags/{dagID}/dagRuns", func(w http.ResponseWriter, r *http.Request) {
		if !hasSessionCookie(r) {
			writeAirflowUnauthorized(w)
			return
		}

		dagID := r.PathValue("dagID")
		dag, known := dags[dagID]
		if !known {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": http.StatusNotFound, "title": "DAG not found",
			})
			return
		}

		if dag.State == "" {
			writeJSON(w, map[string]any{"dag_runs": []any{}, "total_entries": 0})
			return
		}

		writeJSON(w, map[string]any{
			"dag_runs": []any{map[string]any{
				"dag_id":     dagID,
				"dag_run_id": "scheduled__2026-07-29T00:00:00+00:00",
				"state":      dag.State,
				"run_type":   "scheduled",
				"start_date": "2026-07-29T06:00:02.446598+00:00",
				"end_date":   "2026-07-29T06:16:34.380490+00:00",
			}},
			"total_entries": 1,
		})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func hasSessionCookie(r *http.Request) bool {
	cookie, err := r.Cookie(constants.AirflowSessionCookieName)
	return err == nil && strings.TrimSpace(cookie.Value) != ""
}

func writeAirflowUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": http.StatusUnauthorized, "title": "Unauthorized",
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// writeServicesYAML writes a services.yaml into a temp dir and returns its path.
func writeServicesYAML(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "services.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing temp services.yaml: %v", err)
	}
	return path
}
