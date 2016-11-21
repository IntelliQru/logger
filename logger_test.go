package logger

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// Example:
// source: 'ERROR: 2016-11-21T14:50:23+03:00 khramtsov logger.go:124, logger_test.go:102: message text'
// regexp: ': 2016-11-21T14:50:23+03:00 khramtsov logger.go:124, logger_test.go:102:'
var expr = regexp.MustCompile(`: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2} [^ ]+ ([^:]+:\d+)*:`)
var exprTimeUser = regexp.MustCompile(`: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2} [^ ]+`)

func TestAddProvider(t *testing.T) {

	providerInfo := Provider{Level: LEVEL_INFO}
	providerError := Provider{Level: LEVEL_ERROR}
	providerDebug := Provider{Level: LEVEL_DEBUG}

	l := NewLogger()
	l.RegisterProvider(&providerInfo)
	l.RegisterProvider(&providerError)
	l.RegisterProvider(&providerDebug)

	l.AddErrorProvider(providerError.GetID())
	if len(l.errorProviders) != 1 || l.errorProviders[0] != providerError.GetID() {
		t.Error("Failed registration of the provider 'error'.")
	}

	l.AddLogProvider(providerInfo.GetID(), providerInfo.GetID())
	if len(l.logProviders) != 1 || l.logProviders[0] != providerInfo.GetID() {
		t.Error("Failed registration of the provider 'log'.")
	}

	l.AddFatalProvider(providerInfo.GetID(), providerError.GetID())
	if len(l.fatalProviders) != 2 || l.fatalProviders[0] != providerInfo.GetID() || l.fatalProviders[1] != providerError.GetID() {
		t.Error("Failed registration of the provider 'fatal'.")
	}

	l.AddDebugProvider(providerDebug.GetID(), providerDebug.GetID(), providerError.GetID())
	if len(l.debugProviders) != 2 || l.debugProviders[0] != providerDebug.GetID() || l.debugProviders[1] != providerError.GetID() {
		t.Error("Failed registration of the provider 'debug'.")
	}
}

func TestMessage(t *testing.T) {

	// example:
	// Err: 2016-09-29T14:32:49+03:00 MyHost testing.go:610: text1 text2 text3text4
	// Info: 2016-09-29T14:32:49+03:00 MyHost testing.go:610: text1 text2 text3text4

	testData := []interface{}{"line1\nline2", "line3\r\nline4", "text1", "text2", 10}

	for _, prefix := range []string{"Err", "Info"} {
		msg := string(makeMessage(prefix, testData, TRACE_TYPE_ANY))
		result := expr.ReplaceAllString(msg, "")

		etalon := prefix + " line1\tline2 line3\tline4 text1 text2 10"
		if result != etalon {
			t.Errorf("Failed massage: \n'%s' !=\n'%s'\n(source: %s)", result, etalon, msg)
		}
	}
}

func TestFormat(t *testing.T) {

	provider := Provider{Level: LEVEL_DEBUG, NotCheckLevel: true}
	l := NewLogger()
	l.SetLevel(provider.Level)
	l.RegisterProvider(&provider)
	l.AddLogProvider(provider.GetID())
	l.AddErrorProvider(provider.GetID())
	l.AddFatalProvider(provider.GetID())
	l.AddDebugProvider(provider.GetID())

	for _, traceType := range []int{TRACE_TYPE_ONE, TRACE_TYPE_ANY} {
		l.SetTraceType(traceType)

		fn := func(messageType string) {
			if str := recover(); str != nil {
				// example:
				// Err: 2016-09-29T14:32:49+03:00 MyHost testing.go:610: text1 text2 text3text4
				// Info: 2016-09-29T14:32:49+03:00 MyHost testing.go:610: text1 text2 text3text4

				{
					msg := expr.ReplaceAllString(fmt.Sprintf("%v", str), "")
					msg = strings.Replace(msg, fmt.Sprintf("call %s: ", messageType), "", -1)

					// example:
					// ERROR format: text 10
					// INFO format: text 10
					etalon := fmt.Sprintf("%s format: text 10", strings.ToUpper(messageType))
					if msg != etalon {
						t.Errorf("Failed format: \n'%v' !=\n'%v'(source: %s)", msg, etalon, str)
					}
				}

				{
					msg := exprTimeUser.ReplaceAllString(fmt.Sprintf("%v", str), "")
					msg = strings.Replace(msg, fmt.Sprintf("call %s: ", messageType), "", -1)

					switch traceType {
					case TRACE_TYPE_ONE:
						etalon := fmt.Sprintf("%s logger_test.go:124: format: text 10", strings.ToUpper(messageType))

						if msg != etalon {
							t.Errorf("Failed format: \n'%v' !=\n'%v'(source: %s)", msg, etalon, str)
						}
					case TRACE_TYPE_ANY:
						etalon := fmt.Sprintf("%s logger_test.go:124, logger_test.go:138: format: text 10", strings.ToUpper(messageType))

						if msg != etalon {
							t.Errorf("Failed format: \n'%v' !=\n'%v'(source: %s)", msg, etalon, str)
						}
					}
				}
			}
		}

		func() {
			defer fn("Log")
			l.Logf("format: %s %d", "text", 10)
			l.Logf("format: text 10")

			defer fn("Debug")
			l.Debugf("format: %s %d", "text", 10)
			l.Debugf("format: text 10")

			defer fn("Error")
			l.Errorf("format: %s %d", "text", 10)
			l.Errorf("format: text 10")

			defer fn("Fatal")
			l.Fatalf("format: %s %d", "text", 10)
			l.Fatalf("format: text 10")
		}()
	}

}

func TestPringMessage(t *testing.T) {

	fn := func() {
		if str := recover(); str != nil {
			t.Errorf("%#v", str)
		}
	}

	for _, level := range []int{LEVEL_ERROR, LEVEL_INFO, LEVEL_DEBUG} {
		provider := Provider{Level: level}

		l := NewLogger()
		l.SetLevel(level)
		l.RegisterProvider(&provider)

		l.AddLogProvider(provider.GetID())
		l.AddErrorProvider(provider.GetID())
		l.AddFatalProvider(provider.GetID())
		l.AddDebugProvider(provider.GetID())

		defer fn()
		l.Log("text")

		defer fn()
		l.Logf("format: %s", "text")

		defer fn()
		l.Debug("text")

		defer fn()
		l.Debugf("format: text")
	}
}

type Provider struct {
	ProviderInterface
	Level         int
	NotCheckLevel bool
}

func (p Provider) GetID() string {
	return fmt.Sprintf("%d", p.Level)
}

func (p *Provider) Log(msg []byte) {
	if p.Level < LEVEL_INFO || p.NotCheckLevel {
		panic("call Log: " + string(msg))
	}
}

func (p *Provider) Debug(msg []byte) {
	if p.Level < LEVEL_DEBUG || p.NotCheckLevel {
		panic("call Debug: " + string(msg))
	}
}

func (p *Provider) Error(msg []byte) {
	panic("call Error: " + string(msg))
}

func (p *Provider) Fatal(msg []byte) {
	panic("call Fatal: " + string(msg))
}
