package logger

import (
	"os"
	"unicode/utf8"
)

const PROVIDER_CONSOLE = "console"

type ConsoleProvider struct {
	ProviderInterface
}

func (p *ConsoleProvider) GetID() string {
	return PROVIDER_CONSOLE
}

func (p *ConsoleProvider) Write(data []byte) (n int, err error) {
	removeNewLinesInText(data)
	return os.Stdout.Write(data)
}

func removeNewLinesInText(data []byte) {
	for i := 0; i < len(data)-1; {
		r, size := utf8.DecodeRune(data[i:])
		if r == '\r' || r == '\n' {
			data[i] = ' '
		}

		i += size
	}
}
