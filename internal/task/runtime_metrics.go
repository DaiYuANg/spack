package task

import (
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type RuntimeMetrics struct {
	schedulerRunning prometheus.Gauge
	schedulerEvents  *prometheus.CounterVec
	jobEvents        *prometheus.CounterVec
	jobExecution     *prometheus.HistogramVec
	jobDelay         *prometheus.HistogramVec
	limitReached     *prometheus.CounterVec
	jobsRegistered   *prometheus.GaugeVec
	jobsRunning      *prometheus.GaugeVec
}

func NewRuntimeMetrics() *RuntimeMetrics {
	return &RuntimeMetrics{
		schedulerRunning: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "spack_task_scheduler_running",
			Help: "Whether the background task scheduler is currently running.",
		}),
		schedulerEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "spack_task_scheduler_events_total",
			Help: "Total number of background task scheduler lifecycle events.",
		}, []string{"event"}),
		jobEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "spack_task_scheduler_job_events_total",
			Help: "Total number of background task scheduler job events.",
		}, []string{"job", "event"}),
		jobExecution: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "spack_task_scheduler_job_execution_seconds",
			Help:    "Background task scheduler job execution time in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"job"}),
		jobDelay: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "spack_task_scheduler_job_scheduling_delay_seconds",
			Help:    "Background task scheduler delay between scheduled and actual start time in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"job"}),
		limitReached: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "spack_task_scheduler_concurrency_limit_total",
			Help: "Total number of background task scheduler concurrency limit hits.",
		}, []string{"job", "limit_type"}),
		jobsRegistered: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "spack_task_scheduler_jobs_registered_current",
			Help: "Current number of registered background task scheduler jobs by job name.",
		}, []string{"job"}),
		jobsRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "spack_task_scheduler_jobs_running_current",
			Help: "Current number of running background task scheduler jobs by job name.",
		}, []string{"job"}),
	}
}

func (m *RuntimeMetrics) Collectors() []prometheus.Collector {
	if m == nil {
		return nil
	}
	return []prometheus.Collector{
		m.schedulerRunning,
		m.schedulerEvents,
		m.jobEvents,
		m.jobExecution,
		m.jobDelay,
		m.limitReached,
		m.jobsRegistered,
		m.jobsRunning,
	}
}

func (m *RuntimeMetrics) SchedulerStarted() {
	if m == nil {
		return
	}
	m.schedulerRunning.Set(1)
	m.schedulerEvents.WithLabelValues("started").Inc()
}

func (m *RuntimeMetrics) SchedulerStopped() {
	if m == nil {
		return
	}
	m.schedulerRunning.Set(0)
	m.schedulerEvents.WithLabelValues("stopped").Inc()
}

func (m *RuntimeMetrics) SchedulerShutdown() {
	if m == nil {
		return
	}
	m.schedulerRunning.Set(0)
	m.schedulerEvents.WithLabelValues("shutdown").Inc()
}

func (m *RuntimeMetrics) JobRegistered(job gocron.Job) {
	if m == nil {
		return
	}
	name := schedulerJobName(job)
	m.jobEvents.WithLabelValues(name, "registered").Inc()
	m.jobsRegistered.WithLabelValues(name).Inc()
}

func (m *RuntimeMetrics) JobUnregistered(job gocron.Job) {
	if m == nil {
		return
	}
	name := schedulerJobName(job)
	m.jobEvents.WithLabelValues(name, "unregistered").Inc()
	m.jobsRegistered.WithLabelValues(name).Dec()
}

func (m *RuntimeMetrics) JobStarted(job gocron.Job) {
	if m == nil {
		return
	}
	name := schedulerJobName(job)
	m.jobEvents.WithLabelValues(name, "started").Inc()
	m.jobsRunning.WithLabelValues(name).Inc()
}

func (m *RuntimeMetrics) JobRunning(job gocron.Job) {
	if m == nil {
		return
	}
	m.jobEvents.WithLabelValues(schedulerJobName(job), "running").Inc()
}

func (m *RuntimeMetrics) JobFailed(job gocron.Job, err error) {
	_ = err
	if m == nil {
		return
	}
	name := schedulerJobName(job)
	m.jobEvents.WithLabelValues(name, "failed").Inc()
	m.jobsRunning.WithLabelValues(name).Dec()
}

func (m *RuntimeMetrics) JobCompleted(job gocron.Job) {
	if m == nil {
		return
	}
	name := schedulerJobName(job)
	m.jobEvents.WithLabelValues(name, "completed").Inc()
	m.jobsRunning.WithLabelValues(name).Dec()
}

func (m *RuntimeMetrics) JobExecutionTime(job gocron.Job, duration time.Duration) {
	if m == nil {
		return
	}
	m.jobExecution.WithLabelValues(schedulerJobName(job)).Observe(duration.Seconds())
}

func (m *RuntimeMetrics) JobSchedulingDelay(job gocron.Job, scheduledTime, actualStartTime time.Time) {
	if m == nil {
		return
	}
	delay := max(actualStartTime.Sub(scheduledTime), time.Duration(0))
	m.jobDelay.WithLabelValues(schedulerJobName(job)).Observe(delay.Seconds())
}

func (m *RuntimeMetrics) ConcurrencyLimitReached(limitType string, job gocron.Job) {
	if m == nil {
		return
	}
	m.limitReached.WithLabelValues(schedulerJobName(job), normalizeLimitType(limitType)).Inc()
}

func schedulerJobName(job gocron.Job) string {
	if job == nil {
		return "unknown"
	}
	name := strings.TrimSpace(job.Name())
	if name == "" {
		return "unknown"
	}
	return name
}

func normalizeLimitType(limitType string) string {
	clean := strings.TrimSpace(limitType)
	if clean == "" {
		return "unknown"
	}
	return clean
}
