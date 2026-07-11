package web

import (
	"3dp-controller/internal/printer"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) registerAPIRoutes(r *gin.RouterGroup) {
	r.GET("/ping", s.PingHandler)

	r.GET("/printers", s.PrintersHandler)
	r.GET("/printers/:key", s.PrinterHandler)
	r.PUT("/printers/:key", s.UpdatePrinter)
	r.GET("/printers/:key/latest_thumb", s.GetLatestThumbnail)
}

//	@BasePath	/api/v1

// PingHandler godoc
//
//	@Summary	Ping/Pong
//	@Produce	json
//	@Success	200	{string}	Pong
//	@Router		/ping [get]
func (s *Server) PingHandler(g *gin.Context) {
	g.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// PrintersHandler godoc
//
//	@Summary	Get list of printers
//	@Tags		Printers
//	@Produce	json
//	@Success	200	{array}	Printer
//	@Router		/printers [get]
func (s *Server) PrintersHandler(g *gin.Context) {
	printers := make([]Printer, 0)

	for k, m := range s.monitors {
		printers = append(printers, makePrinter(k, m))
	}

	g.JSON(200, printers)
}

// PrinterHandler godoc
//
//	@Summary	Get the printers
//	@Tags		Printers
//	@Param		key	path	string	true	"key of printer"
//	@Produce	json
//	@Success	200	{object}	Printer
//	@Router		/printers/{key} [get]
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

func makePrinter(key string, p printer.Printer) Printer {
	return Printer{
		Key:  key,
		Name: p.PrinterName(),
		Url:  p.PrinterUrl(),
		Type: p.PrinterType(),

		RegJobId:        p.RegisteredJobId(),
		AllowNoRegPrint: p.AllowNoRegPrint(),
		NoPauseDuration: p.Config().NoPauseDuration.Seconds(),

		State:          p.State(),
		Message:        p.Message(),
		ErrorDetail:    p.ErrorDetail(),
		LastUpdateTime: p.LastUpdateTime().UnixMilli(),

		Job: p.Job(),
	}
}

// UpdatePrinter godoc
//
//	@Summary	Update a printer
//	@Tags		Printers
//	@Param		key				path	string	true	"key of printer"
//	@Param		regJobId		query	string	false	"jobId of registered job"
//	@Param		allowNoRegPrint	query	boolean	false	"allow printing without registration"
//	@Produce	json
//	@Success	200	{object}	UpdatePrinterResponse
//	@Failure	404	{object}	APIErrorResp
//	@Router		/printers/{key} [put]
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
//
//	@Summary	Get thumbnail for a file
//	@Tags		Printers
//	@Param		key	path	string	true	"key of printer"
//	@Produce	image/png
//	@Success	200
//	@Router		/printers/{key}/latest_thumb [get]
func (s *Server) GetLatestThumbnail(g *gin.Context) {
	printerKey := g.Param("key")

	p, ok := s.monitors[printerKey]
	if !ok {
		resp := APIErrorResp{
			Error: "printer not found",
		}
		g.JSON(http.StatusNotFound, resp)
		return
	}

	t, ok := p.(printer.Thumbnailer)
	if !ok {
		resp := APIErrorResp{
			Error: "thumbnails not supported by this printer",
		}
		g.JSON(http.StatusNotImplemented, resp)
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	var buf bytes.Buffer
	contentType, err := t.LatestThumbnail(ctx, &buf)
	if errors.Is(err, printer.ErrNoThumbnail) {
		resp := APIErrorResp{
			Error: "no thumbnail available",
		}
		g.JSON(http.StatusNotFound, resp)
		return
	} else if err != nil {
		s.logger.Errorf("get latest thumbnail error: %s", err.Error())
		g.Status(http.StatusInternalServerError)
		return
	}

	g.Status(http.StatusOK)
	g.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))
	g.Header("Content-Type", contentType)

	if _, err := io.Copy(g.Writer, &buf); err != nil {
		s.logger.Errorf("copy error: %s", err.Error())
	}
}
