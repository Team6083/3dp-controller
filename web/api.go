package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s *Server) registerAPIRoutes(r *gin.RouterGroup) {
	r.GET("/ping", s.PingHandler)

	r.GET("/printers", s.PrintersHandler)
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
		printers = append(printers, Printer{
			Key:  k,
			Name: m.PrinterName(),
			Url:  m.PrinterUrl(),

			RegJobId:        m.RegisteredJobId(),
			AllowNoRegPrint: m.AllowNoRegPrint(),

			State:          m.State(),
			PrinterObjects: m.PrinterObjects(),
			LoadedFile:     m.LoadedFile(),
			LatestJob:      m.LatestJob(),
		})
	}

	g.JSON(200, printers)
}
