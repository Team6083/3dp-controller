package moonraker

import (
	"3dp-controller/internal/printer"
	"3dp-controller/internal/util"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

var _ printer.Printer = (*Monitor)(nil)
var _ printer.Thumbnailer = (*Monitor)(nil)

type MonitorPrinterObjects struct {
	DisplayStatus PrinterObjectDisplayStatus
	IdleTimeout   PrinterObjectIdleTimeout
	PrintStats    PrinterObjectPrintStats
	VirtualSDCard PrinterObjectVirtualSDCard
	Webhooks      PrinterObjectWebhooks
}

type Monitor struct {
	printerName string
	printerUrl  *url.URL
	logger      *zap.SugaredLogger
	config      printer.MonitorConfig

	registeredJobId    string
	allowNoRegPrint    bool
	jobPausedByMonitor bool
	lastMessage        string

	state          printer.PrinterState
	lastError      *printer.ErrorInfo
	lastUpdateTime time.Time
	printerObjects *MonitorPrinterObjects
	hasLoadedFile  bool

	latestJob  *Job
	loadedFile *GCodeMetadata

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (m *Monitor) PrinterName() string {
	return m.printerName
}

func (m *Monitor) PrinterUrl() string {
	return m.printerUrl.String()
}

func (m *Monitor) PrinterType() string {
	return "moonraker"
}

func (m *Monitor) Config() printer.MonitorConfig {
	return m.config
}

func (m *Monitor) State() printer.PrinterState {
	return m.state
}

func (m *Monitor) Message() string {
	if m.printerObjects == nil {
		return ""
	}

	if m.printerObjects.Webhooks.State != "ready" {
		return m.printerObjects.Webhooks.StateMessage
	}

	return m.printerObjects.PrintStats.Message
}

func (m *Monitor) ErrorDetail() *printer.ErrorInfo {
	if m.state != printer.Error && m.state != printer.InternalError {
		return nil
	}

	return m.lastError
}

func (m *Monitor) LastUpdateTime() time.Time {
	return m.lastUpdateTime
}

func (m *Monitor) Job() *printer.Job {
	if m.latestJob == nil {
		return nil
	}

	job := m.latestJob

	j := &printer.Job{
		JobId:  job.JobId,
		Name:   job.Filename,
		Status: job.Status,
	}

	if job.Metadata != nil {
		j.ContentId = job.Metadata.UUID
		j.HasThumbnail = len(job.Metadata.Thumbnails) > 0
	}

	if job.StartTime > 0 {
		t := time.UnixMilli(int64(job.StartTime * 1000))
		j.StartTime = &t
	}

	if job.EndTime > 0 {
		t := time.UnixMilli(int64(job.EndTime * 1000))
		j.EndTime = &t
	}

	if job.Status == "in_progress" && m.printerObjects != nil {
		progress := m.printerObjects.VirtualSDCard.Progress
		j.Progress = &progress

		printDurationSec := m.printerObjects.PrintStats.GetPrintDuration().Seconds()
		printDuration := printer.Seconds(printDurationSec)
		j.PrintDuration = &printDuration

		totalDuration := printer.Seconds(m.printerObjects.PrintStats.GetTotalDuration().Seconds())
		j.TotalDuration = &totalDuration

		var estRemainSec float64
		haveEstimate := false

		if m.loadedFile != nil && m.loadedFile.EstimatedTime != nil {
			estimatedTime := float64(*m.loadedFile.EstimatedTime)
			estRemainSec = estimatedTime - float64(progress)*estimatedTime
			haveEstimate = true
		} else if progress > 0 {
			totalTime := printDurationSec / float64(progress)
			estRemainSec = totalTime - printDurationSec
			haveEstimate = true
		}

		if haveEstimate {
			if estRemainSec < 0 {
				estRemainSec = 0
			}

			remaining := printer.Seconds(estRemainSec)
			j.EstimatedRemaining = &remaining
		}
	}

	return j
}

func (m *Monitor) RegisteredJobId() string {
	return m.registeredJobId
}

func (m *Monitor) AllowNoRegPrint() bool {
	return m.allowNoRegPrint
}

func (m *Monitor) JobPausedByMonitor() bool {
	return m.jobPausedByMonitor
}

func (m *Monitor) SetRegisteredJobId(jobId string) {
	m.registeredJobId = jobId

	if m.ctx != nil && jobId != "" {
		err := m.clearMessage(m.ctx)

		if err != nil {
			m.logger.Errorf("Error clearing message: %s\n", err)
		}
	}
}

func (m *Monitor) SetAllowNoRegPrint(allowNoRegPrint bool) {
	m.allowNoRegPrint = allowNoRegPrint

	if m.ctx != nil && allowNoRegPrint {
		err := m.clearMessage(m.ctx)
		if err != nil {
			m.logger.Errorf("Error clearing message: %s\n", err)
		}
	}
}

func NewMonitor(name string, printerURL string, config printer.MonitorConfig, logger *zap.SugaredLogger) (*Monitor, error) {
	m := new(Monitor)

	u, err := url.Parse(printerURL)
	if err != nil {
		return nil, err
	}

	m.printerName = name
	m.printerUrl = u
	m.logger = logger
	m.config = config

	m.registeredJobId = ""
	m.allowNoRegPrint = true
	m.jobPausedByMonitor = false

	m.state = printer.Disconnected
	m.lastUpdateTime = time.Now()
	m.hasLoadedFile = false

	return m, nil
}

func (m *Monitor) Start(ctx context.Context) {
	if m.ctx != nil {
		return
	}

	ctx = context.WithValue(ctx, "moonrakerAPIUrl", m.printerUrl)

	ctx, cancel := context.WithCancel(ctx)
	m.ctx = ctx
	m.cancelFunc = cancel

	ticker1Duration := 2 * time.Second
	ticker1 := time.NewTicker(ticker1Duration)

	ticker2Duration := 5 * time.Second
	ticker2 := time.NewTicker(ticker2Duration)

	go func() {
		m.update()

		for {
			select {
			case <-ctx.Done():
				ticker1.Stop()
				return
			case <-ticker1.C:
				m.update()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker2.Stop()
				return
			case <-ticker2.C:
				// Update latest job
				go func() {
					if m.state == printer.Disconnected || m.state == printer.InternalError {
						m.latestJob = nil
						return
					}

					ctx2, cancel2 := context.WithTimeout(ctx, ticker2Duration)
					defer cancel2()

					job, err := m.getLatestJob(ctx2)
					if err != nil {
						m.logger.Errorf("Failed to get latest job: %s\n", err)
						return
					}
					m.latestJob = job

					if m.latestJob == nil {
						m.logger.Warnln("No latest job found")
						return
					}

					// Clear registeredJobId if job is not in_progress, or jobId not match
					if m.latestJob.Status != "in_progress" || m.latestJob.JobId != m.registeredJobId {
						m.registeredJobId = ""
					}
				}()

				// Update loaded file
				go func() {
					if m.state == printer.Disconnected || m.state == printer.InternalError {
						m.loadedFile = nil
						return
					}

					ctx2, cancel2 := context.WithTimeout(ctx, ticker2Duration)
					defer cancel2()

					metadata, err := m.getLoadedFile(ctx2)
					if err != nil {
						m.logger.Errorf("Failed to get loaded file: %s\n", err)
						return
					}

					m.loadedFile = metadata
				}()
			}
		}
	}()
}

func (m *Monitor) Stop() {
	if m.ctx != nil {
		m.cancelFunc()

		m.ctx = nil
		m.cancelFunc = nil
	}
}

func (m *Monitor) update() {
	printerObjectsResponse, err := GetPrinterObjects(m.ctx)

	m.lastUpdateTime = time.Now()

	if err != nil {
		m.printerObjects = nil
		m.hasLoadedFile = false

		var nonOkErr ERRRespNotOk
		if util.IsErrNetworkProblem(err) {
			m.state = printer.Disconnected
			m.lastError = nil
		} else if errors.As(err, &nonOkErr) {
			if nonOkErr.RespStatusCode() == 502 {
				m.state = printer.Disconnected
				m.lastError = nil
			} else {
				m.state = printer.InternalError
				code := nonOkErr.RespStatusCode()
				m.lastError = &printer.ErrorInfo{Code: &code, Message: err.Error()}
				m.logger.Warnf(
					"Failed to get printer objects: %s, status_code: %d\n", err, nonOkErr.RespStatusCode(),
				)
			}
		} else {
			m.state = printer.InternalError
			m.lastError = &printer.ErrorInfo{Message: err.Error()}
			m.logger.Errorf("Error getting printer objects: %s\n", err)
		}
	} else {
		if printerObjectsResponse.Result.Status == nil {
			m.state = printer.Error
			m.hasLoadedFile = false

			code := printerObjectsResponse.Error.Code
			m.lastError = &printer.ErrorInfo{Code: &code, Message: printerObjectsResponse.Error.Message}

			m.logger.Errorf("MoonrakerError: %d %s\n",
				printerObjectsResponse.Error.Code, printerObjectsResponse.Error.Message)
		} else {
			m.lastError = nil

			status := printerObjectsResponse.Result.Status

			printerObjects := new(MonitorPrinterObjects)
			m.printerObjects = printerObjects

			printerObjects.DisplayStatus = status.DisplayStatus
			printerObjects.IdleTimeout = status.IdleTimeout
			printerObjects.PrintStats = status.PrintStats
			printerObjects.VirtualSDCard = status.VirtualSDCard
			printerObjects.Webhooks = status.Webhooks

			if status.Webhooks.State != "ready" {
				m.hasLoadedFile = false

				switch status.Webhooks.State {
				case "startup":
					m.state = printer.Unknown
				case "shutdown", "error", "disconnected":
					m.state = printer.Error
				default:
					m.state = printer.Unknown
				}
			} else {
				printerShouldPrint := m.allowNoRegPrint || m.registeredJobId != ""
				printDuration := printerObjects.PrintStats.GetPrintDuration()

				switch printerObjects.PrintStats.State {
				case "standby", "complete", "cancelled":
					m.state = printer.Ready
				case "printing":
					if printDuration > 0 {
						m.state = printer.Printing
					} else {
						m.state = printer.PrePrint
					}
				case "paused":
					m.state = printer.Pause
				case "error":
					m.state = printer.Error
				default:
					m.state = printer.Unknown
				}

				m.hasLoadedFile = printerObjects.PrintStats.State != "standby" &&
					m.state != printer.Error && m.state != printer.Unknown

				// Check if printer is illegally printing
				if m.state == printer.Printing && !printerShouldPrint {
					m.logger.Infoln("Printer should not print now!!")

					progress := printerObjects.VirtualSDCard.Progress

					if printDuration > m.config.NoPauseDuration ||
						(m.config.ShouldPauseProgress > 0 && progress >= m.config.ShouldPauseProgress) {
						m.jobPausedByMonitor = true
					}

					if m.config.ShouldCancelProgress > 0 && progress >= m.config.ShouldCancelProgress {
						if m.state == printer.Printing {
							m.logger.Infoln("Canceling")
							err := CancelPrint(m.ctx)
							if err != nil {
								m.logger.Errorf("Failed to cancel printing: %s\n", err)
							}
						}
					}
				}

				// Pause printer if printer should be paused by monitor
				if m.state == printer.Printing && m.jobPausedByMonitor {
					m.logger.Infoln("Pausing")

					err := PausePrint(m.ctx)
					if err != nil {
						m.logger.Errorf("Error pausing the printer: %s\n", err)
					}

					data := struct {
					}{}

					var tpl bytes.Buffer
					if err := m.config.PauseMessage.Execute(&tpl, data); err != nil {
						m.logger.Errorf("Error pausing the pause message: %s\n", err)
					} else {
						err := m.updateStatusMessage(m.ctx, tpl.String())
						if err != nil {
							m.logger.Errorln(err)
						}
					}
				}

				// Show warning countdown if printer will be paused
				if m.state == printer.Printing && !m.jobPausedByMonitor && !printerShouldPrint {
					remDuration := (m.config.NoPauseDuration - printDuration).Round(time.Second)

					data := struct {
						RemainDurationStr string
					}{remDuration.String()}

					var tpl bytes.Buffer
					if err := m.config.WillPauseMessage.Execute(&tpl, data); err != nil {
						m.logger.Errorf("Error pausing the will pause message: %s\n", err)
					} else {
						err := m.updateStatusMessage(m.ctx, tpl.String())
						if err != nil {
							m.logger.Errorln(err)
						}
					}
				} else {
					// TODO: clear will pause message
				}

				// Resume print if allow print set to true
				if m.jobPausedByMonitor && printerShouldPrint {

					if m.state == printer.Pause {
						m.logger.Infoln("Resuming")

						err := ResumePrint(m.ctx)
						if err != nil {
							m.logger.Errorf("Error resuming the printer: %s\n", err)
						}

						err = m.clearMessage(m.ctx)
						if err != nil {
							m.logger.Errorln(err)
						}
					}

					m.jobPausedByMonitor = false
				}
			}
		}
	}
	//m.logger.Debugf("Status: %s\n", m.state)
}

func (m *Monitor) updateStatusMessage(ctx context.Context, msg string) error {
	if m.printerObjects.DisplayStatus.Message == msg {
		return nil
	}

	m.lastMessage = msg

	return SetStatusMessage(ctx, msg)
}

func (m *Monitor) clearMessage(ctx context.Context) error {
	if m.lastMessage == m.printerObjects.DisplayStatus.Message {
		return m.updateStatusMessage(ctx, "")
	}

	return nil
}

func (m *Monitor) getLatestJob(ctx context.Context) (*Job, error) {
	resp, err := GetLatestJob(ctx)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("api response %d %s", resp.Error.Code, resp.Error.Message)
	}

	if len(resp.Result.Jobs) == 0 {
		return nil, nil
	}

	return &(resp.Result.Jobs[0]), nil
}

func (m *Monitor) getLoadedFile(ctx context.Context) (*GCodeMetadata, error) {
	if !m.hasLoadedFile {
		return nil, nil
	}

	metaResponse, err := GetGcodeMetadata(ctx, m.printerObjects.PrintStats.FileName)
	if err != nil {
		return nil, err
	}

	if metaResponse.Error != nil {
		return nil, fmt.Errorf("api response %d %s", metaResponse.Error.Code, metaResponse.Error.Message)
	}

	return metaResponse.Result, nil
}

func (m *Monitor) LatestThumbnail(ctx context.Context, w io.Writer) (string, error) {
	if m.latestJob == nil || m.latestJob.Metadata == nil || len(m.latestJob.Metadata.Thumbnails) == 0 {
		return "", printer.ErrNoThumbnail
	}

	thumb := m.latestJob.Metadata.Thumbnails[len(m.latestJob.Metadata.Thumbnails)-1]

	u := m.printerUrl.JoinPath("/server/files/gcodes").JoinPath(thumb.RelativePath)

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		return "", err
	}

	return resp.Header.Get("Content-Type"), nil
}
