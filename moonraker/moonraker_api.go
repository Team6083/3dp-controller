package moonraker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type APIError struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Traceback string `json:"traceback"`
}

// ---------------------------
// Query Printer Object Status

type PrinterObjectDisplayStatus struct {
	Message  string  `json:"message"`
	Progress float32 `json:"progress"`
}

type PrinterObjectIdleTimeout struct {
	State        string  `json:"state"`
	PrintingTime float32 `json:"printing_time"`
}

type PrinterObjectPrintStats struct {
	FileName      string  `json:"filename"`
	TotalDuration float32 `json:"total_duration"`
	PrintDuration float32 `json:"print_duration"`
	FilamentUsed  float32 `json:"filament_used"`
	State         string  `json:"state"`
	Message       string  `json:"message"`
}

func (p *PrinterObjectPrintStats) GetPrintDuration() time.Duration {
	return time.Duration(p.PrintDuration * float32(time.Second))
}

func (p *PrinterObjectPrintStats) GetTotalDuration() time.Duration {
	return time.Duration(p.TotalDuration * float32(time.Second))
}

type PrinterObjectToolhead struct {
	PrintTime          float32 `json:"print_time"`
	EstimatedPrintTime float32 `json:"estimated_print_time"`
}

type PrinterObjectVirtualSDCard struct {
	Progress float32 `json:"progress"`
	IsActive bool    `json:"is_active"`
}

type PrinterObjectWebhooks struct {
	State        string `json:"state"`
	StateMessage string `json:"state_message"`
}

//goland:noinspection SpellCheckingInspection
type PrinterObjectsResponse struct {
	Result struct {
		EventTime float32 `json:"eventtime"`
		Status    *struct {
			DisplayStatus PrinterObjectDisplayStatus `json:"display_status"`
			IdleTimeout   PrinterObjectIdleTimeout   `json:"idle_timeout"`
			PrintStats    PrinterObjectPrintStats    `json:"print_stats"`
			Toolhead      PrinterObjectToolhead      `json:"toolhead"`
			VirtualSDCard PrinterObjectVirtualSDCard `json:"virtual_sdcard"`
			Webhooks      PrinterObjectWebhooks      `json:"webhooks"`
		} `json:"status"`
	} `json:"result"`
	Error *APIError `json:"error"`
}

func GetPrinterObjects(ctx context.Context) (*PrinterObjectsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	moonrakerAPIUrl := ctx.Value("moonrakerAPIUrl").(*url.URL)

	// build URL
	u := moonrakerAPIUrl.JoinPath("/printer/objects/query")

	query := u.Query()
	query.Set("webhooks", "")
	query.Set("print_stats", "")
	query.Set("idle_timeout", "")
	query.Set("display_status", "")
	query.Set("toolhead", "")
	query.Set("virtual_sdcard", "")
	u.RawQuery = query.Encode()

	// build request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	// do request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	out := new(PrinterObjectsResponse)
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// ---------------------------
// Get Klippy host information

type KlipperInfo struct {
	State        string `json:"state"`
	StateMessage string `json:"state_message"`
	HostName     string `json:"hostname"`
	SWVersion    string `json:"software_version"`
	CPUInfo      string `json:"cpu_info"`
	KlipperPath  string `json:"klipper_path"`
	PythonPath   string `json:"python_path"`
	LogFile      string `json:"log_file"`
	ConfigFile   string `json:"config_file"`
}

type GetKlippyHostInfoResponse struct {
	Result KlipperInfo `json:"result"`
}

func GetKlippyHostInfo(ctx context.Context) (*GetKlippyHostInfoResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	moonrakerAPIUrl := ctx.Value("moonrakerAPIUrl").(*url.URL)

	u := moonrakerAPIUrl.JoinPath("/printer/info")

	// build request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	// do request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	out := new(GetKlippyHostInfoResponse)
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// -------------
// Pause a Print

type PausePrintResponse struct {
	Result string `json:"result"`
}

func PausePrint(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	moonrakerAPIUrl := ctx.Value("moonrakerAPIUrl").(*url.URL)

	// build URL
	u := moonrakerAPIUrl.JoinPath("/printer/print/pause")

	// build request
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// do request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	out := new(PausePrintResponse)
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return err
	}

	if out.Result != "ok" {
		return errors.New(out.Result)
	}

	return nil
}

// --------------
// Resume a Print

type ResumePrintResponse struct {
	Result string `json:"result"`
}

func ResumePrint(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	moonrakerAPIUrl := ctx.Value("moonrakerAPIUrl").(*url.URL)

	// build URL
	u := moonrakerAPIUrl.JoinPath("/printer/print/resume")

	// build request
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// do request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	out := new(ResumePrintResponse)
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return err
	}

	if out.Result != "ok" {
		return errors.New(out.Result)
	}

	return nil
}

// -----------
// Run a GCode

type RunGCodeResponse struct {
	Result string `json:"result"`
}

func RunGCode(ctx context.Context, script string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	moonrakerAPIUrl := ctx.Value("moonrakerAPIUrl").(*url.URL)

	u := moonrakerAPIUrl.JoinPath("/printer/gcode/script")
	query := u.Query()
	query.Set("script", script)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	out := new(RunGCodeResponse)
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return err
	}

	if out.Result != "ok" {
		return errors.New(out.Result)
	}

	return nil
}

func SetStatusMessage(ctx context.Context, msg string) error {
	return RunGCode(ctx, "M117 "+msg)
}

// ------------------
// Get Gcode Metadata

type GCodeMetadata struct {
	PrintStartTime   float32 `json:"print_start_time"`
	JobId            string  `json:"job_id"`
	Size             int     `json:"size"`
	Modified         float32 `json:"modified"`
	UUID             string  `json:"uuid"`
	Slicer           string  `json:"slicer"`
	SlicerVersion    string  `json:"slicer_version"`
	LayerHeight      float32 `json:"layer_height"`
	FirstLayerHeight float32 `json:"first_layer_height"`
	ObjectHeight     float32 `json:"object_height"`
	FilamentTotal    float32 `json:"filament_total"`
	EstimatedTime    float32 `json:"estimated_time"`
	Thumbnails       []struct {
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		Size         int    `json:"size"`
		RelativePath string `json:"relative_path"`
	} `json:"thumbnails"`
	GCodeStartByte float32 `json:"gcode_start_byte"`
	GCodeEndByte   float32 `json:"gcode_end_byte"`
	Filename       string  `json:"filename"`
}

type GetGCodeMetaResponse struct {
	Result *GCodeMetadata `json:"result"`
	Error  *APIError      `json:"error"`
}

func GetGcodeMetadata(ctx context.Context, fileName string) (*GetGCodeMetaResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	moonrakerAPIUrl := ctx.Value("moonrakerAPIUrl").(*url.URL)

	u := moonrakerAPIUrl.JoinPath("/server/files/metadata")

	query := u.Query()
	query.Set("filename", fileName)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	out := new(GetGCodeMetaResponse)
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// ------------
// Get Job List

type Job struct {
	JobId         string         `json:"job_id"`
	Exists        bool           `json:"exists"`
	StartTime     float32        `json:"start_time"`
	EndTime       float32        `json:"end_time"`
	TotalDuration float32        `json:"total_duration"`
	PrintDuration float32        `json:"print_duration"`
	FilamentUsed  float32        `json:"filament_used"`
	Filename      string         `json:"filename"`
	Metadata      *GCodeMetadata `json:"metadata"`
	Status        string         `json:"status"`
}

type GetJobListResponse struct {
	Result *struct {
		Count int   `json:"count"`
		Jobs  []Job `json:"jobs"`
	} `json:"result"`
	Error *APIError `json:"error"`
}

type GetJobListOrder string

const (
	OrderAsc  GetJobListOrder = "asc"
	OrderDesc GetJobListOrder = "desc"
)

type GetJobListParams struct {
	Limit  *int
	Start  *int
	Since  *time.Time
	Before *time.Time
	Order  GetJobListOrder
}

func GetJobList(ctx context.Context, params GetJobListParams) (*GetJobListResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	moonrakerAPIUrl := ctx.Value("moonrakerAPIUrl").(*url.URL)

	u := moonrakerAPIUrl.JoinPath("/server/history/list")

	query := u.Query()
	if params.Limit != nil {
		query.Set("limit", fmt.Sprintf("%d", *params.Limit))
	}
	if params.Start != nil {
		query.Set("start", fmt.Sprintf("%d", *params.Start))
	}
	if params.Since != nil {
		query.Set("since", fmt.Sprintf("%d", (*params.Since).UnixMilli()*1000))
	}
	if params.Before != nil {
		query.Set("before", fmt.Sprintf("%d", (*params.Before).UnixMilli()*1000))
	}
	if params.Order == OrderAsc {
		query.Set("order", "asc")
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	out := new(GetJobListResponse)
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GetLatestJob(ctx context.Context) (*GetJobListResponse, error) {
	limit := 1
	return GetJobList(ctx, GetJobListParams{Limit: &limit})
}
