package printer

import (
	"context"
	"errors"
	"io"
	"text/template"
	"time"
)

// PrinterState is the backend-agnostic state of a printer.
type PrinterState string

const (
	Ready    PrinterState = "ready"
	PrePrint PrinterState = "pre_print"
	Printing PrinterState = "printing"
	Pause    PrinterState = "pause"
	Error    PrinterState = "error"

	Disconnected  PrinterState = "disconnected"
	Unknown       PrinterState = "unknown"
	InternalError PrinterState = "internal_error" // Internal error
)

// MonitorConfig holds the job-authorization enforcement policy shared by every
// backend.
type MonitorConfig struct {
	NoPauseDuration      time.Duration
	ShouldPauseProgress  float32
	ShouldCancelProgress float32
	WillPauseMessage     *template.Template
	PauseMessage         *template.Template
}

// Printer is the backend-agnostic contract implemented by every printer
// backend (moonraker.Monitor, bambu.Monitor). The web, controller and main
// packages depend only on this interface.
type Printer interface {
	// Identity / config
	PrinterName() string
	PrinterUrl() string
	PrinterType() string
	Config() MonitorConfig

	// Observed state (read by web + controller)
	State() PrinterState
	Message() string
	// ErrorDetail is non-nil only while State() is Error or InternalError and
	// the backend has something to report (a code and/or message beyond what
	// Message() already carries).
	ErrorDetail() *ErrorInfo
	LastUpdateTime() time.Time
	Job() *Job

	// Authorization state
	RegisteredJobId() string
	AllowNoRegPrint() bool
	JobPausedByMonitor() bool

	// Commands
	SetRegisteredJobId(jobId string)
	SetAllowNoRegPrint(allow bool)

	// Lifecycle
	Start(ctx context.Context)
	Stop()
}

// ErrNoThumbnail is returned by Thumbnailer.LatestThumbnail when no thumbnail
// is available for the latest job.
var ErrNoThumbnail = errors.New("no thumbnail available")

// Thumbnailer is an optional capability. Backends that can serve the latest
// job's thumbnail implement it; the web layer type-asserts for it and returns
// 501/404 when a backend does not.
type Thumbnailer interface {
	// LatestThumbnail writes the thumbnail image bytes to w and returns the
	// content type. It returns ErrNoThumbnail when none is available.
	LatestThumbnail(ctx context.Context, w io.Writer) (contentType string, err error)
}

// RawReporter is an optional capability for backends that can expose their
// raw wire-level report state verbatim (e.g. Bambu's merged MQTT print
// report), beyond what the neutral Printer DTO carries. The returned value
// must be JSON-marshalable; the web layer type-asserts for it and returns
// 501 when a backend does not implement it.
type RawReporter interface {
	RawReport() any
}
