package moonraker

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net"
	"net/url"
	"syscall"
	"time"
)

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

type MonitorConfig struct {
	NoPauseDuration time.Duration
}

type Monitor struct {
	printerName string
	printerUrl  *url.URL
	logger      *zap.SugaredLogger
	config      MonitorConfig

	AllowPrint         bool
	jobPausedByMonitor bool

	state          PrinterState
	lastUpdateTime time.Time
	printerObjects *MonitorPrinterObjects
	hasLoadedFile  bool

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (m *Monitor) PrinterName() string {
	return m.printerName
}

func (m *Monitor) PrinterUrl() url.URL {
	return *m.printerUrl
}

func (m *Monitor) State() PrinterState {
	return m.state
}

func (m *Monitor) LastUpdateTime() time.Time {
	return m.lastUpdateTime
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

	m.AllowPrint = true
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

	ticker := time.NewTicker(2 * time.Second)

	go func() {
		m.update()

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				m.update()
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

func (m *Monitor) GetLoadedFile() (*GCodeMetadata, error) {
	if !m.hasLoadedFile {
		return nil, nil
	}

	metaResponse, err := GetGcodeMetadata(m.ctx, m.printerObjects.PrintStats.FileName)
	if err != nil {
		return nil, err
	}

	if metaResponse.Error != nil {
		return nil, fmt.Errorf("api response %d %s", metaResponse.Error.Code, metaResponse.Error.Message)
	}

	return metaResponse.Result, nil
}

func (m *Monitor) GetLatestJob() (*Job, error) {
	resp, err := GetLatestJob(m.ctx)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("api response %d %s", resp.Error.Code, resp.Error.Message)
	}

	return &(resp.Result.Jobs[0]), nil
}

func (m *Monitor) update() {
	printerObjectsResponse, err := GetPrinterObjects(m.ctx)

	m.lastUpdateTime = time.Now()

	if err != nil {
		m.printerObjects = nil

		var netErr net.Error
		if (errors.As(err, &netErr) && netErr.Timeout()) ||
			errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
			m.state = Disconnected
		} else {
			m.state = InternalError
			m.logger.Errorf("Error getting orinter objects: %s\n", err)
		}
	} else {
		if printerObjectsResponse.Result.Status == nil {
			m.state = Error
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
				printerShouldPrint := m.AllowPrint
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

					// TODO: Use template
					err = m.updateStatusMessage("No reg, force pause") // 無使用登記，已暫停列印工作
					if err != nil {
						m.logger.Errorln(err)
					}
				}

				// Show warning countdown if printer will be paused
				if m.state == Printing && !m.jobPausedByMonitor && !printerShouldPrint {
					remDuration := (m.config.NoPauseDuration - printDuration).Round(time.Second)

					// TODO: Use template
					err := m.updateStatusMessage(
						fmt.Sprintf("Will pause after %s", remDuration.String()),
					) // 請進行使用登記，否則將於%s後暫停工作
					if err != nil {
						m.logger.Errorln(err)
					}
				}

				// Resume print if allow print set to true
				if m.jobPausedByMonitor && m.AllowPrint {

					if m.state == Pause {
						m.logger.Infoln("Resuming")

						err := ResumePrint(m.ctx)
						if err != nil {
							m.logger.Errorf("Error resuming the printer: %s\n", err)
						}

						err = m.updateStatusMessage("")
						if err != nil {
							m.logger.Errorln(err)
						}
					}

					m.jobPausedByMonitor = false
				}
			}
		}
	}
	m.logger.Debugf("Status: %s\n", m.state)
}

func (m *Monitor) updateStatusMessage(msg string) error {
	if m.printerObjects.DisplayStatus.Message == msg {
		return nil
	}

	return SetStatusMessage(m.ctx, msg)
}
