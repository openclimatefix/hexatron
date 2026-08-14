package tests

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openclimatefix/hexatron/backend/internal/clients"
	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	"github.com/openclimatefix/hexatron/backend/internal/routes"
	"github.com/openclimatefix/hexatron/backend/internal/services"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
	"github.com/openclimatefix/hexatron/backend/internal/structures/responses"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestHTTPHeartbeatServing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := clients.NewHeartbeatClient().Probe(context.Background(), models.HealthCheck{
		Type:   models.HealthCheckHTTP,
		Target: server.URL,
	})

	if !result.Serving {
		t.Fatalf("expected serving, got detail %q err %v", result.Detail, result.Err)
	}
}

// A responding service returning the wrong status is down, not unknown — it
// answered, and the answer was bad.
func TestHTTPHeartbeatWrongStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	result := clients.NewHeartbeatClient().Probe(context.Background(), models.HealthCheck{
		Type:   models.HealthCheckHTTP,
		Target: server.URL,
	})

	if result.Serving {
		t.Fatal("expected a 503 to fail the probe")
	}
	if result.Detail != "HTTP 503 (want 200)" {
		t.Errorf("expected the detail to name both statuses, got %q", result.Detail)
	}
}

func TestHTTPHeartbeatUnreachable(t *testing.T) {
	// Bound to a port nothing listens on: a refused connection, not a timeout.
	result := clients.NewHeartbeatClient().Probe(context.Background(), models.HealthCheck{
		Type:           models.HealthCheckHTTP,
		Target:         "http://127.0.0.1:1/health",
		TimeoutSeconds: 2,
	})

	if result.Serving {
		t.Fatal("expected an unreachable target to fail the probe")
	}
}

// The Data Platform path: a real gRPC server speaking grpc.health.v1.
func TestGRPCHeartbeatServing(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	result := clients.NewHeartbeatClient().Probe(context.Background(), models.HealthCheck{
		Type:   models.HealthCheckGRPC,
		Target: addr,
	})

	if !result.Serving {
		t.Fatalf("expected serving, got detail %q err %v", result.Detail, result.Err)
	}
}

// A server that answers but reports NOT_SERVING is down — the distinction the
// gRPC health protocol exists to make.
func TestGRPCHeartbeatNotServing(t *testing.T) {
	addr, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	result := clients.NewHeartbeatClient().Probe(context.Background(), models.HealthCheck{
		Type:   models.HealthCheckGRPC,
		Target: addr,
	})

	if result.Serving {
		t.Fatal("expected NOT_SERVING to fail the probe")
	}
}

func TestGRPCHeartbeatUnreachable(t *testing.T) {
	result := clients.NewHeartbeatClient().Probe(context.Background(), models.HealthCheck{
		Type:           models.HealthCheckGRPC,
		Target:         "127.0.0.1:1",
		TimeoutSeconds: 2,
	})

	if result.Serving {
		t.Fatal("expected an unreachable target to fail the probe")
	}
}

// The heartbeat decides for services Airflow cannot speak for, and can always
// veto to down — but it must not paper over failing DAGs.
func TestApplyHeartbeat(t *testing.T) {
	down := &responses.Heartbeat{Status: constants.StatusFailed}
	up := &responses.Heartbeat{Status: constants.StatusHealthy}
	unknown := &responses.Heartbeat{Status: constants.StatusUnknown}

	cases := []struct {
		name      string
		dagStatus string
		heartbeat *responses.Heartbeat
		want      string
	}{
		{"no check leaves dag status alone", constants.StatusHealthy, nil, constants.StatusHealthy},
		{"no dags, probe up", constants.StatusUnknown, up, constants.StatusHealthy},
		{"no dags, probe down", constants.StatusUnknown, down, constants.StatusFailed},
		{"no dags, probe inconclusive", constants.StatusUnknown, unknown, constants.StatusUnknown},
		{"probe down beats healthy dags", constants.StatusHealthy, down, constants.StatusFailed},
		{"probe up does not mask degraded dags", constants.StatusDegraded, up, constants.StatusDegraded},
		{"probe up does not mask failing dags", constants.StatusFailed, up, constants.StatusFailed},
		{"paused survives a healthy probe", constants.StatusPaused, up, constants.StatusPaused},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := services.ApplyHeartbeat(tc.dagStatus, tc.heartbeat)
			if got != tc.want {
				t.Errorf("ApplyHeartbeat(%q, %v) = %q, want %q", tc.dagStatus, tc.heartbeat, got, tc.want)
			}
		})
	}
}

// Results are cached so that many dashboard viewers cannot turn a heartbeat
// into load on the service it is meant to be watching.
func TestHeartbeatCaching(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	svc := models.Service{
		ID: "cached",
		HealthCheck: &models.HealthCheck{
			Type:   models.HealthCheckHTTP,
			Target: server.URL,
		},
	}

	heartbeats := services.NewHeartbeatService()
	for range 5 {
		if hb := heartbeats.Check(context.Background(), svc); hb == nil {
			t.Fatal("expected a heartbeat for a service with a health_check")
		}
	}

	if hits != 1 {
		t.Errorf("expected 5 checks inside the TTL to make 1 request, got %d", hits)
	}
}

func TestNoHealthCheckMeansNoHeartbeat(t *testing.T) {
	heartbeats := services.NewHeartbeatService()
	if hb := heartbeats.Check(context.Background(), models.Service{ID: "bare"}); hb != nil {
		t.Errorf("expected nil for a service with no health_check, got %+v", hb)
	}
}

func TestValidateHealthCheck(t *testing.T) {
	cases := []struct {
		name    string
		check   models.HealthCheck
		wantErr bool
	}{
		{"http with url", models.HealthCheck{Type: "http", Target: "http://x/health"}, false},
		{"type defaults to http", models.HealthCheck{Target: "https://x/health"}, false},
		{"grpc host port", models.HealthCheck{Type: "grpc", Target: "localhost:50051"}, false},
		{"no target", models.HealthCheck{Type: "http"}, true},
		{"http without scheme", models.HealthCheck{Type: "http", Target: "example.com"}, true},
		{"grpc as url", models.HealthCheck{Type: "grpc", Target: "grpc://localhost:50051"}, true},
		{"grpc without port", models.HealthCheck{Type: "grpc", Target: "localhost"}, true},
		{"unknown type", models.HealthCheck{Type: "smoke-signal", Target: "x:1"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := models.ValidateHealthCheck(tc.check)
			if tc.wantErr && err == nil {
				t.Error("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

// startHealthServer runs a gRPC server implementing the standard health service
// on an ephemeral port, and returns its dial address.
func startHealthServer(t *testing.T, status grpc_health_v1.HealthCheckResponse_ServingStatus) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}

	server := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", status)
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	go func() { _ = server.Serve(listener) }()

	// Stop rather than GracefulStop: the probe leaves no in-flight work worth
	// draining, and a hung client should not hold the test open.
	return listener.Addr().String(), func() {
		server.Stop()
		_ = listener.Close()
	}
}

// A target whose DNS name does not exist is a configuration problem, not an
// outage — the placeholders in services.yaml must not show the fleet as down.
func TestUnresolvableTargetIsInconclusive(t *testing.T) {
	result := clients.NewHeartbeatClient().Probe(context.Background(), models.HealthCheck{
		Type:           models.HealthCheckHTTP,
		Target:         "https://api.example.invalid/health",
		TimeoutSeconds: 5,
	})

	if result.Serving {
		t.Fatal("expected an unresolvable target to fail the probe")
	}
	if !result.Inconclusive {
		t.Errorf("expected NXDOMAIN to be inconclusive, got detail %q", result.Detail)
	}
}

// The same target, end to end: unknown rather than down.
func TestUnresolvableHeartbeatReportsUnknown(t *testing.T) {
	svc := models.Service{
		ID: "placeholder",
		HealthCheck: &models.HealthCheck{
			Type:           models.HealthCheckHTTP,
			Target:         "https://ui.example.invalid/",
			TimeoutSeconds: 5,
		},
	}

	hb := services.NewHeartbeatService().Check(context.Background(), svc)
	if hb == nil {
		t.Fatal("expected a heartbeat")
	}
	if hb.Status != constants.StatusUnknown {
		t.Errorf("expected %q for an unresolvable target, got %q", constants.StatusUnknown, hb.Status)
	}
}

// End-to-end through the real router: a health_check in services.yaml must
// reach the JSON response and decide the status of a service with no DAGs.
func TestHeartbeatReachesServicesEndpoint(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer downstream.Close()

	airflow := startFakeAirflow(t, map[string]fakeAirflowDAG{
		"forecast-ok": {State: constants.AirflowStateSuccess},
	})

	yaml := "services:\n" +
		"  - id: solar-forecast\n" +
		"    name: Solar Forecast\n" +
		"    category: Forecast\n" +
		"    depends_on: []\n" +
		"    dag_patterns:\n" +
		"      - forecast-*\n" +
		"  - id: ui\n" +
		"    name: UI\n" +
		"    category: Application\n" +
		"    depends_on: []\n" +
		"    dag_patterns: []\n" +
		"    health_check:\n" +
		"      type: http\n" +
		"      target: " + upstream.URL + "\n" +
		"  - id: api\n" +
		"    name: API\n" +
		"    category: Application\n" +
		"    depends_on: []\n" +
		"    dag_patterns: []\n" +
		"    health_check:\n" +
		"      type: http\n" +
		"      target: " + downstream.URL + "\n"

	router := routes.NewRouter(&configstructs.Config{
		Addr:               ":8080",
		ServicesConfigPath: writeServicesYAML(t, yaml),
		AirflowBaseURL:     airflow.URL,
		AirflowCookie:      "test-cookie",
	})

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var got responses.ServiceListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	byID := make(map[string]responses.ServiceResponse, len(got))
	for _, svc := range got {
		byID[svc.ID] = svc
	}

	// No DAGs, probe healthy: the heartbeat is the only evidence, so it decides.
	ui, ok := byID["ui"]
	if !ok {
		t.Fatal("ui missing from response")
	}
	if ui.Heartbeat == nil {
		t.Fatal("expected a heartbeat on ui")
	}
	if ui.Heartbeat.Status != constants.StatusHealthy {
		t.Errorf("expected ui heartbeat healthy, got %q (%v)", ui.Heartbeat.Status, ui.Heartbeat.Detail)
	}
	if ui.Status != constants.StatusHealthy {
		t.Errorf("expected ui status healthy, got %q", ui.Status)
	}
	if ui.Heartbeat.LatencyMs == nil {
		t.Error("expected a latency reading on a completed probe")
	}

	// No DAGs, probe failing: down rather than the unknown it used to report.
	api := byID["api"]
	if api.Heartbeat == nil {
		t.Fatal("expected a heartbeat on api")
	}
	if api.Status != constants.StatusFailed {
		t.Errorf("expected api status %q, got %q", constants.StatusFailed, api.Status)
	}

	// Services with no health_check must be untouched by any of this.
	solar := byID["solar-forecast"]
	if solar.Heartbeat != nil {
		t.Errorf("expected no heartbeat on solar-forecast, got %+v", solar.Heartbeat)
	}
	if solar.Status != constants.StatusHealthy {
		t.Errorf("expected solar-forecast to stay healthy, got %q", solar.Status)
	}
}

// The reason expect_body_contains exists: a page that catches its own backend
// failure still returns 200, so the status code alone cannot tell the two
// apart.
func TestHTTPHeartbeatBodyMarker(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	check := models.HealthCheck{
		Type:               models.HealthCheckHTTP,
		Target:             server.URL,
		ExpectBodyContains: "Hexatron",
	}
	client := clients.NewHeartbeatClient()

	body = "<html><body><h1>Hexatron</h1><main>ok</main></body></html>"
	if result := client.Probe(context.Background(), check); !result.Serving {
		t.Errorf("expected a rendered page to pass, got %q", result.Detail)
	}

	// The error boundary: 200, but not the page anyone wanted.
	body = "<html><body>Could not reach the backend</body></html>"
	result := client.Probe(context.Background(), check)
	if result.Serving {
		t.Error("expected a 200 error page to fail the probe")
	}
	if result.Detail == "" {
		t.Error("expected the detail to explain the body mismatch")
	}
}

// Redirects are answers, not detours. An auth-gated app 307s to its login page;
// following that silently would have the probe report on a page nobody
// configured, so the 3xx must survive to be asserted on.
func TestHTTPHeartbeatDoesNotFollowRedirects(t *testing.T) {
	var loginHits int
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		loginHits++
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := clients.NewHeartbeatClient()

	// Expecting 200 must fail: the app itself said 307.
	result := client.Probe(context.Background(), models.HealthCheck{
		Type:   models.HealthCheckHTTP,
		Target: server.URL + "/",
	})
	if result.Serving {
		t.Error("expected a 307 to fail a probe that wants 200")
	}
	if loginHits != 0 {
		t.Errorf("expected the redirect not to be followed, but /login was hit %d times", loginHits)
	}

	// Expecting the redirect itself passes — this is the app.quartz.solar case.
	result = client.Probe(context.Background(), models.HealthCheck{
		Type:         models.HealthCheckHTTP,
		Target:       server.URL + "/",
		ExpectStatus: http.StatusTemporaryRedirect,
	})
	if !result.Serving {
		t.Errorf("expected expect_status 307 to pass, got %q", result.Detail)
	}
}

// gRPC resolves names in its own resolver and reports failure as an opaque
// status, so the typed DNS check does not match it. A misconfigured grpc target
// must report "not checked" like its http equivalent, not a false outage.
//
// This pins a grpc-go error string: if the wording changes, this test fails
// loudly rather than the behaviour regressing silently.
func TestGRPCUnresolvableTargetIsInconclusive(t *testing.T) {
	result := clients.NewHeartbeatClient().Probe(context.Background(), models.HealthCheck{
		Type:           models.HealthCheckGRPC,
		Target:         "no-such-host.example.invalid:50051",
		TimeoutSeconds: 5,
	})

	if result.Serving {
		t.Fatal("expected an unresolvable grpc target to fail the probe")
	}
	if !result.Inconclusive {
		t.Errorf("expected an unresolvable grpc target to be inconclusive, got detail %q", result.Detail)
	}
}
