package moonraker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
)

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

type PrinterObjectToolhead struct {
	PrintTime          float32 `json:"print_time"`
	EstimatedPrintTime float32 `json:"estimated_print_time"`
}

type PrinterObjectVirtualSDCard struct {
	Progress float32 `json:"progress"`
	IsActive bool    `json:"is_active"`
}

type PrinterObjectsResponse struct {
	Result struct {
		EventTime float32 `json:"eventtime"`
		Status    struct {
			DisplayStatus PrinterObjectDisplayStatus `json:"display_status"`
			IdleTimeout   PrinterObjectIdleTimeout   `json:"idle_timeout"`
			PrintStats    PrinterObjectPrintStats    `json:"print_stats"`
			Toolhead      PrinterObjectToolhead      `json:"toolhead"`
			VirtualSDCard PrinterObjectVirtualSDCard `json:"virtual_sdcard"`
		} `json:"status"`
	} `json:"result"`
}

func GetPrinterObjects(ctx context.Context) (*PrinterObjectsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	moonrakerAPIUrl := ctx.Value("moonrakerAPIUrl").(*url.URL)

	// build URL
	u := moonrakerAPIUrl.JoinPath("/printer/objects/query")

	query := u.Query()
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

// Get Klippy host information

type GetKlippyHostInfoResponse struct {
	Result struct {
		State        string `json:"state"`
		StateMessage string `json:"state_message"`
		HostName     string `json:"hostname"`
		SWVersion    string `json:"software_version"`
		CPUInfo      string `json:"cpu_info"`
		KlipperPath  string `json:"klipper_path"`
		PythonPath   string `json:"python_path"`
		LogFile      string `json:"log_file"`
		ConfigFile   string `json:"config_file"`
	} `json:"result"`
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

// Run a GCode

type RunGcodeResponse struct {
	Result string `json:"result"`
}

func RunGcode(ctx context.Context, script string) error {
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

	out := new(RunGcodeResponse)
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
	return RunGcode(ctx, "M117 "+msg)
}
