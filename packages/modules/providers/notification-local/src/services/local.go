// Package services mirrors @medusajs/notification-local/src/services.
package services

import (
	"fmt"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// LocalNotificationService mirrors LocalNotificationService: it only logs.
type LocalNotificationService struct {
	logger *slog.Logger
}

// NewLocalNotificationService builds the provider.
func NewLocalNotificationService(logger *slog.Logger) *LocalNotificationService {
	if logger == nil {
		logger = slog.Default()
	}
	return &LocalNotificationService{logger: logger}
}

// Send mirrors send: it returns {} (no external id).
func (s *LocalNotificationService) Send(notification *dal.Record) (*dal.Record, error) {
	if notification == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No notification information provided")
	}
	str := func(k string) string {
		v, ok := notification.Get(k)
		if !ok {
			return "undefined"
		}
		return fmt.Sprint(v)
	}
	data := "undefined"
	if v, ok := notification.Get("data"); ok {
		if b, err := dal.MarshalValue(v); err == nil {
			data = string(b)
		}
	}
	s.logger.Info(fmt.Sprintf("Attempting to send a notification to: '%s' on the channel: '%s' with template: '%s' and data: '%s'", str("to"), str("channel"), str("template"), data))
	return dal.NewRecord(), nil
}
