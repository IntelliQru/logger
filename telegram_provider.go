package logger

import (
	"bytes"
	"errors"
	"net/http"
)

const PROVIDER_TELEGRAM = "telegram"

type TelegramProvider struct {
	url     string
	chatIds []string
}

func NewTelegramProvider(url string, chatIds []string) (*TelegramProvider, error) {

	if len(url) == 0 {
		return nil, errors.New("Empty telegram url.")
	} else if chatIds == nil {
		return nil, errors.New("Empty telegram chat ids.")
	}

	provider := &TelegramProvider{
		url:     url,
		chatIds: chatIds,
	}

	return provider, nil
}

func (p TelegramProvider) GetID() string {
	return PROVIDER_TELEGRAM
}

func (p TelegramProvider) Log(msg []byte) {
	p.send("Log message\n", msg)
}

func (p TelegramProvider) Error(msg []byte) {
	p.send("Error message\n", msg)
}

func (p TelegramProvider) Fatal(msg []byte) {
	p.send("Fatal message\n", msg)
}

func (p TelegramProvider) Debug(msg []byte) {
	p.send("Debug message\n", msg)
}

func (p TelegramProvider) send(subject string, body []byte) {
	go tg_send(p.url, p.chatIds, subject, body)
}

func tg_send(url string, chatIds []string, subject string, body []byte) {
	for _, chatId := range chatIds {
		msg := "{\"chat_id\":" + chatId + ",\"text\":" + "\"" + subject + string(body) + "\"" + "}"
		var jsonStr = []byte(msg)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, _ := client.Do(req)
		defer resp.Body.Close()
	}
}
