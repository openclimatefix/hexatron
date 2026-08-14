package clients

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/openclimatefix/hexatron/backend/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// maxProbeBodyBytes caps how much of a response a body check will read.
const maxProbeBodyBytes = 256 * 1024

// ProbeResult is one liveness observation of a service.
//
// Serving is deliberately a tri-state via Err: a probe that could not be
// carried out at all (bad config, cancelled context) must not read the same as
// one that ran and found the service down.
type ProbeResult struct {
	Serving bool
	Latency time.Duration
	// Detail is a short operator-facing reason, populated whether or not the
	// probe succeeded — "HTTP 503" is as useful as "connection refused".
	Detail string
	Err    error

	// Inconclusive marks a probe that says nothing about the service: the name
	// does not resolve, so there is nothing to be down. Reporting that as an
	// outage would raise an alarm about a typo in services.yaml.
	Inconclusive bool
}

// unresolvable reports a DNS name that does not exist. A refused connection
// means something is meant to be there and isn't — a real outage — but an
// unknown host means the target was never right.
func unresolvable(err error) bool {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		return true
	}
	// gRPC resolves names inside its own resolver and reports the failure as an
	// opaque status, so the typed check above never matches it. Without this a
	// misconfigured grpc target reports a false outage while the equivalent
	// http target correctly reports "not checked".
	return strings.Contains(err.Error(), grpcNoAddressesMsg)
}

// The resolver's wording when a target resolves to nothing at all.
const grpcNoAddressesMsg = "produced zero addresses"

// HeartbeatClient runs liveness probes against non-Airflow services.
//
// The HTTP client is shared so keep-alives survive between polls; a fresh
// client per probe would make every heartbeat pay a new TCP and TLS handshake
// and report latency that says more about connection setup than about the
// service.
type HeartbeatClient struct {
	http *http.Client
}

// NewHeartbeatClient creates a HeartbeatClient.
//
// No client-level timeout: each probe carries its own deadline on the context,
// so a slow service is bounded by its own configured timeout rather than by a
// single value shared across every service in the fleet.
func NewHeartbeatClient() *HeartbeatClient {
	return &HeartbeatClient{
		http: &http.Client{
			// Redirects are answers, not detours. An auth-gated app redirects to
			// its login page: followed silently, the probe reports on a page
			// nobody configured, and a body check against the real app fails
			// while the app is perfectly healthy. Surfacing the 3xx lets the
			// check assert on it instead.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Probe runs check and reports what it found. It never returns an error to the
// caller — a failed probe is a result, not a fault.
func (c *HeartbeatClient) Probe(ctx context.Context, check models.HealthCheck) ProbeResult {
	n := check.Normalised()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(n.TimeoutSeconds)*time.Second)
	defer cancel()

	start := time.Now()
	switch n.Type {
	case models.HealthCheckGRPC:
		return c.probeGRPC(ctx, n, start)
	default:
		return c.probeHTTP(ctx, n, start)
	}
}

func (c *HeartbeatClient) probeHTTP(ctx context.Context, n models.HealthCheck, start time.Time) ProbeResult {
	// GET rather than HEAD: health endpoints routinely 404 or 405 a HEAD even
	// when the service behind them is perfectly well.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.Target, nil)
	if err != nil {
		return ProbeResult{Detail: "malformed target", Err: err}
	}

	resp, err := c.http.Do(req)
	latency := time.Since(start)
	if err != nil {
		return ProbeResult{
			Latency:      latency,
			Detail:       unreachable(err),
			Err:          err,
			Inconclusive: unresolvable(err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != n.ExpectStatus {
		return ProbeResult{
			Latency: latency,
			Detail:  fmt.Sprintf("HTTP %d (want %d)", resp.StatusCode, n.ExpectStatus),
		}
	}

	if n.ExpectBodyContains != "" {
		// Capped rather than read whole: a health probe must not pull a large
		// page into memory, and the marker is near the top of a rendered
		// document in practice.
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxProbeBodyBytes))
		if err != nil {
			return ProbeResult{Latency: latency, Detail: "could not read body", Err: err}
		}
		if !bytes.Contains(body, []byte(n.ExpectBodyContains)) {
			return ProbeResult{
				Latency: latency,
				Detail:  fmt.Sprintf("HTTP %d but body missing %q", resp.StatusCode, n.ExpectBodyContains),
			}
		}
	}

	return ProbeResult{Serving: true, Latency: latency, Detail: fmt.Sprintf("HTTP %d", resp.StatusCode)}
}

// probeGRPC speaks the standard gRPC health checking protocol
// (grpc.health.v1.Health/Check).
//
// Credentials are insecure by design: the only gRPC target here is a localhost
// sidecar, where TLS would be ceremony. A remote target would need real
// credentials before this could be pointed at it.
func (c *HeartbeatClient) probeGRPC(ctx context.Context, n models.HealthCheck, start time.Time) ProbeResult {
	// NewClient does not dial; the connection is established lazily by the RPC
	// below, so the context deadline covers connect and call together.
	conn, err := grpc.NewClient(n.Target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return ProbeResult{Detail: "bad dial target", Err: err}
	}
	defer conn.Close()

	resp, err := grpc_health_v1.NewHealthClient(conn).Check(
		ctx,
		&grpc_health_v1.HealthCheckRequest{Service: n.GRPCService},
		// Without this the RPC fails fast on a cold connection instead of
		// waiting for the dial the deadline already allows for.
		grpc.WaitForReady(true),
	)
	latency := time.Since(start)
	if err != nil {
		return ProbeResult{
			Latency:      latency,
			Detail:       unreachable(err),
			Err:          err,
			Inconclusive: unresolvable(err),
		}
	}

	serving := resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING
	return ProbeResult{
		Serving: serving,
		Latency: latency,
		Detail:  fmt.Sprintf("gRPC %s", resp.GetStatus()),
	}
}

// unreachable keeps probe failures to one readable line — the full transport
// error is several, and the dashboard has room for a phrase.
func unreachable(err error) string {
	msg := err.Error()
	if len(msg) > 120 {
		return msg[:117] + "..."
	}
	return msg
}
