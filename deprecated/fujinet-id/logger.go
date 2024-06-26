package main

import (
	"io"
	"log"
	"os"
)

type CustomLogger struct {
	*log.Logger
	name string
	on   bool
}

func (logger *CustomLogger) GetName() string {
	return logger.name
}

func (logger *CustomLogger) IsOn() bool {
	return logger.on
}

func (logger *CustomLogger) SetActive(newstatus bool) {

	switch newstatus {

	case true:
		logger.SetOutput(os.Stdout)
		logger.on = true
	case false:
		logger.SetOutput(io.Discard)
		logger.on = false

	}
}

func (logger *CustomLogger) String() string {

	if logger.IsOn() {
		return logger.GetName() + " is on"
	}

	return logger.GetName() + " is off"
}

func NewCustomLogger(name string, prefix string, flag int) CustomLogger {

	logger := CustomLogger{
		Logger: log.New(os.Stdout, prefix, log.LstdFlags|flag),
		name:   name,
		on:     true,
	}

	return logger
}
