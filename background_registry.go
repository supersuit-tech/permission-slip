package main

import (
	"context"
	"log/slog"

	"github.com/supersuit-tech/permission-slip-web/api"
)

// BackgroundJob describes a self-registering background job.
// Each job provides a Name for logging and a Start function that launches
// the goroutine and returns a done channel (closed when the goroutine exits).
// Start may return nil if required dependencies are absent — callers skip nil channels.
type BackgroundJob struct {
	Name  string
	Start func(ctx context.Context, deps *api.Deps, logger *slog.Logger) <-chan struct{}
}

var backgroundJobs []BackgroundJob

// RegisterBackgroundJob appends a job to the global registry.
// Call this from an init() function in each job file.
func RegisterBackgroundJob(job BackgroundJob) {
	backgroundJobs = append(backgroundJobs, job)
}
