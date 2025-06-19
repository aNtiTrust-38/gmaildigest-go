package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// JobsScheduled is a counter for jobs scheduled.
	JobsScheduled = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gmaildigest_jobs_scheduled_total",
			Help: "The total number of jobs scheduled.",
		},
		[]string{"job_type"},
	)

	// JobsCompleted is a counter for jobs completed successfully.
	JobsCompleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gmaildigest_jobs_completed_total",
			Help: "The total number of jobs completed successfully.",
		},
		[]string{"job_type"},
	)

	// JobsFailed is a counter for jobs that failed.
	JobsFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gmaildigest_jobs_failed_total",
			Help: "The total number of jobs that failed.",
		},
		[]string{"job_type"},
	)

	// JobRetries is a counter for job retries.
	JobRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gmaildigest_job_retries_total",
			Help: "The total number of times a job has been retried.",
		},
		[]string{"job_type"},
	)

	// JobDuration is a histogram of the time it takes to execute a job.
	JobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gmaildigest_job_duration_seconds",
			Help:    "A histogram of the job execution duration.",
			Buckets: prometheus.LinearBuckets(0.1, 0.1, 10), // 10 buckets, 0.1s width
		},
		[]string{"job_type"},
	)

	// JobsInFlight is a gauge that shows the number of currently running jobs.
	JobsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gmaildigest_jobs_in_flight",
			Help: "The number of jobs currently being executed.",
		},
	)
) 