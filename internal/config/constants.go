package config

type JobStatus string

var (
	AllowedQueues                = []string{"default", "email", "webhooks", "payment"}
	AllowedJobTypes              = []string{"send_email", "process_payment", "send_webhook"}
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCompleted JobStatus = "completed"
)
