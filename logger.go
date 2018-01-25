package logger

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	_LEVEL_UNKNOWN int = -1

	LEVEL_FATAL int = iota
	LEVEL_ERROR
	LEVEL_WARNING
	LEVEL_INFO
	LEVEL_DEBUG
)

type ProviderInterface interface {
	GetID() string
	Write(p []byte) (n int, err error) // io.Writer
}

type Logger struct {
	mu   sync.RWMutex
	once sync.Once

	level     int
	buf       *bytes.Buffer
	prefix    string
	host      string
	providers map[int][]ProviderInterface
}

func NewLogger() *Logger {
	return new(Logger)
}

func (l *Logger) SetLevel(val int) {
	l.internalInit()

	l.mu.Lock()
	l.level = val
	l.mu.Unlock()
}

func (l *Logger) RegisterProvider(p ProviderInterface) {
	l.internalInit()

	l.mu.Lock()
	defer l.mu.Unlock()

	list, ok := l.providers[_LEVEL_UNKNOWN]
	if !ok {
		list = make([]ProviderInterface, 0)
	}

	var exist bool
	for _, val := range list {
		if val.GetID() == p.GetID() {
			exist = true
			break
		}
	}

	if !exist {
		l.providers[_LEVEL_UNKNOWN] = append(list, p)
	}
}

func (l *Logger) AddFatalProvider(idsList ...string) {
	for _, id := range idsList {
		l.AddProvider(id, LEVEL_FATAL)
	}
}

func (l *Logger) AddErrorProvider(idsList ...string) {
	for _, id := range idsList {
		l.AddProvider(id, LEVEL_ERROR)
	}
}

func (l *Logger) AddWarningProvider(idsList ...string) {
	for _, id := range idsList {
		l.AddProvider(id, LEVEL_WARNING)
	}
}

func (l *Logger) AddInfoProvider(idsList ...string) {
	for _, id := range idsList {
		l.AddProvider(id, LEVEL_INFO)
	}
}

// OLD API (recommended use AddInfoProvider)
func (l *Logger) AddLogProvider(idsList ...string) {
	for _, id := range idsList {
		l.AddProvider(id, LEVEL_INFO)
	}
}

func (l *Logger) AddDebugProvider(idsList ...string) {
	for _, id := range idsList {
		l.AddProvider(id, LEVEL_DEBUG)
	}
}

func (l *Logger) AddProvider(id string, levelList ...int) {
	l.internalInit()

	l.mu.Lock()
	defer l.mu.Unlock()

	var provider ProviderInterface
	for _, p := range l.providers[_LEVEL_UNKNOWN] {
		if p.GetID() == id {
			provider = p
			break
		}
	}

	if provider == nil {
		l.Fatal("Unknown provider id:", id)
	}

	for _, level := range levelList {

		list, ok := l.providers[level]
		if !ok {
			list = make([]ProviderInterface, 0)
		}

		var exist bool
		for _, p := range list {
			if p.GetID() == id {
				exist = true
				break
			}
		}

		if !exist {
			l.providers[level] = append(list, provider)
		}
	}
}

func (l *Logger) Prefix() string {
	l.internalInit()

	l.mu.RLock()
	pref := l.prefix
	l.mu.RUnlock()

	return pref
}

func (l *Logger) SetPrefix(prefix string) {
	l.internalInit()

	l.mu.Lock()
	l.prefix = prefix
	l.mu.Unlock()
}

var _COMPLEX_DELEMETER = []byte{':', ' '}

func (l *Logger) Output(calldepth int, level int, s string) error {
	l.internalInit()

	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}

	nowStr := time.Now().Format("2006-01-02 15:04:05.0000000 -07:00")
	lineStr := strconv.Itoa(line)

	l.mu.Lock()
	l.buf.Reset()
	l.buf.WriteString(levelToString(level))
	l.buf.Write(_COMPLEX_DELEMETER)
	l.buf.WriteString(nowStr)
	if len(l.host) > 0 {
		l.buf.WriteByte(' ')
		l.buf.WriteString(l.host)
	}
	if len(l.prefix) > 0 {
		l.buf.WriteByte('-')
		l.buf.WriteString(l.prefix)
	}
	l.buf.WriteByte(' ')
	l.buf.Write([]byte(file))
	l.buf.WriteByte(':')
	l.buf.WriteString(lineStr)
	l.buf.Write(_COMPLEX_DELEMETER)
	l.buf.WriteString(s)
	l.buf.WriteByte('\n')

	data := l.buf.Bytes()
	for _, pr := range l.providers[level] {
		pr.Write(data)
	}
	l.mu.Unlock()

	return nil
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.printf(LEVEL_ERROR, format, v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.print(LEVEL_ERROR, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.printf(LEVEL_INFO, format, v...)
}

func (l *Logger) Info(v ...interface{}) {
	l.print(LEVEL_INFO, v...)
}

// OLD API (recommended use Infof)
func (l *Logger) Logf(format string, v ...interface{}) {
	l.printf(LEVEL_INFO, format, v...)
}

// OLD API (recommended use Info)
func (l *Logger) Log(v ...interface{}) {
	l.print(LEVEL_INFO, v...)
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	l.printf(LEVEL_WARNING, format, v...)
}

func (l *Logger) Warning(v ...interface{}) {
	l.print(LEVEL_WARNING, v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.printf(LEVEL_DEBUG, format, v...)
}

func (l *Logger) Debug(v ...interface{}) {
	l.print(LEVEL_DEBUG, v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.printf(LEVEL_FATAL, format, v...)
	os.Exit(1)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.print(LEVEL_FATAL, v...)
	os.Exit(1)
}

func (l *Logger) print(level int, v ...interface{}) {
	if l.isValidLogLevel(level) {
		l.Output(3, level, fmt.Sprintln(v...))
	}
}

func (l *Logger) printf(level int, format string, v ...interface{}) {
	if l.isValidLogLevel(level) {
		l.Output(3, level, fmt.Sprintf(format, v...))
	}
}

func (l *Logger) isValidLogLevel(level int) bool {
	l.internalInit()

	l.mu.RLock()
	lvl := l.level
	l.mu.RUnlock()

	return level <= lvl
}

func (l *Logger) internalInit() {
	l.once.Do(func() {
		l.buf = bytes.NewBuffer(make([]byte, 0, 4096))
		if len(l.prefix) == 0 {
			l.prefix = path.Base(os.Args[0])
		}

		if val, err := os.Hostname(); err == nil {
			l.host = val
		}

		if l.providers == nil {
			l.providers = make(map[int][]ProviderInterface)
		}

		if l.level == 0 {
			l.level = LEVEL_DEBUG
		}
	})
}

func levelToString(l int) string {
	switch l {
	case LEVEL_FATAL:
		return "FTL"
	case LEVEL_ERROR:
		return "ERR"
	case LEVEL_WARNING:
		return "WRN"
	case LEVEL_INFO:
		return "INF"
	case LEVEL_DEBUG:
		return "DBG"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}
