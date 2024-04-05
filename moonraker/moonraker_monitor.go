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

type PrinterStatus string

const (
	Unknown      PrinterStatus = "unknown"
	Idle         PrinterStatus = "idle"
	PrePrint     PrinterStatus = "pre_print"
	Printing     PrinterStatus = "printing"
	ForcePause   PrinterStatus = "force_pause"
	Pause        PrinterStatus = "pause"
	Error        PrinterStatus = "error"
	Disconnected PrinterStatus = "disconnected"
	MonError     PrinterStatus = "mon_error"
)

type Monitor struct {
	PrinterName string
	PrinterUrl  *url.URL

	logger *zap.SugaredLogger

	NoPauseDuration time.Duration

	AllowPrint bool

	LastUpdateTime time.Time

	Status        PrinterStatus
	DisplayStatus *PrinterObjectDisplayStatus
	IdleTimeout   *PrinterObjectIdleTimeout
	PrintStats    *PrinterObjectPrintStats
	VirtualSDCard *PrinterObjectVirtualSDCard

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewMonitor(name string, printerURL string, logger *zap.SugaredLogger) (*Monitor, error) {
	m := new(Monitor)

	m.logger = logger

	u, err := url.Parse(printerURL)
	if err != nil {
		return nil, err
	}

	m.PrinterName = name
	m.PrinterUrl = u

	m.AllowPrint = true
	m.Status = Unknown

	return m, nil
}

func (m *Monitor) Start(ctx context.Context) {
	if m.ctx != nil {
		return
	}

	ctx = context.WithValue(ctx, "moonrakerAPIUrl", m.PrinterUrl)

	ctx, cancel := context.WithCancel(ctx)
	m.ctx = ctx
	m.cancelFunc = cancel

	ticker := time.NewTicker(5 * time.Second)

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

func (m *Monitor) update() {
	status, err := GetPrinterObjects(m.ctx)

	m.LastUpdateTime = time.Now()

	if err != nil {
		m.DisplayStatus = nil
		m.IdleTimeout = nil
		m.PrintStats = nil
		m.VirtualSDCard = nil

		var netErr net.Error
		if (errors.As(err, &netErr) && netErr.Timeout()) ||
			errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
			m.Status = Disconnected
		} else {
			m.logger.Errorf("Error getting orinter objects: %s\n", err)
			m.Status = MonError
		}
	} else {
		m.DisplayStatus = new(PrinterObjectDisplayStatus)
		*m.DisplayStatus = status.Result.Status.DisplayStatus

		m.IdleTimeout = new(PrinterObjectIdleTimeout)
		*m.IdleTimeout = status.Result.Status.IdleTimeout

		m.PrintStats = new(PrinterObjectPrintStats)
		*m.PrintStats = status.Result.Status.PrintStats

		m.VirtualSDCard = new(PrinterObjectVirtualSDCard)
		*m.VirtualSDCard = status.Result.Status.VirtualSDCard

		printerShouldPrint := m.AllowPrint

		printDuration := time.Duration(m.PrintStats.PrintDuration * float32(time.Second))

		var realPrinterStatus PrinterStatus
		switch m.PrintStats.State {
		case "standby", "complete", "cancelled":
			realPrinterStatus = Idle
		case "printing":
			if printDuration > 0 {
				realPrinterStatus = Printing
			} else {
				realPrinterStatus = PrePrint
			}
		case "paused":
			realPrinterStatus = Pause
		case "error":
			realPrinterStatus = Error
		default:
			realPrinterStatus = Unknown
		}

		// Update Printer Status if printer is idle/error or current status is not force pause
		if realPrinterStatus == Idle || realPrinterStatus == Error || m.Status != ForcePause {
			m.Status = realPrinterStatus
		}

		if m.Status == Printing && !printerShouldPrint {
			m.logger.Infoln("Printer should not print now!!")

			if printDuration > m.NoPauseDuration {
				m.Status = ForcePause
			}

			if m.VirtualSDCard.Progress > 0.5 {
				// Stop job
			}
		}

		// Use m.PrintStats.State to check real printer status
		if realPrinterStatus == Printing && m.Status == ForcePause {
			m.logger.Infoln("Pausing")

			err := PausePrint(m.ctx)
			if err != nil {
				m.logger.Errorf("Error pausing the printer: %s\n", err)
			}
		}

		if m.Status == Printing && !printerShouldPrint {
			remDuration := (m.NoPauseDuration - printDuration).Round(time.Second)

			err := m.updateStatusMessage(fmt.Sprintf("Will pause after %s", remDuration.String())) // 請進行使用登記，否則將於%s後暫停工作
			if err != nil {
				m.logger.Errorln(err)
			}
		} else if m.Status == ForcePause {
			err := m.updateStatusMessage("No reg, force pause") // 無使用登記，已暫停列印工作
			if err != nil {
				m.logger.Errorln(err)
			}
		}

		// If currently force paused and printer is paused, then resume printer
		if m.Status == ForcePause && m.AllowPrint && realPrinterStatus == Pause {
			m.Status = Pause

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

		//m.logger.Debugf("%+v\n", status.Result.Status)
	}
	m.logger.Debugf("Status: %s\n", m.Status)
}

func (m *Monitor) updateStatusMessage(msg string) error {
	if m.DisplayStatus.Message == msg {
		return nil
	}

	return SetStatusMessage(m.ctx, msg)
}
