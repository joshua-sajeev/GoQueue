package config

type JobStatus string

var (
	AllowedQueues                = []string{"default", "email", "webhooks", "payment"}
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCompleted JobStatus = "completed"
)
