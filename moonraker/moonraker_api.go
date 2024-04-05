package moonraker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

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

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	out := new(PrinterObjectsResponse)
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return nil, err
	}

	return out, nil
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

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("Pause " + string(b))

	return nil
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

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("Resume " + string(b))

	return nil
}

func RunGcode(ctx context.Context, script string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
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

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("RunGcode " + string(b))

	return nil
}

func SetStatusMessage(ctx context.Context, msg string) error {
	return RunGcode(ctx, "M117 "+msg)
}
