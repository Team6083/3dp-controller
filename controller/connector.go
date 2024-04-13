package controller

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"net/url"
	"time"
	"v400_monitor/controller/api"
	"v400_monitor/moonraker"
)

type Connector struct {
	controllerUrl *url.URL
	hubId         string
	logger        *zap.SugaredLogger

	monitors        map[string]*moonraker.Monitor
	controlSettings map[string]api.ControlSetting

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewConnector(controllerUrl *url.URL, hubId string, logger *zap.SugaredLogger, monitors map[string]*moonraker.Monitor) *Connector {
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
		latestJob := monitor.LatestJob()

		var jobReport api.JobReport
		if latestJob != nil {
			// build jobStatus
			var jobStatus api.ReportJobStatus
			switch latestJob.Status {
			case "in_progress":
				jobStatus = api.ReportInProgress
			case "completed":
				jobStatus = api.ReportDone
			default:
				jobStatus = api.ReportQuit
			}

			// build JobReport
			jobReport = api.JobReport{
				Id:     latestJob.JobId,
				Status: jobStatus,

				ContentId: latestJob.Metadata.UUID,
				StartTime: time.UnixMilli(int64(latestJob.StartTime * 1000)),
			}
		}

		// build status
		var status api.Status
		switch monitor.State() {
		case moonraker.Ready:
			status = api.StatusIdle
		case moonraker.PrePrint, moonraker.Printing:
			status = api.StatusRunning
		case moonraker.Pause:
			status = api.StatusPaused
		case moonraker.Error, moonraker.InternalError,
			moonraker.KlippyError, moonraker.KlippyShutdown, moonraker.KlippyDisconnected:
			status = api.StatusError
		case moonraker.Disconnected:
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
