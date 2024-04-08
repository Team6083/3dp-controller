package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"text/template"
	"time"
)

type ConfigServer struct {
	Bind string `yaml:"bind"`
	Port int    `yaml:"port"`
}

type RawConfigDisplayMessages struct {
	WillPauseMessage string `yaml:"will_pause_message"`
	PauseMessage     string `yaml:"pause_message"`
}

type RawConfig struct {
	Server          ConfigServer             `yaml:"server"`
	NoPauseDuration string                   `yaml:"no_pause_duration"`
	DisplayMessages RawConfigDisplayMessages `yaml:"display_messages"`
	Printers        []struct {
		Key  string `yaml:"key"`
		Name string `yaml:"name"`
		Url  string `yaml:"url"`
		// Should be allow_print or no_print, default allow_print
		ControllerFailMode string `yaml:"controller_fail_mode"`
	} `yaml:"printers"`
}

type ControllerFailMode string

func (m ControllerFailMode) String() string {
	return string(m)
}

const (
	FailModeAllowPrint ControllerFailMode = "allow_print"
	FailModeNoPrint    ControllerFailMode = "no_print"
)

type ConfigPrinter struct {
	Key                string
	Name               string
	Url                string
	ControllerFailMode ControllerFailMode
}

type ConfigDisplayMessages struct {
	WillPauseMessage *template.Template
	PauseMessage     *template.Template
}

type Config struct {
	Server          ConfigServer
	NoPauseDuration time.Duration
	DisplayMessages ConfigDisplayMessages
	Printers        map[string]ConfigPrinter
}

func LoadConfig(fileName string) (*Config, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)

	var rawCfg RawConfig
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&rawCfg)
	if err != nil {
		return nil, err
	}

	return ParseRawConfig(rawCfg)
}

func ParseRawConfig(raw RawConfig) (*Config, error) {
	cfg := Config{
		Printers: make(map[string]ConfigPrinter),
	}
	cfg.Server = raw.Server

	willPauseMsg, err := template.New("will_pause").Parse(raw.DisplayMessages.WillPauseMessage)
	if err != nil {
		return nil, err
	}

	pauseMsg, err := template.New("pause").Parse(raw.DisplayMessages.PauseMessage)
	if err != nil {
		return nil, err
	}

	cfg.DisplayMessages = ConfigDisplayMessages{
		WillPauseMessage: willPauseMsg,
		PauseMessage:     pauseMsg,
	}

	noPauseDuration, err := time.ParseDuration(raw.NoPauseDuration)
	if err != nil {
		return nil, err
	}
	cfg.NoPauseDuration = noPauseDuration

	for _, rp := range raw.Printers {
		p := ConfigPrinter{
			Key:  rp.Key,
			Name: rp.Name,
			Url:  rp.Url,
		}

		switch rp.ControllerFailMode {
		case FailModeNoPrint.String():
			p.ControllerFailMode = FailModeNoPrint
		case FailModeAllowPrint.String(), "":
			p.ControllerFailMode = FailModeAllowPrint
		default:
			return nil, errors.New(fmt.Sprintf("Unknown ControllerFailMode %s", rp.ControllerFailMode))
		}

		if _, ok := cfg.Printers[p.Key]; ok {
			return nil, fmt.Errorf("duplicated printer '%s'", p.Key)
		}
		cfg.Printers[rp.Key] = p
	}

	return &cfg, nil
}
