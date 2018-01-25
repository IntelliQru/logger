package logger

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
)

type BufProvider struct {
	Buf *bytes.Buffer
}

func (b *BufProvider) GetID() string                     { return "BUF" }
func (b *BufProvider) Write(p []byte) (n int, err error) { return b.Buf.Write(p) }

func TestAddProvider(t *testing.T) {
	cp := new(ConsoleProvider)

	l := new(Logger)

	for i, lvl := range []int{LEVEL_DEBUG, LEVEL_INFO, LEVEL_WARNING, LEVEL_ERROR, LEVEL_FATAL} {
		l.RegisterProvider(cp)
		l.RegisterProvider(cp)

		if len(l.providers) != i+1 {
			t.Error(l.providers)
		} else if len(l.providers[_LEVEL_UNKNOWN]) != 1 {
			t.Error(l.providers)
		}

		l.AddProvider(cp.GetID(), lvl)
		l.AddProvider(cp.GetID(), lvl)
		if len(l.providers) != i+2 {
			t.Error(l.providers)
		} else if len(l.providers[_LEVEL_UNKNOWN]) != 1 {
			t.Error(l.providers)
		} else if len(l.providers[lvl]) != 1 {
			t.Error(l.providers)
		}
	}

}

func TestLevels(t *testing.T) {

	bufProvider := &BufProvider{
		Buf: bytes.NewBuffer(nil),
	}

	// example:
	// ERR: 2017-05-31 22:29:11.7489315 +03:00 <host>/<app>: logger_test.go:41: err
	// WRN: 2017-05-31 22:29:11.7489315 +03:00 <host>/<app>: logger_test.go:41: wrn
	expr := regexp.MustCompile(`: \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{7} [+-]\d{2}:\d{2} [^ ]+`)

	checkOutput := func(lvl int, line int) {
		defer bufProvider.Buf.Reset()

		msg := bufProvider.Buf.String()
		msg = strings.TrimSpace(expr.ReplaceAllString(msg, ""))

		// example: ERR logger_test.go:87: ERR
		etalon := fmt.Sprintf("%s logger_test.go:%d: %s", levelToString(lvl), line, strings.ToLower(levelToString(lvl)))

		if msg != etalon {
			t.Errorf("Failed massage: '%s' != '%s' (src: '%s')", etalon, msg, bufProvider.Buf.String())
		}
	}

	l := new(Logger)

	l.RegisterProvider(bufProvider)
	l.AddProvider(bufProvider.GetID(), LEVEL_DEBUG, LEVEL_INFO, LEVEL_WARNING, LEVEL_ERROR, LEVEL_FATAL)

	for _, lvl := range []int{LEVEL_DEBUG, LEVEL_INFO, LEVEL_WARNING, LEVEL_ERROR, LEVEL_FATAL} {
		l.SetLevel(lvl)

		for i, testData := range []struct {
			Level  int
			Print  func(v ...interface{})
			Printf func(format string, v ...interface{})
		}{
			{LEVEL_ERROR, l.Error, l.Errorf},
			{LEVEL_WARNING, l.Warning, l.Warningf},
			{LEVEL_INFO, l.Log, l.Logf},
			{LEVEL_INFO, l.Info, l.Infof},
			{LEVEL_DEBUG, l.Debug, l.Debugf},
		} {
			_, _, line, _ := runtime.Caller(0)
			var funcLine = line - 6 + i // 6 - Look at the six lines above

			text := strings.ToLower(levelToString(testData.Level))

			testData.Print(text)
			if lvl >= testData.Level {
				checkOutput(testData.Level, funcLine)
			} else if bufProvider.Buf.Len() != 0 {
				t.Error("Fail")
			}

			testData.Printf(text)
			if lvl >= testData.Level {
				checkOutput(testData.Level, funcLine)
			} else if bufProvider.Buf.Len() != 0 {
				t.Error("Fail")
			}
		}
	}
}

func TestMultyThreads(t *testing.T) {

	bufProvider := &BufProvider{
		Buf: bytes.NewBuffer(nil),
	}

	l := NewLogger()
	l.SetLevel(LEVEL_DEBUG)
	l.RegisterProvider(bufProvider)
	l.AddDebugProvider(bufProvider.GetID())

	defer func() {
		l.Debug("end")
	}()

	wg := new(sync.WaitGroup)

	for thread := 0; thread < 1000; thread++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < 1000; i++ {
				l.Debug("test debug")
				if i%10 == 9 {
					l.Error("test error")
				}
			}
		}()
	}

	wg.Wait()

	if bufProvider.Buf.Len() == 0 {
		t.Error("Fail")
	}
}

func BenchmarkOutput(b *testing.B) {

	b.Run("Symply", func(b *testing.B) {
		cp := new(ConsoleProvider)

		l := NewLogger()
		l.SetLevel(LEVEL_DEBUG)
		l.RegisterProvider(cp)
		l.AddDebugProvider(cp.GetID())

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.Debug("benchmark\n")
			}
		})
	})
}
