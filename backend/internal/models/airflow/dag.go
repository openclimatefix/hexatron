// Package airflow holds Airflow domain models used in Phase 2.
package airflow

import "time"

// DAG represents an Airflow DAG definition returned by GET /api/v1/dags.
type DAG struct {
	DAGID       string `json:"dag_id"`
	DisplayName string `json:"dag_display_name"`
	Description string `json:"description"`

	// IsActive reports whether the DAG is still present in the DAG bag; it goes
	// false when a DAG file is deleted. IsPaused reports whether scheduling is
	// switched off. Neither says anything about a run currently in flight.
	IsActive        bool `json:"is_active"`
	IsPaused        bool `json:"is_paused"`
	HasImportErrors bool `json:"has_import_errors"`

	Schedule             *Schedule  `json:"schedule_interval"`
	TimetableDescription string     `json:"timetable_description"`
	NextDagRun           *time.Time `json:"next_dagrun"`
}

// Name is the DAG's display name, falling back to its ID.
func (d DAG) Name() string {
	if d.DisplayName != "" {
		return d.DisplayName
	}
	return d.DAGID
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

// DAGList is the envelope returned by GET /api/v1/dags.
type DAGList struct {
	DAGs         []DAG `json:"dags"`
	TotalEntries int   `json:"total_entries"`
}
