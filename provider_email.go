package logger

import (
	"errors"
	"fmt"
	"sync"

	"github.com/IntelliQru/mail"
)

const PROVIDER_EMAIL = "email"

type EmailProvider struct {
	ProviderInterface

	mu      sync.RWMutex
	message *mail.Message
}

func NewEmailProvider(address string, smtpClient *mail.SmtpClient) (*EmailProvider, error) {

	if len(address) == 0 {
		return nil, errors.New("Empty email address.")
	} else if smtpClient == nil {
		return nil, errors.New("Empty smtp client.")
	} else if _, err := smtpClient.Connection(); err != nil {
		return nil, fmt.Errorf("Failed create email provider: %s", err)
	}

	provider := &EmailProvider{
		message: mail.NewMessage(smtpClient, address, "Logger", ""),
	}

	return provider, nil
}

func (p *EmailProvider) GetID() string {
	return PROVIDER_EMAIL
}

func (p *EmailProvider) Write(data []byte) (n int, err error) {

	if len(data) == 0 {
		return 0, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.message.Body = string(data)

	if err := p.message.SendMail(); err != nil {
		return 0, err
	}

	return len(data), nil
}
