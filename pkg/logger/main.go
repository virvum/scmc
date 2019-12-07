package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

type Level int

const (
	Trace Level = iota
	Debug
	Info
	Warn
	Error
	Fatal
)

var LogLevels []string = []string{"trace", "debug", "info", "warn", "error", "fatal"}

func (l Level) String() string {
	return LogLevels[l]
}

func (l Level) Type() string {
	return "string"
}

func (l *Level) Set(level string) error {
	switch strings.ToLower(level) {
	case "trace":
		*l = Trace
	case "debug":
		*l = Debug
	case "info":
		*l = Info
	case "warn", "warning":
		*l = Warn
	case "error":
		*l = Error
	case "fatal":
		*l = Fatal
	default:
		return fmt.Errorf("invalid log level, use one of %v (case insensitive)", LogLevels)
	}

	return nil
}

func (l Level) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string

	if err := unmarshal(&s); err != nil {
		return err
	}

	if err := l.Set(s); err != nil {
		return err
	}

	return nil
}

type Log struct {
	Level    Level
	Color    bool
	RootPath string
}

func New(level Level, color bool, rootPath string) Log {
	return Log{
		Level:    level,
		Color:    color,
		RootPath: rootPath,
	}
}

func (l *Log) log(level Level, format string, args []interface{}) {
	if level < l.Level {
		return
	}

	var color string

	now := time.Now()
	message := strings.ReplaceAll(fmt.Sprintf(format, args...), "\n", " ")
	_, file, line, ok := runtime.Caller(2)

	if !ok {
		file = "???"
		line = 0
	} else if l.RootPath != "" {
		file = strings.TrimPrefix(file, l.RootPath)
	}

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		if l.Color {
			switch level {
			case Debug:
				color = "37"
			case Info:
				color = "34"
			case Warn:
				color = "33"
			case Error, Fatal:
				color = "91"
			}

			fmt.Printf("%s \033[%sm%-5s\033[0m %s:%d: %s\n", now.Format("2006-01-02 15:04:05"), color, strings.ToUpper(level.String()), file, line, message)
		} else {
			fmt.Printf("%s %-5s %s:%d: %s\n", now.Format("2006-01-02 15:04:05"), strings.ToUpper(level.String()), file, line, message)
		}
	}
}

func (l *Log) IsTrace() bool {
	return l.Level <= Trace
}

func (l *Log) IsDebug() bool {
	return l.Level <= Debug
}

func (l *Log) IsInfo() bool {
	return l.Level <= Info
}

func (l *Log) IsWarn() bool {
	return l.Level <= Warn
}

func (l *Log) IsError() bool {
	return l.Level <= Error
}

func (l *Log) Trace(format string, args ...interface{}) {
	l.log(Trace, format, args)
}

func (l *Log) Debug(format string, args ...interface{}) {
	l.log(Debug, format, args)
}

func (l *Log) Info(format string, args ...interface{}) {
	l.log(Info, format, args)
}

func (l *Log) Warn(format string, args ...interface{}) {
	l.log(Warn, format, args)
}

func (l *Log) Error(format string, args ...interface{}) {
	l.log(Error, format, args)
}

func (l *Log) Fatal(format string, args ...interface{}) {
	l.log(Fatal, format, args)
	os.Exit(1)
}
