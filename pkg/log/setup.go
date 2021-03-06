package log

import (
	"io"
	"io/ioutil"
	stdlog "log"
	"os"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = time.RFC3339Nano
	stdlog.SetFlags(stdlog.Lshortfile)
	setup()
}

func setup() {
	stdlog.SetOutput(
		LoggerWithoutCaller.
			With().
			Str("module", "stdlog").
			Str("level", "info").
			Logger())
	zlog.Logger = Logger
}

// SetDiscardLogger sets global log level to trace and log to a ioutil.Discard writer.
// Use in tests to ensure that writing logs don't panic and are sileced.
func SetDiscardLogger() {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	Logger = zerolog.New(ioutil.Discard)
	LoggerWithoutCaller = zerolog.New(ioutil.Discard)
	setup()
}

// SetBlockingLogger sets up a logger with a non blocking writer
func SetBlockingLogger(w io.Writer) {
	LoggerWithoutCaller = zerolog.New(w).With().Timestamp().Logger()
	Logger = LoggerWithoutCaller.With().Caller().Logger()
	setup()
}

// SetConsoleLogger sets up logging for console logging (developmnet)
func SetConsoleLogger() {
	wr := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "15:04:05.000",
	}
	LoggerWithoutCaller = zerolog.New(wr).With().Timestamp().Logger()
	Logger = LoggerWithoutCaller.With().Caller().Logger()
	setup()
}

func nilFormatter(interface{}) string {
	return ""
}

// SetVerboseTestingConsoleLogger
func SetVerboseTestingConsoleLogger(level zerolog.Level) {
	wr := zerolog.ConsoleWriter{
		NoColor:         true,
		Out:             os.Stdout,
		TimeFormat:      "15:04:05",
		FormatTimestamp: nilFormatter,
		FormatLevel:     nilFormatter,
		// FormatCaller:    nilFormatter,
	}
	LoggerWithoutCaller = zerolog.New(wr).With().Timestamp().Logger()
	Logger = LoggerWithoutCaller.With().Caller().Logger()
	zerolog.SetGlobalLevel(level)
	setup()
}

// SetSimpleConsoleLogger sets up logging for console logging to cli tool end users
func SetSimpleConsoleLogger() {
	wr := zerolog.ConsoleWriter{
		Out:             os.Stdout,
		TimeFormat:      "15:04:05",
		FormatTimestamp: nilFormatter,
		FormatLevel:     nilFormatter,
		FormatCaller:    nilFormatter,
	}
	LoggerWithoutCaller = zerolog.New(wr).With().Timestamp().Logger()
	Logger = LoggerWithoutCaller.With().Caller().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	setup()
}
