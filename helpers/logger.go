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

// PrintBanner prints a nice ASCII banner for ReviewForge
func PrintBanner() {
	println(`
  _____            _             ______                      
 |  __ \          (_)           |  ____|                     
 | |__) |_____   ___  _____      | |__ ___  _ __ __ _  ___ 
 |  _  // _ \ \ / / |/ _ \ \ /\ / /  __/ _ \| '__/ _' |/ _ \
 | | \ \  __/\ V /| |  __/\ V  V /| | | (_) | | | (_| |  __/
 |_|  \_\___| \_/ |_|\___| \_/\_/ |_|  \___/|_|  \__, |\___|
                                                  __/ |     
                                                 |___/      
	`)
}
