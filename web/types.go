package web

import "v400_monitor/moonraker"

type APIErrorResp struct {
	Error string `json:"error"`
}

type Printer struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Url  string `json:"url"`

	RegJobId        string  `json:"registered_job_id"`
	AllowNoRegPrint bool    `json:"allow_no_register_print"`
	NoPauseDuration float64 `json:"no_pause_duration"`

	State   moonraker.PrinterState `json:"state"`
	Message string                 `json:"message"`

	DisplayStatus *moonraker.PrinterObjectDisplayStatus `json:"display_status"`
	PrinterStats  *moonraker.PrinterObjectPrintStats    `json:"printer_stats"`
	VirtualSDCard *moonraker.PrinterObjectVirtualSDCard `json:"virtual_sd_card"`

	LoadedFile *moonraker.GCodeMetadata `json:"loaded_file"`
	LatestJob  *moonraker.Job           `json:"latest_job"`
}
