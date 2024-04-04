package moonraker

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"
)

type PrinterStatus string

const (
	Unknown    PrinterStatus = "unknown"
	Idle       PrinterStatus = "idle"
	PrePrint   PrinterStatus = "pre_print"
	Printing   PrinterStatus = "printing"
	ForcePause PrinterStatus = "force_pause"
	Pause      PrinterStatus = "pause"
	Error      PrinterStatus = "error"
	Timeout    PrinterStatus = "timeout"
	MonError   PrinterStatus = "mon_error"
)

type Monitor struct {
	PrinterName string
	PrinterUrl  *url.URL

	NoPauseDuration time.Duration

	AllowPrint bool

	LastUpdateTime time.Time

	Status        PrinterStatus
	DisplayStatus *PrinterObjectDisplayStatus
	IdleTimeout   *PrinterObjectIdleTimeout
	PrintStats    *PrinterObjectPrintStats
	VirtualSDCard *PrinterObjectVirtualSDCard

	ticker      *time.Ticker
	stopPolling chan bool
}

func NewMonitor(name string, printerURL string) (*Monitor, error) {
	m := new(Monitor)

	u, err := url.Parse(printerURL)
	if err != nil {
		return nil, err
	}

	m.PrinterName = name
	m.PrinterUrl = u

	m.AllowPrint = false
	m.Status = Unknown

	return m, nil
}

func (m *Monitor) Start() {
	if m.ticker != nil {
		return
	}

	m.ticker = time.NewTicker(5 * time.Second)
	m.stopPolling = make(chan bool)

	go func() {
		m.update()

		for {
			select {
			case <-m.stopPolling:
				m.ticker.Stop()
				m.ticker = nil
				return
			case <-m.ticker.C:
				m.update()
			}
		}
	}()
}

func (m *Monitor) Stop() {
	m.stopPolling <- true
}

func (m *Monitor) update() {
	status, err := GetPrinterObjects(m.PrinterUrl)

	m.LastUpdateTime = time.Now()

	if err != nil {
		m.DisplayStatus = nil
		m.IdleTimeout = nil
		m.PrintStats = nil
		m.VirtualSDCard = nil

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			// A timeout error occurred
			m.Status = Timeout
			fmt.Printf("Error: Printer %s Timeout!\n", m.PrinterName)
		} else {
			fmt.Println(err)
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
			fmt.Printf("Printer %s should not print now!!\n", m.PrinterName)

			if printDuration > m.NoPauseDuration {
				m.Status = ForcePause
			}

			if m.VirtualSDCard.Progress > 0.5 {
				// Stop job
			}
		}

		// Use m.PrintStats.State to check real printer status
		if realPrinterStatus == Printing && m.Status == ForcePause {
			fmt.Printf("Pausing printer %s\n", m.PrinterName)

			err := PausePrint(m.PrinterUrl)
			if err != nil {
				fmt.Printf("Error pausing the printer: %s\n", err)
			}
		}

		if m.Status == Printing && !printerShouldPrint {
			remDuration := (m.NoPauseDuration - printDuration).Round(time.Second)

			err := m.updateStatusMessage(fmt.Sprintf("Will pause after %s", remDuration.String())) // 請進行使用登記，否則將於%s後暫停工作
			if err != nil {
				fmt.Println(err)
			}
		} else if m.Status == ForcePause {
			err := m.updateStatusMessage("No reg, force pause") // 無使用登記，已暫停列印工作
			if err != nil {
				fmt.Println(err)
			}
		}

		// If currently force paused and printer is paused, then resume printer
		if m.Status == ForcePause && m.AllowPrint && realPrinterStatus == Pause {
			m.Status = Pause

			fmt.Printf("Resuming printer %s\n", m.PrinterName)

			err := ResumePrint(m.PrinterUrl)
			if err != nil {
				fmt.Printf("Error resuming the printer: %s\n", err)
			}

			err = m.updateStatusMessage("")
			if err != nil {
				fmt.Println(err)
			}
		}

		fmt.Printf("Printer Status: %s\n", m.Status)
		fmt.Printf("%+v\n", status.Result.Status)
	}
}

func (m *Monitor) updateStatusMessage(msg string) error {
	if m.DisplayStatus.Message == msg {
		return nil
	}

	return SetStatusMessage(m.PrinterUrl, msg)
}
