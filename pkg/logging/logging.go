package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	isatty_pkg "github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"

	"Rhoc/pkg/storage"
)

var (
	logWriter  *io.PipeWriter
	logExt     string = ".log"
	timeLayout string = "2006-01-02-15-04-05"
	repeatLogs bool   = false
)

// GetLogWriter gives access to `logWriter` without the possibility of changing it
func GetLogWriter() io.PipeWriter {
	return *logWriter
}

// GetRepeatLogs gives access to `repeatLogs` without the possibility of changing it
func GetRepeatLogs() bool {
	return repeatLogs
}

type textHook struct {
	out    io.Writer
	logger *log.Logger
	fmt    log.TextFormatter
	levels []log.Level
}

func (hook *textHook) Levels() []log.Level {
	return hook.levels
}

func (hook *textHook) textWrite(args ...interface{}) error {
	if _, err := hook.out.Write(args[0].([]byte)); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot write format error message to %s: %s\n", hook.out, err)
		return err
	}

	return nil
}

func (hook *textHook) Fire(entry *log.Entry) error {
	if hook.logger != nil {
		newEntry := *entry // make a copy where we can swap logger without touching the entry
		newEntry.Logger = hook.logger

		switch entry.Level {
		case log.TraceLevel:
			newEntry.Trace(entry.Message)
		case log.DebugLevel:
			newEntry.Debug(entry.Message)
		case log.InfoLevel:
			newEntry.Info(entry.Message)
		case log.WarnLevel:
			newEntry.Warn(entry.Message)
		case log.ErrorLevel:
			newEntry.Error(entry.Message)
		case log.FatalLevel:
			newEntry.Error("[FATAL] " + entry.Message)
		case log.PanicLevel:
			newEntry.Error("[PANIC] " + entry.Message)
		default:
			newEntry.Print(entry.Message)
		}
	} else {
		if msg, err := hook.fmt.Format(entry); err != nil {
			if err1 := hook.textWrite(fmt.Sprintf("Cannot format entry %v: %s\n", entry, err)); err1 != nil {
				return err
			}
		} else {
			if err2 := hook.textWrite(msg); err2 != nil {
				return err
			}
		}
	}

	return nil
}

func isatty(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	return isatty_pkg.IsTerminal(file.Fd()) || isatty_pkg.IsCygwinTerminal(file.Fd())
}

func addHook(minlevel log.Level, maxlevel log.Level, out io.Writer) {
	levels := []log.Level{}

	for _, level := range log.AllLevels {
		if uint32(level) >= uint32(minlevel) && uint32(level) <= uint32(maxlevel) {
			levels = append(levels, level)
		}
	}

	isTerm := isatty(out)
	hook := textHook{
		out:    out,
		logger: nil,
		fmt: log.TextFormatter{
			DisableColors: !isTerm,
			FullTimestamp: !isTerm,
		},
		levels: levels,
	}

	if isTerm {
		// we have to add a "real" logrus logger pointing to terminal as hook.logger
		// to make logrus colorize terminal-targeted output, as logrus detects
		// when and how to colorize by inspecting entry.Logger.Out
		logger := log.New()
		logger.SetOutput(out)
		logger.SetLevel(log.TraceLevel)
		hook.out = logger.Writer()
		hook.logger = logger
	}

	log.AddHook(&hook)
}

// InitLogging sets up logging, including the one for external actions launched via commands
func InitLogging(verbose bool) {
	logger := log.StandardLogger()
	logWriter = logger.WriterLevel(log.InfoLevel)
	repeatLogs = verbose

	log.SetOutput(ioutil.Discard)

	if verbose {
		addHook(log.PanicLevel, log.WarnLevel, os.Stderr)
		addHook(log.InfoLevel, log.InfoLevel, os.Stdout)
	} else {
		addHook(log.PanicLevel, log.ErrorLevel, os.Stderr)
	}

	mainLogPrefix := storage.MakeStorageFilename(storage.LogCategory, []string{"Rhoc"}, "")
	lazyLog := &lazyFile{prefix: mainLogPrefix}

	log.RegisterExitHandler(func() {
		lazyLog.Close()
	})

	if verbose {
		addHook(log.PanicLevel, log.DebugLevel, lazyLog)
	} else {
		addHook(log.PanicLevel, log.InfoLevel, lazyLog)
	}
	// set log level of default logger to Trace so all hooks are called,
	// as hooks themselves determine which levels they want
	log.SetLevel(log.TraceLevel)
}

// MakeLogWriter creates a logfile name based on prefix and current datetime,
// creates the directory for it and opens it for writing
func MakeLogWriter(logfilePrefix string) (string, io.WriteCloser, error) {
	if logfilePrefix == "" {
		return "", nil, fmt.Errorf("requested to make empty logfile name")
	}

	logname := fmt.Sprintf("%s-%s%s", logfilePrefix, time.Now().Format(timeLayout), logExt)

	if err := storage.CreateDirForFile(logname); err != nil {
		log.WithFields(log.Fields{
			"path": logname,
		}).Errorf("makeLogWriter: cannot create directory for log file: %s", err)

		return "", nil, err
	}

	logfile, err := os.OpenFile(logname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0640)
	if err != nil {
		log.WithFields(log.Fields{
			"path": logname,
		}).Errorf("makeLogWriter: cannot open log file for writing: %s", err)

		return "", nil, err
	}

	return logname, logfile, nil
}
