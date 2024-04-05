package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
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
		fmt.Printf("%+v\n", p)

		m, err := moonraker.NewMonitor(p.Name, p.Url)
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
					println("Error: Printer no. %d not found!\n", num)
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
