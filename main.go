package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"v400_monitor/moonraker"
)

func getTerminalInput(input chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		input <- scanner.Text()
	}
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil && !errors.Is(err, syscall.ENOTTY) {
			panic(err)
		}
	}(logger)

	sugar := logger.Sugar()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	termInput := make(chan string)
	go getTerminalInput(termInput)

	config, err := LoadConfig("./config.yaml")
	if err != nil {
		panic(err)
	}

	var monitors []*moonraker.Monitor

	ctx := context.Background()

	for _, p := range config.Printers {
		m, err := moonraker.NewMonitor(p.Name, p.Url, sugar.With(
			"PrinterName", p.Name,
		))
		if err != nil {
			panic(err)
		}

		if p.ControllerFailMode == FailModeNoPrint {
			m.AllowPrint = false
		}

		m.NoPauseDuration = config.NoPauseDuration
		m.Start(ctx)

		monitors = append(monitors, m)
	}

	for {
		select {
		case inputStr := <-termInput:
			num, err := strconv.ParseInt(inputStr, 10, 32)
			if err != nil {
				fmt.Println(err)
			} else {
				if int(num) >= len(monitors) {
					fmt.Printf("Error: Printer no. %d not found!\n", num)
				} else {
					m := monitors[int(num)]

					if m.AllowPrint {
						fmt.Println("Setting AllowPrint to false")
						m.AllowPrint = false
					} else {
						fmt.Println("Setting AllowPrint to true")
						m.AllowPrint = true
					}
				}
			}
		case s := <-interrupt:
			for _, m := range monitors {
				m.Stop()
			}

			fmt.Println("Got signal:", s)
			return
		}
	}
}
