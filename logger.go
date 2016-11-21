package logger

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	LEVEL_ERROR int = iota
	LEVEL_INFO
	LEVEL_DEBUG

	TRACE_TYPE_ONE int = 0 // Only the caller function
	TRACE_TYPE_ANY int = 1 // Full stack trace
)

type ProviderInterface interface {
	GetID() string
	Log(msg []byte)
	Error(msg []byte)
	Fatal(msg []byte)
	Debug(msg []byte)
}

type Logger struct {
	providers      map[string]*ProviderInterface
	logProviders   []string
	errorProviders []string
	fatalProviders []string
	debugProviders []string
	level          int
	traceType      int
}

func NewLogger() *Logger {
	return &Logger{
		providers: make(map[string]*ProviderInterface, 0),
	}
}

func (l *Logger) SetTraceType(val int) error {

	if val != TRACE_TYPE_ONE && val != TRACE_TYPE_ANY {
		return errors.New("Unknown trace type.")
	}

	l.traceType = val
	return nil
}

func (l *Logger) SetLevel(level int) {
	l.level = level
}

func (l *Logger) RegisterProvider(p ProviderInterface) {
	l.providers[p.GetID()] = &p
}

func (l *Logger) AddLogProvider(provIDs ...string) {
	l.addProvider("log", provIDs...)
}

func (l *Logger) AddErrorProvider(provIDs ...string) {
	l.addProvider("error", provIDs...)
}

func (l *Logger) AddFatalProvider(provIDs ...string) {
	l.addProvider("fatal", provIDs...)
}

func (l *Logger) AddDebugProvider(provIDs ...string) {
	l.addProvider("debug", provIDs...)
}

func (l *Logger) addProvider(providerType string, providersIDs ...string) {

	var IDs *[]string
	switch providerType {
	case "debug":
		IDs = &l.debugProviders
	case "log":
		IDs = &l.logProviders
	case "error":
		IDs = &l.errorProviders
	case "fatal":
		IDs = &l.fatalProviders
	default:
		panic("Wrong type of the provider.")
	}

	alreadyRegistred := func(id string, idsList *[]string) bool {
		for _, val := range *idsList {
			if val == id {
				return true
			}
		}
		return false
	}

	for _, id := range providersIDs {

		provider, bFound := l.providers[id]
		if bFound {
			pID := (*provider).GetID()
			if !alreadyRegistred(pID, IDs) {
				*IDs = append(*IDs, pID)
			}
		}
	}
}

func (l *Logger) Logf(format string, params ...interface{}) {
	if l.level < LEVEL_INFO {
		return
	}

	l.Log(fmt.Sprintf(format, params...))
}

func (l *Logger) Log(messageParts ...interface{}) {
	if l.level < LEVEL_INFO {
		return
	}
	msg := makeMessage("LOG", messageParts, l.traceType)
	for _, pID := range l.logProviders {
		p, bFound := l.providers[pID]
		if bFound {
			(*p).Log(msg)
		}
	}
}

func (l *Logger) Errorf(format string, params ...interface{}) {
	l.Error(fmt.Sprintf(format, params...))
}

func (l *Logger) Error(messageParts ...interface{}) {

	msg := makeMessage("ERROR", messageParts, l.traceType)
	for _, pID := range l.errorProviders {
		p, bFound := l.providers[pID]
		if bFound {
			(*p).Error(msg)
		}
	}
}

func (l *Logger) Debugf(format string, params ...interface{}) {
	if l.level < LEVEL_DEBUG {
		return
	}

	l.Debug(fmt.Sprintf(format, params...))
}

func (l *Logger) Debug(messageParts ...interface{}) {
	if l.level < LEVEL_DEBUG {
		return
	}

	msg := makeMessage("DEBUG", messageParts, l.traceType)
	for _, pID := range l.debugProviders {
		p, bFound := l.providers[pID]
		if bFound {
			(*p).Debug(msg)
		}
	}
}

func (l *Logger) Fatalf(format string, params ...interface{}) {
	l.Fatal(fmt.Sprintf(format, params...))
}

func (l *Logger) Fatal(messageParts ...interface{}) {
	msg := makeMessage("FATAL", messageParts, l.traceType)
	for _, pID := range l.fatalProviders {
		p, bFound := l.providers[pID]
		if bFound {
			(*p).Fatal(msg)
		}
	}

	os.Exit(1)
}

var (
	HOST                 string
	MESSAGE_REPLACER     = strings.NewReplacer("\r", "", "\n", "\t")
	MESSAGE_SEPARATOR    = []byte(" ")
	LOGGER_LINE_REPLACER = strings.NewReplacer(": ", "", " ", "", "\n", ", ")
)

func makeMessage(typeLog string, err []interface{}, traceType int) []byte {

	if len(HOST) == 0 {
		HOST, _ = os.Hostname()
	}

	buf := bytes.NewBuffer(nil)
	lineBuf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", log.Lshortfile)

	for i := 2; i < 6; i++ {
		logger.Output(i, "")
		val := buf.String() // example: <filename>:<line number>:\n => testing.go:107\n
		buf.Reset()

		if strings.HasPrefix(val, "logger.go:") {
			continue // skip current module
		} else if strings.HasPrefix(val, "testing.go:") {
			break
		}

		lineBuf.WriteString(val)

		if traceType == TRACE_TYPE_ONE {
			break
		}
	}

	line := strings.TrimRight(LOGGER_LINE_REPLACER.Replace(lineBuf.String()), ", ")
	prefix := fmt.Sprintf("%s: %s %s %s: ", typeLog, time.Now().Format(time.RFC3339), HOST, line)
	logger.SetFlags(0)
	logger.SetPrefix(prefix)

	msg := bytes.NewBuffer(nil)
	for i, v := range err {
		if i > 0 {
			msg.Write(MESSAGE_SEPARATOR)
		}
		fmt.Fprint(msg, v)
	}

	// Example:
	//  ERROR: 2016-11-21T14:50:23+03:00 khramtsov logger.go:124, logger_test.go:102: message text
	logger.Output(0, MESSAGE_REPLACER.Replace(msg.String()))

	return bytes.Replace(buf.Bytes(), []byte("\n"), []byte{}, -1)
}
