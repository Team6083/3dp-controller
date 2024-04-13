package api

import "time"

type OperationState string

const (
	OpStateOutOfService OperationState = "out_of_service"
	OpStateNormal       OperationState = "normal"
	OpStateClosed       OperationState = "closed"
)

type ControlSetting struct {
	OpState           OperationState `json:"state"`
	EnableMaintenance bool           `json:"enable_maintenance"`

	IsActive      bool   `json:"is_active"`
	UsageRecordId string `json:"usage_record_id"`
}

type UpdateMessage struct {
	Key   string `json:"key"`
	State Report `json:"state"`
}

type ControlMessage struct {
	Key            string         `json:"key"`
	ControlSetting ControlSetting `json:"control_state"`
	ActiveJobId    string         `json:"active_job_id"`
}

type Status string

const (
	StatusIdle         Status = "idle"
	StatusRunning      Status = "running"
	StatusPaused       Status = "paused"
	StatusError        Status = "error"
	StatusDisconnected Status = "disconnected"
	StatusUnknown      Status = "unknown"
)

type Report struct {
	Status                Status         `json:"status"`
	JobReport             JobReport      `json:"job_report"`
	CurrentControlSetting ControlSetting `json:"current_control_setting"`
}

type ReportJobStatus string

const (
	ReportInProgress ReportJobStatus = "in_progress"
	ReportDone       ReportJobStatus = "done"
	ReportQuit       ReportJobStatus = "quit"
)

type JobReport struct {
	Id     string          `json:"id"`
	Status ReportJobStatus `json:"status"`

	ContentId string    `json:"content_id"`
	StartTime time.Time `json:"start_time"`
}
