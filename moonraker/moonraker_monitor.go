package moonraker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net"
	"net/url"
	"syscall"
	"text/template"
	"time"
)

type MonitorConfig struct {
	NoPauseDuration  time.Duration
	WillPauseMessage *template.Template
	PauseMessage     *template.Template
}

type PrinterState string

const (
	KlippyStartup      PrinterState = "klippy_startup"
	KlippyShutdown     PrinterState = "klippy_shutdown"
	KlippyError        PrinterState = "klippy_error"
	KlippyDisconnected PrinterState = "klippy_disconnected"

	Ready    PrinterState = "ready"
	PrePrint PrinterState = "pre_print"
	Printing PrinterState = "printing"
	Pause    PrinterState = "pause"
	Error    PrinterState = "error"

	Disconnected  PrinterState = "disconnected"
	Unknown       PrinterState = "unknown"
	InternalError PrinterState = "internal_error" // Internal error
)

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
	config      MonitorConfig

	registeredJobId    string
	allowNoRegPrint    bool
	jobPausedByMonitor bool
	lastMessage        string

	state          PrinterState
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

func (m *Monitor) Config() MonitorConfig {
	return m.config
}

func (m *Monitor) State() PrinterState {
	return m.state
}

func (m *Monitor) LastUpdateTime() time.Time {
	return m.lastUpdateTime
}

func (m *Monitor) PrinterObjects() *MonitorPrinterObjects {
	return m.printerObjects
}

func (m *Monitor) LatestJob() *Job {
	return m.latestJob
}

func (m *Monitor) LoadedFile() *GCodeMetadata {
	return m.loadedFile
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

func NewMonitor(name string, printerURL string, config MonitorConfig, logger *zap.SugaredLogger) (*Monitor, error) {
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

	m.state = Disconnected
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
					if m.state == Disconnected || m.state == InternalError {
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
					if m.state == Disconnected || m.state == InternalError {
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

		var netErr net.Error
		if (errors.As(err, &netErr) && netErr.Timeout()) ||
			errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) ||
			errors.Is(err, syscall.EHOSTDOWN) || errors.Is(err, syscall.ENETDOWN) ||
			errors.Is(err, syscall.EHOSTUNREACH) || errors.Is(err, syscall.ENETUNREACH) {
			m.state = Disconnected
		} else {
			m.state = InternalError
			m.logger.Errorf("Error getting printer objects: %s\n", err)
		}
	} else {
		if printerObjectsResponse.Result.Status == nil {
			m.state = Error
			m.hasLoadedFile = false

			m.logger.Errorf("MoonrakerError: %d %s\n",
				printerObjectsResponse.Error.Code, printerObjectsResponse.Error.Message)
		} else {
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
					m.state = KlippyStartup
				case "shutdown":
					m.state = KlippyShutdown
				case "error":
					m.state = KlippyError
				case "disconnected":
					m.state = KlippyDisconnected
				default:
					m.state = Unknown
				}
			} else {
				printerShouldPrint := m.allowNoRegPrint || m.registeredJobId != ""
				printDuration := printerObjects.PrintStats.GetPrintDuration()

				switch printerObjects.PrintStats.State {
				case "standby", "complete", "cancelled":
					m.state = Ready
				case "printing":
					if printDuration > 0 {
						m.state = Printing
					} else {
						m.state = PrePrint
					}
				case "paused":
					m.state = Pause
				case "error":
					m.state = Error
				default:
					m.state = Unknown
				}

				m.hasLoadedFile = printerObjects.PrintStats.State != "standby" &&
					m.state != Error && m.state != Unknown

				// Check if printer is illegally printing
				if m.state == Printing && !printerShouldPrint {
					m.logger.Infoln("Printer should not print now!!")

					if printDuration > m.config.NoPauseDuration {
						m.jobPausedByMonitor = true
					}

					if printerObjects.VirtualSDCard.Progress > 0.5 {
						// Stop job
					}
				}

				// Pause printer if printer should be paused by monitor
				if m.state == Printing && m.jobPausedByMonitor {
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
						err := m.updateStatusMessage(m.ctx, tpl.String()) // 無使用登記，已暫停列印工作
						if err != nil {
							m.logger.Errorln(err)
						}
					}
				}

				// Show warning countdown if printer will be paused
				if m.state == Printing && !m.jobPausedByMonitor && !printerShouldPrint {
					remDuration := (m.config.NoPauseDuration - printDuration).Round(time.Second)

					data := struct {
						RemainDurationStr string
					}{remDuration.String()}

					var tpl bytes.Buffer
					if err := m.config.WillPauseMessage.Execute(&tpl, data); err != nil {
						m.logger.Errorf("Error pausing the will pause message: %s\n", err)
					} else {
						err := m.updateStatusMessage(m.ctx, tpl.String()) // 請進行使用登記，否則將於%s後暫停工作
						if err != nil {
							m.logger.Errorln(err)
						}
					}
				} else {
					// TODO: clear will pause message
				}

				// Resume print if allow print set to true
				if m.jobPausedByMonitor && printerShouldPrint {

					if m.state == Pause {
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
