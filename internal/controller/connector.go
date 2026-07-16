package controller

import (
	"3dp-controller/internal/controller/api"
	"3dp-controller/internal/printer"
	"3dp-controller/internal/util"
	"context"
	"errors"
	"net/url"
	"time"

	"go.uber.org/zap"
)

type Connector struct {
	controllerUrl *url.URL
	hubId         string
	logger        *zap.SugaredLogger

	monitors        map[string]printer.Printer
	controlSettings map[string]api.ControlSetting

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewConnector(controllerUrl *url.URL, hubId string, logger *zap.SugaredLogger, monitors map[string]printer.Printer) *Connector {
	return &Connector{
		controllerUrl:   controllerUrl,
		hubId:           hubId,
		logger:          logger,
		monitors:        monitors,
		controlSettings: make(map[string]api.ControlSetting),
	}
}

func (c *Connector) Connect(ctx context.Context) {
	ctx = context.WithValue(ctx, "controllerAPIUrl", c.controllerUrl)
	ctx = context.WithValue(ctx, "hubId", c.hubId)

	ctx, cancel := context.WithCancel(ctx)
	c.ctx = ctx
	c.cancelFunc = cancel

	ticker1Duration := 2 * time.Second
	ticker1 := time.NewTicker(ticker1Duration)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker1.Stop()
				return
			case <-ticker1.C:
				c.update()
			}
		}
	}()
}

func (c *Connector) Close() {
	if c.ctx != nil {
		c.cancelFunc()

		c.ctx = nil
		c.cancelFunc = nil
	}
}

func (c *Connector) update() {
	var updates []api.UpdateMessage

	for key, monitor := range c.monitors {
		job := monitor.Job()

		var jobReport api.JobReport
		if job != nil {
			// build jobStatus
			var jobStatus api.ReportJobStatus
			switch job.Status {
			case "in_progress":
				jobStatus = api.ReportInProgress
			case "completed":
				jobStatus = api.ReportDone
			default:
				jobStatus = api.ReportQuit
			}

			// build JobReport
			jobReport = api.JobReport{
				Id:     job.JobId,
				Status: jobStatus,

				ContentId: job.ContentId,
			}

			if job.StartTime != nil {
				jobReport.StartTime = *job.StartTime
			}
		}

		// build status
		var status api.Status
		switch monitor.State() {
		case printer.Ready:
			status = api.StatusIdle
		case printer.PrePrint, printer.Printing:
			status = api.StatusRunning
		case printer.Pause:
			status = api.StatusPaused
		case printer.Error, printer.InternalError:
			status = api.StatusError
		case printer.Disconnected:
			status = api.StatusDisconnected
		default:
			status = api.StatusUnknown
		}

		report := api.Report{
			Status:                status,
			JobReport:             jobReport,
			CurrentControlSetting: c.controlSettings[key],
		}

		msg := api.UpdateMessage{
			Key:   key,
			State: report,
		}

		updates = append(updates, msg)
	}

	ctrlMessages, err := api.UpdateHubStatus(c.ctx, updates)
	if err != nil {
		if util.IsErrNetworkProblem(err) {
			c.logger.Warnln("can't connect to controller")
			return
		}

		c.logger.Errorf("update hub status err: %s\n", err)

		var errRespNotOk api.ERRRespNotOk
		if ok := errors.As(err, &errRespNotOk); ok {
			c.logger.Error(errRespNotOk)
		}
		return
	}

	for _, msg := range ctrlMessages {
		monitor, ok := c.monitors[msg.Key]
		if !ok {
			c.logger.Warnf("monitor not found for %s\n", msg.Key)
			return
		}

		c.controlSettings[msg.Key] = msg.ControlSetting

		regJobId := ""
		allowNoRegPrint := false

		if msg.ControlSetting.IsActive {
			regJobId = msg.ActiveJobId
			if regJobId == "" {
				allowNoRegPrint = true
			}
		}

		monitor.SetRegisteredJobId(regJobId)
		monitor.SetAllowNoRegPrint(allowNoRegPrint)

		// TODO: implement close, and maintenance
	}
}
