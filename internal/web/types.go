package web

import "3dp-controller/internal/printer"

type APIErrorResp struct {
	Error string `json:"error"`
}

type Printer struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Url  string `json:"url"`
	Type string `json:"type"`

	RegJobId        string  `json:"registered_job_id"`
	AllowNoRegPrint bool    `json:"allow_no_register_print"`
	NoPauseDuration float64 `json:"no_pause_duration"`

	State          printer.PrinterState `json:"state"`
	Message        string               `json:"message"`
	ErrorDetail    *printer.ErrorInfo   `json:"error_detail"`
	LastUpdateTime int64                `json:"last_update_time"`

	Job *printer.Job `json:"job"`
}
