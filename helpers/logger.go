package helpers

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SetupLogger configures the global logger with the specified level
func SetupLogger(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Parse log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	// Set global log level
	zerolog.SetGlobalLevel(logLevel)

	// Configure console writer with colors
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "15:04:05",
	}

	// Set global logger
	log.Logger = log.Output(consoleWriter)
}
