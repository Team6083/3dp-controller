package web

import "v400_monitor/moonraker"

type Printer struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Url  string `json:"url"`

	RegJobId        string `json:"registered_job_id"`
	AllowNoRegPrint bool   `json:"allow_no_register_print"`

	State          moonraker.PrinterState           `json:"state"`
	PrinterObjects *moonraker.MonitorPrinterObjects `json:"printer_objects"`
	LoadedFile     *moonraker.GCodeMetadata         `json:"loaded_file"`
	LatestJob      *moonraker.Job                   `json:"latest_job"`
}
