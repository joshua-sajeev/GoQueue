package config

var (
	AllowedQueues   = []string{"default", "email", "webhooks"}
	AllowedJobTypes = []string{"send_email", "process_payment", "send_webhook"}
)
