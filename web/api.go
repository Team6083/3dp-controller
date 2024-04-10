package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"v400_monitor/moonraker"
)

func (s *Server) registerAPIRoutes(r *gin.RouterGroup) {
	r.GET("/ping", s.PingHandler)

	r.GET("/printers", s.PrintersHandler)
	r.GET("/printers/:key", s.PrinterHandler)
	r.PUT("/printers/:key", s.UpdatePrinter)
	r.GET("/printers/:key/latest_thumb", s.GetLatestThumbnail)
}

// @BasePath /api/v1

// PingHandler godoc
// @Summary Ping/Pong
// @Produce json
// @Success 200 {string} Pong
// @Router /ping [get]
func (s *Server) PingHandler(g *gin.Context) {
	g.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// PrintersHandler godoc
// @Summary Get list of printers
// @Tags Printers
// @Produce json
// @Success 200 {array} Printer
// @Router /printers [get]
func (s *Server) PrintersHandler(g *gin.Context) {
	printers := make([]Printer, 0)

	for k, m := range s.monitors {
		printers = append(printers, makePrinter(k, m))
	}

	g.JSON(200, printers)
}

// PrinterHandler godoc
// @Summary Get the printers
// @Tags Printers
// @Param key path string true "key of printer"
// @Produce json
// @Success 200 {object} Printer
// @Router /printers/{key} [get]
func (s *Server) PrinterHandler(g *gin.Context) {
	key := g.Param("key")

	if m, ok := s.monitors[key]; ok {
		g.JSON(200, makePrinter(key, m))
	} else {
		resp := APIErrorResp{
			Error: "printer not found",
		}

		g.JSON(http.StatusNotFound, resp)
	}
}

type UpdatePrinterResponse struct {
	RegJobId        *string `json:"reg_job_id"`
	AllowNoRegPrint *bool   `json:"allow_no_reg_print"`
}

func makePrinter(key string, m *moonraker.Monitor) Printer {
	printerObjs := m.PrinterObjects()

	var message string
	var printerStats *moonraker.PrinterObjectPrintStats
	var displayStatus *moonraker.PrinterObjectDisplayStatus
	var virtualSDCard *moonraker.PrinterObjectVirtualSDCard

	if printerObjs != nil {
		if printerObjs.Webhooks.State != "ready" {
			message = printerObjs.Webhooks.StateMessage
		} else {
			message = printerObjs.PrintStats.Message
		}

		printerStats = &printerObjs.PrintStats
		displayStatus = &printerObjs.DisplayStatus
		virtualSDCard = &printerObjs.VirtualSDCard
	}

	return Printer{
		Key:  key,
		Name: m.PrinterName(),
		Url:  m.PrinterUrl(),

		RegJobId:        m.RegisteredJobId(),
		AllowNoRegPrint: m.AllowNoRegPrint(),
		NoPauseDuration: m.Config().NoPauseDuration.Seconds(),

		State:          m.State(),
		Message:        message,
		LastUpdateTime: m.LastUpdateTime().UnixMilli(),

		PrinterStats:  printerStats,
		DisplayStatus: displayStatus,
		VirtualSDCard: virtualSDCard,

		LoadedFile: m.LoadedFile(),
		LatestJob:  m.LatestJob(),
	}
}

// UpdatePrinter godoc
// @Summary Update a printer
// @Tags Printers
// @Param key 				path 	string 		true 	"key of printer"
// @Param regJobId 			query 	string 		false 	"jobId of registered job"
// @Param allowNoRegPrint 	query 	boolean 	false	"allow printing without registration"
// @Produce json
// @Success 200 {object} UpdatePrinterResponse
// @Failure 404 {object} APIErrorResp
// @Router /printers/{key} [put]
func (s *Server) UpdatePrinter(g *gin.Context) {
	printerKey := g.Param("key")

	regJobId, shouldUpdateRegJobId := g.GetQuery("regJobId")

	var allowNoRegPrint bool
	shouldUpdateAllowNoRegPrint := false

	allowNoRegPrintRaw := g.Query("allowNoRegPrint")
	if allowNoRegPrintRaw != "" {
		b, err := strconv.ParseBool(allowNoRegPrintRaw)
		if err != nil {
			fmt.Println(err)
		} else {
			shouldUpdateAllowNoRegPrint = true
			allowNoRegPrint = b
		}
	}

	if m, ok := s.monitors[printerKey]; ok {
		resp := UpdatePrinterResponse{}

		if shouldUpdateRegJobId {
			m.SetRegisteredJobId(regJobId)
			resp.RegJobId = &regJobId
		}

		if shouldUpdateAllowNoRegPrint {
			m.SetAllowNoRegPrint(allowNoRegPrint)
			resp.AllowNoRegPrint = &allowNoRegPrint
		}

		g.JSON(http.StatusOK, resp)
	} else {
		resp := APIErrorResp{
			Error: "printer not found",
		}

		g.JSON(http.StatusNotFound, resp)
	}
}

// GetLatestThumbnail godoc
// @Summary Get thumbnail for a file
// @Tags Printers
// @Param key				path	string	true	"key of printer"
// @Produce image/png
// @Success 200
// @Router /printers/{key}/latest_thumb [get]
func (s *Server) GetLatestThumbnail(g *gin.Context) {
	printerKey := g.Param("key")

	if m, ok := s.monitors[printerKey]; ok {
		latestJob := m.LatestJob()
		if latestJob == nil {
			resp := APIErrorResp{
				Error: "no latest job",
			}
			g.JSON(http.StatusNotFound, resp)
			return
		}

		if len(latestJob.Metadata.Thumbnails) == 0 {
			resp := APIErrorResp{
				Error: "no thumbnails",
			}
			g.JSON(http.StatusNotFound, resp)
			return
		}

		thumb := latestJob.Metadata.Thumbnails[len(latestJob.Metadata.Thumbnails)-1]

		u, err := url.Parse(m.PrinterUrl())
		if err != nil {
			s.logger.Errorf("url parse error: %s", err.Error())
			g.Status(http.StatusInternalServerError)
			return
		}

		u = u.JoinPath("/server/files/gcodes").JoinPath(thumb.RelativePath)

		req, err := http.NewRequestWithContext(s.ctx, "GET", u.String(), nil)
		if err != nil {
			s.logger.Errorf("create request error: %s", err.Error())
			g.Status(http.StatusInternalServerError)
			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			s.logger.Errorf("request error: %s", err.Error())
			g.Status(http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()

		g.Status(http.StatusOK)
		g.Header("Context-Length", fmt.Sprintf("%d", resp.ContentLength))
		g.Header("Content-Type", resp.Header.Get("Content-Type"))

		if _, err = io.Copy(g.Writer, resp.Body); err != nil {
			// handle error
		}
	} else {
		resp := APIErrorResp{
			Error: "printer not found",
		}
		g.JSON(http.StatusNotFound, resp)
	}
}
