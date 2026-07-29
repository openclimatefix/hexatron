// Command hexatron runs the Hexatron backend.
//
// For now it dumps every DAG Airflow knows about, so we can see real data
// flowing before the service layer goes on top.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/openclimatefix/hexatron/backend/internal/clients/airflow"
)

func main() {
	log.SetFlags(0)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client, err := airflow.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	dags, err := client.ListDAGs(ctx)
	if err != nil {
		log.Fatal(err)
	}

	printDAGs(os.Stdout, dags)
}

// printDAGs writes the DAGs as a table, sorted by ID.
func printDAGs(out *os.File, dags []airflow.DAG) {
	sort.Slice(dags, func(i, j int) bool { return dags[i].ID < dags[j].ID })

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATE\tDAG ID\tSCHEDULE\tDESCRIPTION")

	var active int
	for _, dag := range dags {
		if state(dag) == "active" {
			active++
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", state(dag), dag.ID, dag.Schedule, summarise(dag.Description))
	}
	w.Flush()

	fmt.Fprintf(out, "\n%d DAGs, %d active\n", len(dags), active)
}

// summarise reduces a DAG description to a single table-friendly line. Several
// OCF DAGs carry multi-paragraph docstrings.
func summarise(description string) string {
	const maxLen = 60

	line, _, _ := strings.Cut(description, "\n")
	line = strings.TrimSpace(line)
	if len(line) > maxLen {
		line = line[:maxLen-1] + "…"
	}
	return line
}

// state summarises a DAG's scheduling state. This is not run state: whether a
// run is in flight comes from the dagRuns endpoint, which is the next step.
func state(dag airflow.DAG) string {
	switch {
	case dag.HasImportErrors:
		return "broken"
	case !dag.IsActive:
		return "inactive"
	case dag.IsPaused:
		return "paused"
	default:
		return "active"
	}
}
