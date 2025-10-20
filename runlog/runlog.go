package runlog

type RunLogger interface {
	// Info logs information with a specified message.
	Info(message string)

	// Error logs error information with a specified message.
	Error(message string)
}
