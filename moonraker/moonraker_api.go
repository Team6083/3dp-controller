package moonraker

import (
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

func GetPrinterObjects(printerURL *url.URL) (*PrinterObjectsResponse, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	u := printerURL.JoinPath("/printer/objects/query")

	query := u.Query()
	query.Set("print_stats", "")
	query.Set("idle_timeout", "")
	query.Set("display_status", "")
	query.Set("toolhead", "")
	query.Set("virtual_sdcard", "")

	u.RawQuery = query.Encode()

	resp, err := client.Get(u.String())
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

func PausePrint(printerURL *url.URL) error {
	u := printerURL.JoinPath("/printer/print/pause")

	resp, err := http.Post(u.String(), "application/json", nil)
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

func ResumePrint(printerURL *url.URL) error {
	u := printerURL.JoinPath("/printer/print/resume")

	resp, err := http.Post(u.String(), "application/json", nil)
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

func RunGcode(printerURL *url.URL, script string) error {
	u := printerURL.JoinPath("/printer/gcode/script")

	query := u.Query()
	query.Set("script", script)
	u.RawQuery = query.Encode()

	resp, err := http.Get(u.String())
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

func SetStatusMessage(printerURL *url.URL, msg string) error {
	return RunGcode(printerURL, "M117 "+msg)
}
