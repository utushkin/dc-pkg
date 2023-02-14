package logging

import (
	"bytes"
	"fmt"
	"github.com/utushkin/dc-pkg/printer"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/utushkin/dc-pkg/env"

	"github.com/sirupsen/logrus"
)

type writerHook struct {
	Writer    []io.Writer
	LogLevels []logrus.Level
}

func (hook *writerHook) Fire(entry *logrus.Entry) (err error) {
	line, err := entry.String()
	if err != nil {
		return err
	}
	for _, w := range hook.Writer {
		_, err = w.Write([]byte(line))
	}
	return
}

func (hook *writerHook) Levels() []logrus.Level {
	return hook.LogLevels
}

type sendHook struct {
	LogLevels  []logrus.Level
	TelegramId string
}

func repScore(msg string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(msg, "\u001B[0;37m", ""), "\u001B[0m", ""), "\u001B[0;34m", ""), "\t", ""), "\"", "'"), "\\'", ""), "\n", "")
}

func (hook *sendHook) Fire(entry *logrus.Entry) error {
	printer.PrintBlue(entry.Message, entry.Time.Format("02-01-2006 15:04.999"), entry.Level, entry.Caller.Entry, entry.Caller.Line, entry.Caller.Function, entry.Caller.Func, entry.Data)
	line, err := entry.String()
	if err != nil {
		return err
	}
	printer.PrintCyan(repScore(line), len(line))

	message := fmt.Sprintf("Time: %s; LogLevel: _%s_; %s:%d %s", entry.Time.Format("02-01-2006 15:04.999"), entry.Level, entry.Caller.File, entry.Caller.Line, repScore(entry.Message))

	req, err := http.NewRequest("POST", "http://localhost:8001/set/", bytes.NewBuffer([]byte(fmt.Sprintf("{\"message\":\"%s\",\"id\":\"%s\"}", message, hook.TelegramId))))
	req.Header.Set("Content-Type", "application/json")
	printer.PrintCyan(repScore(line))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
		fmt.Println(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	return err
}

func (hook *sendHook) Levels() []logrus.Level {
	return hook.LogLevels
}

var e *logrus.Entry

type Logger struct {
	*logrus.Entry
	LogLevel string
}

func GetLogger() *Logger {
	return &Logger{Entry: e, LogLevel: strings.ToLower(env.GetEnv("LOG_LEVEL", "ERROR"))}
}

func (l *Logger) GetLoggerWithField(k string, v interface{}) *Logger {
	return &Logger{Entry: l.WithField(k, v)}
}

func InitLogger() {
	l := logrus.New()
	l.SetReportCaller(true)
	l.Formatter = &logrus.TextFormatter{
		ForceColors: true,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			filename := path.Base(frame.File)
			if env.GetEnvAsBool("LOG_COLOR", false) {
				return fmt.Sprintf("\u001B[0;37m%s()\u001B[0m", frame.Function), fmt.Sprintf("\u001B[0;34m%s:%d\u001B[0m", filename, frame.Line)
			} else {
				return fmt.Sprintf("%s()", frame.Function), fmt.Sprintf("%s:%d", filename, frame.Line)
			}
		},
		DisableColors:   false,
		FullTimestamp:   true,
		TimestampFormat: "02-01-2006 15:04.999",
	}

	if env.GetEnvAsBool("LOG_WRITE", true) {
		err := os.MkdirAll("logs", 0644)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		err = os.Chmod("logs", 0777)
		if err != nil {
			fmt.Println(err)
		}

		allFile, err := os.OpenFile("logs/all.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
		if err != nil {
			fmt.Println(err)
		}

		l.SetOutput(io.Discard)

		l.AddHook(&writerHook{
			Writer:    []io.Writer{allFile, os.Stdout},
			LogLevels: logrus.AllLevels,
		})
	}

	l.AddHook(&sendHook{
		LogLevels:  []logrus.Level{logrus.FatalLevel, logrus.PanicLevel, logrus.ErrorLevel},
		TelegramId: env.GetEnv("LOG_TELEGRAM_SENDER", "521804630"),
	})

	switch strings.ToLower(env.GetEnv("LOG_LEVEL", "ERROR")) {
	case "info":
		l.SetLevel(logrus.InfoLevel)
	case "debug":
		l.SetLevel(logrus.DebugLevel)
	case "trace":
		l.SetLevel(logrus.TraceLevel)
	case "error":
		l.SetLevel(logrus.ErrorLevel)
	case "panic":
		l.SetLevel(logrus.PanicLevel)
	case "fatal":
		l.SetLevel(logrus.FatalLevel)
	default:
		l.SetLevel(logrus.InfoLevel)
	}

	e = logrus.NewEntry(l)
}
