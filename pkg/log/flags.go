package log

import (
	"flag"
	"os"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config is a collection of stdlib flags configuring logging
type Config struct {
	Debug          bool
	Trace          bool
	Console        bool
	FileName       string
	FileMaxBackups int
	FileMaxSize    int
	FileMaxAge     int
}

func (f *Config) registerFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.Trace, "log.trace", false, "trace logging")
	fs.StringVar(&f.FileName, "log.file.name", "", "log file name")
	fs.IntVar(&f.FileMaxBackups, "log.file.maxbackups", 5, "max log file backups")
	fs.IntVar(&f.FileMaxSize, "log.file.maxsize", 10, "max log file size (megabytes)")
	fs.IntVar(&f.FileMaxAge, "log.file.maxage", 98, "max log file age (days)")
}

// Register registers the flags in a flag.FlagSet with defaults for console logging
func (f *Config) RegisterFlagsConsole(fs *flag.FlagSet) {
	f.registerFlags(fs)
	fs.BoolVar(&f.Debug, "log.debug", true, "debug logging")
	fs.BoolVar(&f.Console, "log.console", true, "console formatter")
}

// RegisterFlags registers the flags in a flag.FlagSet with default for json logging
func (f *Config) RegisterFlags(fs *flag.FlagSet) {
	f.registerFlags(fs)
	fs.BoolVar(&f.Debug, "log.debug", false, "debug logging")
	fs.BoolVar(&f.Console, "log.console", false, "console formatter")
}

// Setup sets up logging accorind to Flags values.
func (f Config) Setup() error {
	if f.Trace {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else if f.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if f.FileName != "" {
		w := &lumberjack.Logger{
			Filename:   f.FileName,
			MaxBackups: f.FileMaxBackups,
			MaxSize:    f.FileMaxSize,
			MaxAge:     f.FileMaxAge,
		}
		SetBlockingLogger(w)
	} else if f.Console {
		SetConsoleLogger()
	} else {
		SetBlockingLogger(os.Stderr)
	}
	return nil
}
