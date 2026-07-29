// Package airflow is a client for the Airflow REST API.
//
// It is the only component in Hexatron aware of Airflow's API shape: everything
// above it works in terms of the types declared here, never raw JSON.
//
// Authentication is a session cookie carried on every request. That is a
// development-only arrangement — production will use a service account.
package airflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	// sessionCookieName is the cookie Airflow's web session auth expects.
	sessionCookieName = "session"

	defaultTimeout = 15 * time.Second

	// pageSize is the per-request page size for list endpoints. Airflow caps
	// this at 100 by default.
	pageSize = 100
)

// Client talks to a single Airflow deployment.
type Client struct {
	baseURL       *url.URL
	sessionCookie string
	httpClient    *http.Client
}

// New builds a client for the Airflow instance at baseURL, authenticating with
// the given session cookie value.
func New(baseURL, sessionCookie string) (*Client, error) {
	if baseURL == "" {
		return nil, errors.New("airflow: base URL is required")
	}
	if sessionCookie == "" {
		return nil, errors.New("airflow: session cookie is required")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("airflow: invalid base URL %q: %w", baseURL, err)
	}

	return &Client{
		baseURL:       parsed,
		sessionCookie: sessionCookie,
		httpClient:    &http.Client{Timeout: defaultTimeout},
	}, nil
}

// NewFromEnv builds a client from AIRFLOW_BASE_URL and AIRFLOW_SESSION_COOKIE.
func NewFromEnv() (*Client, error) {
	client, err := New(os.Getenv("AIRFLOW_BASE_URL"), os.Getenv("AIRFLOW_SESSION_COOKIE"))
	if err != nil {
		return nil, fmt.Errorf("%w (set AIRFLOW_BASE_URL and AIRFLOW_SESSION_COOKIE)", err)
	}
	return client, nil
}

// APIError is an error response from the Airflow REST API. A stale or missing
// session cookie surfaces here as a 401.
type APIError struct {
	StatusCode int    `json:"status"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Type       string `json:"type"`
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
// needs refreshing.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

// get performs a GET against the Airflow API and decodes the JSON body into out.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("airflow: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: c.sessionCookie})

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

// DAG is a workflow registered in Airflow.
type DAG struct {
	ID           string `json:"dag_id"`
	DisplayName  string `json:"dag_display_name"`
	Description  string `json:"description"`
	FileLocation string `json:"fileloc"`

	Owners []string `json:"owners"`
	Tags   []Tag    `json:"tags"`

	// IsActive reports whether the DAG is still present in the DAG bag; it goes
	// false when a DAG file is deleted. IsPaused reports whether scheduling is
	// switched off. Neither says anything about a run currently in flight.
	IsActive        bool `json:"is_active"`
	IsPaused        bool `json:"is_paused"`
	HasImportErrors bool `json:"has_import_errors"`

	Schedule             *Schedule `json:"schedule_interval"`
	TimetableDescription string    `json:"timetable_description"`

	NextDagRun     *time.Time `json:"next_dagrun"`
	LastParsedTime *time.Time `json:"last_parsed_time"`
}

// Name is the DAG's display name, falling back to its ID.
func (d DAG) Name() string {
	if d.DisplayName != "" {
		return d.DisplayName
	}
	return d.ID
}

// Tag is a label attached to a DAG.
type Tag struct {
	Name string `json:"name"`
}

// Schedule is Airflow's polymorphic schedule_interval field. Every OCF DAG
// currently uses the CronExpression variant, which carries a cron string in
// Value; the TimeDelta variant uses the duration fields instead.
type Schedule struct {
	Type  string `json:"__type"`
	Value string `json:"value"`

	Days         int `json:"days"`
	Seconds      int `json:"seconds"`
	Microseconds int `json:"microseconds"`
}

func (s *Schedule) String() string {
	switch {
	case s == nil:
		return ""
	case s.Value != "":
		return s.Value
	default:
		d := time.Duration(s.Days)*24*time.Hour +
			time.Duration(s.Seconds)*time.Second +
			time.Duration(s.Microseconds)*time.Microsecond
		return d.String()
	}
}

// dagList is the envelope returned by the /dags endpoint.
type dagList struct {
	DAGs         []DAG `json:"dags"`
	TotalEntries int   `json:"total_entries"`
}

// ListDAGs returns every DAG registered in Airflow, following pagination.
func (c *Client) ListDAGs(ctx context.Context) ([]DAG, error) {
	var all []DAG

	for {
		query := url.Values{}
		query.Set("limit", strconv.Itoa(pageSize))
		query.Set("offset", strconv.Itoa(len(all)))

		var page dagList
		if err := c.get(ctx, "/api/v1/dags", query, &page); err != nil {
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
