// Package clients is the only component aware of Airflow APIs.
//
// Everything above this package works in terms of the models it returns, never
// raw JSON. Authentication is a session cookie carried on every request, which
// is a development-only arrangement — production will use a service account.
package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
	airflowmodels "github.com/openclimatefix/hexatron/backend/internal/models/airflow"
	clientstructs "github.com/openclimatefix/hexatron/backend/internal/structures/clients"
)

// ErrDAGNotFound means Airflow does not know a DAG, which usually indicates
// services.yaml references one that has been renamed or removed.
var ErrDAGNotFound = errors.New("airflow: dag not found")

// AirflowClient communicates with the Airflow REST API.
type AirflowClient struct {
	config     clientstructs.AirflowClientConfig
	baseURL    *url.URL
	httpClient *http.Client
}

// NewAirflowClient returns a new AirflowClient using the given config. An
// unparseable base URL is tolerated here so construction stays infallible;
// requests then fail with a clear error.
func NewAirflowClient(cfg clientstructs.AirflowClientConfig) *AirflowClient {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		baseURL = nil
	}

	return &AirflowClient{
		config:     cfg,
		baseURL:    baseURL,
		httpClient: NewHTTPClient(),
	}
}

// APIError is an error response from the Airflow REST API. A stale or missing
// session cookie surfaces here as a 401.
type APIError struct {
	StatusCode int    `json:"status"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("airflow: %d %s", e.StatusCode, e.Title)
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	if e.StatusCode == http.StatusUnauthorized {
		msg += " (session cookie is missing or expired)"
	}
	return msg
}

// IsUnauthorized reports whether err is an Airflow 401, i.e. the session cookie
// needs refreshing. Callers use it to tell "Airflow says no" apart from "this
// DAG is genuinely failing".
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

// get performs a GET against the Airflow API and decodes the JSON body into out.
func (c *AirflowClient) get(ctx context.Context, path string, query url.Values, out any) error {
	if c.baseURL == nil {
		return fmt.Errorf("airflow: invalid base URL %q", c.config.BaseURL)
	}

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("airflow: build request: %w", err)
	}
	req.Header.Set("Accept", constants.ContentTypeJSON)
	req.AddCookie(&http.Cookie{Name: constants.AirflowSessionCookieName, Value: c.config.Cookie})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("airflow: GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return newAPIError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("airflow: decode %s response: %w", path, err)
	}
	return nil
}

// newAPIError converts a non-200 response into an *APIError, falling back to the
// raw body when Airflow does not return its usual JSON error envelope (a proxy
// or login redirect, say).
func newAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	apiErr := &APIError{StatusCode: resp.StatusCode}
	if err := json.Unmarshal(body, apiErr); err != nil || apiErr.Title == "" {
		return &APIError{
			StatusCode: resp.StatusCode,
			Title:      http.StatusText(resp.StatusCode),
			Detail:     string(body),
		}
	}
	// Trust the transport status over the one in the body.
	apiErr.StatusCode = resp.StatusCode
	return apiErr
}

// GetHealth calls GET /api/v1/health on the Airflow REST API. A nil error means
// Airflow is reachable and the session cookie is accepted.
func (c *AirflowClient) GetHealth(ctx context.Context) error {
	var out map[string]any
	return c.get(ctx, constants.AirflowHealthPath, nil, &out)
}

// ListDAGs returns every DAG registered in Airflow, following pagination.
func (c *AirflowClient) ListDAGs(ctx context.Context) ([]airflowmodels.DAG, error) {
	var all []airflowmodels.DAG

	for {
		query := url.Values{}
		query.Set("limit", strconv.Itoa(constants.AirflowPageSize))
		query.Set("offset", strconv.Itoa(len(all)))

		var page airflowmodels.DAGList
		if err := c.get(ctx, constants.AirflowDagsPath, query, &page); err != nil {
			return nil, fmt.Errorf("list dags: %w", err)
		}

		all = append(all, page.DAGs...)

		// A short page means Airflow has nothing more to give, even if
		// total_entries disagrees. Checking it guards against looping forever.
		if len(page.DAGs) == 0 || len(all) >= page.TotalEntries {
			return all, nil
		}
	}
}

// GetLatestDagRun fetches the most recent run for the given DAG ID. A DAG that
// exists but has never run yields (nil, nil); an unknown DAG yields
// ErrDAGNotFound.
func (c *AirflowClient) GetLatestDagRun(ctx context.Context, dagID string) (*airflowmodels.DagRun, error) {
	query := url.Values{}
	query.Set("order_by", "-execution_date")
	query.Set("limit", "1")

	var list airflowmodels.DagRunList
	path := constants.AirflowDagsPath + "/" + url.PathEscape(dagID) + "/dagRuns"

	if err := c.get(ctx, path, query, &list); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %s", ErrDAGNotFound, dagID)
		}
		return nil, fmt.Errorf("latest run for %s: %w", dagID, err)
	}

	if len(list.DagRuns) == 0 {
		return nil, nil
	}
	return &list.DagRuns[0], nil
}

// GetLatestDagRuns fetches the most recent run of each DAG concurrently. DAGs
// that have never run, and DAGs Airflow does not know, map to a nil run so the
// caller can render them as unknown rather than failing the whole dashboard.
//
// Airflow's batch endpoint (POST /dags/~/dagRuns/list) cannot do this job: its
// order_by sorts the combined result set, so a single busy DAG crowds every
// other DAG off the page. One small request per DAG is the reliable way to get
// latest-per-DAG.
func (c *AirflowClient) GetLatestDagRuns(ctx context.Context, dagIDs []string) (map[string]*airflowmodels.DagRun, error) {
	runs := make(map[string]*airflowmodels.DagRun, len(dagIDs))
	errs := make([]error, len(dagIDs))

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, constants.AirflowMaxConcurrentRequests)

	for i, dagID := range dagIDs {
		wg.Add(1)
		go func() {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			run, err := c.GetLatestDagRun(ctx, dagID)
			if err != nil && !errors.Is(err, ErrDAGNotFound) {
				errs[i] = err
				return
			}

			mu.Lock()
			defer mu.Unlock()
			runs[dagID] = run
		}()
	}
	wg.Wait()

	// Errors are returned alongside the runs that did succeed: an expired cookie
	// fails every DAG and is worth surfacing, but one flaky request should not
	// discard the rest.
	return runs, errors.Join(errs...)
}

// DAGURL is a deep link to a DAG's grid view in the Airflow UI. Airflow remains
// the source of truth for logs, task failures and retry history, so Hexatron
// hands users off to it rather than reimplementing it.
func (c *AirflowClient) DAGURL(dagID string) string {
	if c.baseURL == nil {
		return ""
	}
	ref := &url.URL{Path: "/dags/" + url.PathEscape(dagID) + "/grid"}
	return c.baseURL.ResolveReference(ref).String()
}

// GetRecentDagRuns fetches up to limit recent runs for a given DAG ID.
func (c *AirflowClient) GetRecentDagRuns(ctx context.Context, dagID string, limit int) ([]airflowmodels.DagRun, error) {
	if limit <= 0 {
		limit = 10
	}
	query := url.Values{}
	query.Set("order_by", "-execution_date")
	query.Set("limit", strconv.Itoa(limit))

	var list airflowmodels.DagRunList
	path := constants.AirflowDagsPath + "/" + url.PathEscape(dagID) + "/dagRuns"

	if err := c.get(ctx, path, query, &list); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %s", ErrDAGNotFound, dagID)
		}
		return nil, fmt.Errorf("recent runs for %s: %w", dagID, err)
	}

	return list.DagRuns, nil
}

// GetRecentDagRunsForDAGs fetches recent runs for multiple DAGs concurrently.
func (c *AirflowClient) GetRecentDagRunsForDAGs(ctx context.Context, dagIDs []string, limit int) (map[string][]airflowmodels.DagRun, error) {
	runsMap := make(map[string][]airflowmodels.DagRun, len(dagIDs))
	errs := make([]error, len(dagIDs))

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, constants.AirflowMaxConcurrentRequests)

	for i, dagID := range dagIDs {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			runs, err := c.GetRecentDagRuns(ctx, id, limit)
			if err != nil && !errors.Is(err, ErrDAGNotFound) {
				errs[idx] = err
				return
			}

			mu.Lock()
			defer mu.Unlock()
			runsMap[id] = runs
		}(i, dagID)
	}
	wg.Wait()

	return runsMap, errors.Join(errs...)
}

