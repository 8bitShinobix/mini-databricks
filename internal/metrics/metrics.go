package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// API Gateway metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	JobsSubmittedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jobs_submitted_total",
			Help: "Total number of jobs submitted",
		},
	)

	JobsCancelledTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jobs_cancelled_total",
			Help: "Total number of jobs cancelled",
		},
	)

	// Worker metrics
	TasksProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tasks_processed_total",
			Help: "Total number of tasks processed",
		},
		[]string{"state"}, // succeeded, failed, dead
	)

	TaskDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "task_duration_seconds",
			Help:    "Task processing duration in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
		},
	)

	PythonSubprocessDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "python_subprocess_duration_seconds",
			Help:    "Python subprocess execution duration in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
		},
	)

	// Scheduler metrics
	RunsCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "runs_created_total",
			Help: "Total number of runs created",
		},
	)

	TasksCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_created_total",
			Help: "Total number of tasks created",
		},
	)

	// Autoscaler metrics
	WorkerReplicasCurrent = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "worker_replicas_current",
			Help: "Current number of worker replicas",
		},
	)

	PendingTasksCurrent = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pending_tasks_current",
			Help: "Current number of pending tasks",
		},
	)

	RunningTasksCurrent = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "running_tasks_current",
			Help: "Current number of running tasks",
		},
	)
)
