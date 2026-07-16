package printer

import "time"

// This file holds the neutral data-transfer types shared by all printer
// backends and consumed by the web and controller layers. Each backend
// (moonraker, bambu) computes these from its own wire format; neither
// backend's raw shape leaks through here. They also double as the web API's
// JSON DTOs (json tags below) — internal/web references them directly rather
// than redefining near-duplicate copies.

// Seconds is a duration expressed in (fractional) seconds for JSON. swag
// (this project's OpenAPI generator) reads a field's Go type statically to
// build the schema, so a plain time.Duration would be described as its
// default JSON encoding (raw nanoseconds) — wrong, since callers want
// seconds. Construct with Seconds(d.Seconds()).
type Seconds float64

// ErrorInfo describes a backend's error state in more detail than
// PrinterState.Error/InternalError alone conveys. Code is a backend-specific
// raw code, nil when the backend has none available — there's no
// cross-vendor decoding of it (e.g. Bambu's print_error/hms codes are
// undocumented). Message is human-readable text when the backend has any;
// may be empty. Unrelated to Printer.Message(), which is a general
// status/display-text concept, not specifically about errors.
type ErrorInfo struct {
	Code    *int   `json:"code"`
	Message string `json:"message"`
}

// Job is the neutral representation of a print job, returned by
// Printer.Job(). It's non-nil once any job has been observed.
// Progress/PrintDuration/TotalDuration/EstimatedRemaining are non-nil only
// while that job is actively pre-printing/printing/paused — nil once it's no
// longer active, since they're meaningless at that point. The
// identity/outcome fields below are always populated once a job exists,
// whether or not it's still active.
type Job struct {
	JobId        string `json:"job_id"`
	Name         string `json:"name"`
	HasThumbnail bool   `json:"has_thumbnail"`

	// Identity/outcome — always populated once a job exists.
	Status string `json:"status"` // "in_progress" | "completed" | "cancelled" | "interrupted" | "error" | "quit" | ...
	// StartTime/EndTime are nil when a backend genuinely doesn't know them
	// (e.g. EndTime while a job is still in progress) — not fabricated as a
	// zero time.
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	// ContentId is an opaque content identifier (e.g. Moonraker's
	// slicer-computed file UUID); empty when the backend has no such concept.
	// Do not repurpose this for anything else (display titles, etc.).
	ContentId string `json:"content_id"`

	// Live progress — non-nil only while the job is active. Nil when a
	// backend genuinely doesn't know a value — not every printer
	// model/vendor exposes the same information, so callers must handle
	// absence rather than receiving a fabricated value.
	Progress      *float32 `json:"progress"` // 0..1, unified across vendors
	PrintDuration *Seconds `json:"print_duration"`
	// TotalDuration is best-effort; it may equal PrintDuration if no better
	// estimate exists.
	TotalDuration      *Seconds `json:"total_duration"`
	EstimatedRemaining *Seconds `json:"remaining_sec"`
}
