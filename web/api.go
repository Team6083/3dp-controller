package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"v400_monitor/moonraker"
)

func (s *Server) registerAPIRoutes(r *gin.RouterGroup) {
	r.GET("/ping", s.PingHandler)

	r.GET("/printers", s.PrintersHandler)
	r.GET("/printers/:key", s.PrinterHandler)
	r.PUT("/printers/:key", s.UpdatePrinter)
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
	return Printer{
		Key:  key,
		Name: m.PrinterName(),
		Url:  m.PrinterUrl(),

		RegJobId:        m.RegisteredJobId(),
		AllowNoRegPrint: m.AllowNoRegPrint(),

		State:          m.State(),
		PrinterObjects: m.PrinterObjects(),
		LoadedFile:     m.LoadedFile(),
		LatestJob:      m.LatestJob(),
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
