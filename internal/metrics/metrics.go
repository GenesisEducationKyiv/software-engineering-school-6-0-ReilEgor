package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "app_http_requests_total",
	Help: "Total HTTP requests by method, route, and status code.",
}, []string{"method", "route", "status_code"})

var HTTPRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "app_http_request_duration_seconds",
	Help:    "HTTP request duration in seconds by method and route.",
	Buckets: prometheus.DefBuckets,
}, []string{"method", "route"})

var UsecaseOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "app_usecase_operations_total",
	Help: "Total use case operations by operation name and status.",
}, []string{"operation", "status"})

var UsecaseOperationDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "app_usecase_operation_duration_seconds",
	Help:    "Use case operation duration in seconds.",
	Buckets: prometheus.DefBuckets,
}, []string{"operation"})

var NotificationsProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "app_notifications_processed_total",
	Help: "Total notification processing runs by status.",
}, []string{"status"})

var NotificationProcessingDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "app_notification_processing_duration_seconds",
	Help:    "Duration of full notification batch processing in seconds.",
	Buckets: prometheus.DefBuckets,
})

var NotificationEmailsSentTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "app_notification_emails_sent_total",
	Help: "Total notification emails attempted by status.",
}, []string{"status"})

var ConfirmationEmailsSentTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "app_confirmation_emails_sent_total",
	Help: "Total subscription confirmation emails attempted by status.",
}, []string{"status"})
