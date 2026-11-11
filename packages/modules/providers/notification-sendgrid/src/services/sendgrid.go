// Package services mirrors @medusajs/notification-sendgrid/src/services.
package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"syscall"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Identifier is SendgridNotificationService.identifier.
const Identifier = "notification-sendgrid"

const (
	sendgridBaseURL   = "https://api.sendgrid.com"
	sendgridUserAgent = "sendgrid/8.1.6;nodejs"
)

// SendgridNotificationService mirrors SendgridNotificationService.
type SendgridNotificationService struct {
	apiKey     string
	from       string
	logger     *slog.Logger
	httpClient *http.Client
}

// NewSendgridNotificationService mirrors the constructor (options api_key,
// from).
func NewSendgridNotificationService(options map[string]any, logger *slog.Logger) *SendgridNotificationService {
	if logger == nil {
		logger = slog.Default()
	}
	s := &SendgridNotificationService{logger: logger, httpClient: http.DefaultClient}
	s.apiKey, _ = options["api_key"].(string)
	s.from, _ = options["from"].(string)
	return s
}

func get(r *dal.Record, key string) (any, bool) {
	if r == nil {
		return nil, false
	}
	return r.Get(key)
}

// Send mirrors send: the notification becomes a @sendgrid/mail message
// (a dynamic template, or subject + html content) posted to /v3/mail/send.
func (s *SendgridNotificationService) Send(notification *dal.Record) (*dal.Record, error) {
	if notification == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No notification information provided")
	}

	message := dal.NewRecord()
	from := s.from
	if f, ok := notification.Get("from"); ok {
		if fs, isStr := f.(string); isStr && strings.TrimSpace(fs) != "" {
			from = strings.TrimSpace(fs)
		}
	}
	message.Set("from", from)
	if data, ok := notification.Get("data"); ok && data != nil {
		message.Set("dynamicTemplateData", data)
	}
	if list, ok := get(notification, "attachments"); ok {
		if atts, isList := list.([]any); isList {
			out := make([]any, len(atts))
			for i, a := range atts {
				ar, _ := a.(*dal.Record)
				att := dal.NewRecord()
				for _, k := range []string{"content", "filename", "content_type"} {
					if v, ok := get(ar, k); ok {
						att.Set(k, v)
					}
				}
				disposition, ok := get(ar, "disposition")
				if !ok || disposition == nil {
					disposition = "attachment"
				}
				att.Set("disposition", disposition)
				if id, ok := get(ar, "id"); ok && id != nil {
					att.Set("id", id)
				}
				out[i] = att
			}
			message.Set("attachments", out)
		}
	}
	providerData, _ := notification.Get("provider_data")
	pd, _ := providerData.(*dal.Record)
	personalizations, _ := get(pd, "personalizations")
	if list, ok := personalizations.([]any); ok && len(list) > 0 {
		message.Set("personalizations", list)
	} else if to, ok := notification.Get("to"); ok {
		message.Set("to", to)
	}
	if content, ok := notification.Get("content"); ok && content != nil {
		cr, _ := content.(*dal.Record)
		if v, ok := get(cr, "subject"); ok {
			message.Set("subject", v)
		}
		if v, ok := get(cr, "html"); ok {
			message.Set("html", v)
		}
	} else {
		template, _ := notification.Get("template")
		message.Set("templateId", template)
	}

	body, err := buildMail(message)
	if err == nil {
		err = s.post(body)
	}
	if err != nil {
		code, msg := "undefined", "unknown error"
		var re *responseError
		if errors.As(err, &re) {
			code = fmt.Sprint(re.code)
			if re.message != "" {
				msg = re.message
			}
		} else if errors.Is(err, syscall.ECONNREFUSED) {
			code = "ECONNREFUSED"
		}
		return nil, common.NewMedusaError(common.MedusaErrorTypes.UNEXPECTED_STATE, fmt.Sprintf("Failed to send email: %s - %s", code, msg))
	}
	return dal.NewRecord(), nil
}

// responseError mirrors @sendgrid/helpers ResponseError: code is the HTTP
// status, message the first error of the body.
type responseError struct {
	code    int
	message string
}

func (e *responseError) Error() string { return fmt.Sprint(e.code) }

// post mirrors client.request({ method: "POST", url: "/v3/mail/send", body }).
func (s *SendgridNotificationService) post(body []byte) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, sendgridBaseURL+"/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", sendgridUserAgent)
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	res, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		re := &responseError{code: res.StatusCode}
		if v, err := dal.DecodeOrderedJSON(raw); err == nil {
			errs, _ := get(asRecord(v), "errors")
			if list, ok := errs.([]any); ok && len(list) > 0 {
				if m, ok := get(asRecord(list[0]), "message"); ok && m != nil {
					re.message = fmt.Sprint(m)
				}
			}
		}
		return re
	}
	return nil
}

func asRecord(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}
