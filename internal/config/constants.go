package config

var (
	AllowedQueues   = []string{"default", "email", "reports", "webhooks"}
	AllowedJobTypes = []string{"send_email", "process_payment", "generate_report", "send_webhook"}
)
