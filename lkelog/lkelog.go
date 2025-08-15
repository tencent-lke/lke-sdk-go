// Package event contains event definitions.
package lkelog

type RunLogger interface {
	// Info logs information with a specified message.
	Info(message string)

	// Error logs error information with a specified message.
	Error(message string)
}

var Logger RunLogger

func SetRunLogger(l RunLogger) {
	Logger = l
}

// func SetRunLogger(l RunLogger) {
// 	logger = l
// }
