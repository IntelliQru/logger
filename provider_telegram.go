package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"time"
)

const PROVIDER_TELEGRAM = "telegram"

type TelegramProvider struct {
	buf        *bytes.Buffer
	req        *http.Request
	httpClient *http.Client
	chatIds    []string
	mu         sync.Mutex
	once       sync.Once
}

func NewTelegramProvider(token string, chatIds []string) (*TelegramProvider, error) {

	if len(token) == 0 {
		return nil, errors.New("Telegram token is empty.")
	}
	if chatIds == nil {
		return nil, errors.New("Empty telegram chat ids.")
	}

	var url string
	if strings.HasPrefix(token, "https://api.telegram.org/bot") && strings.HasSuffix(token, "sendMessage") {
		url = token // OLD API
	} else {
		url = fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	provider := &TelegramProvider{
		req:     req,
		chatIds: chatIds,
	}

	return provider, nil
}

func (p *TelegramProvider) GetID() string {
	return PROVIDER_TELEGRAM
}

var (
	_MESSAGE_DELEMER = []byte("\n\n------------------\n")
	_TIME_DELEMER    = []byte(":\n")
)

func (p *TelegramProvider) Write(data []byte) (n int, err error) {
	p.internalInit()

	if len(data) == 0 {
		return 0, nil
	}

	p.mu.Lock()
	if p.buf.Len() > 0 {
		p.buf.Write(_MESSAGE_DELEMER)
	}
	p.buf.WriteString(time.Now().Format(time.RFC3339Nano))
	p.buf.Write(_TIME_DELEMER)
	p.buf.Write(data)
	p.mu.Unlock()

	return len(data), nil
}

func (p *TelegramProvider) send() (n int, err error) {

	p.mu.Lock()
	defer func() {
		p.mu.Unlock()
	}()

	if p.buf.Len() == 0 {
		return 0, nil
	}

	// Telegram response example:
	// {
	//      "ok":false,
	//      "error_code":404,
	//      "description":"Not Found: method not found"
	//      "result":{"id":290082045,....}
	// }

	type Response struct {
		Ok          bool        `json:"ok"`
		ErrorCode   int32       `json:"error_code,omitempty"`
		Description string      `json:"description,omitempty"`
		Parameters  interface{} `json:"parameters,omitempty"`
		Result      interface{} `json:"result,omitempty"`
	}

	text := p.buf.String()

	for _, chatId := range p.chatIds {
		jd, err := json.Marshal(map[string]string{
			"chat_id": chatId,
			"text":    text,
		})

		if err != nil {
			return 0, err
		}

		body := bytes.NewReader(jd)

		p.req.Body = ioutil.NopCloser(body)
		p.req.ContentLength = int64(len(jd))
		p.req.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(body), nil
		}

		resp, err := p.httpClient.Do(p.req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			continue
		}

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}

		if len(respBody) == 0 {
			return 0, fmt.Errorf("Failed send message to telegram: %+v", resp.Header)
		} else {
			r := new(Response)
			if err := json.Unmarshal(respBody, r); err != nil {
				r.Description = err.Error()
			}

			return 0, fmt.Errorf("Failed send message to telegram: %+v", r)
		}
	}

	p.buf.Reset()
	return p.buf.Len(), nil
}

func (p *TelegramProvider) flush() {
	go func() {
		for {
			time.Sleep(time.Second)
			p.send()
		}
	}()
}

func (p *TelegramProvider) internalInit() {
	p.once.Do(func() {
		p.buf = new(bytes.Buffer)
		p.httpClient = new(http.Client)
		p.flush()
	})
}
