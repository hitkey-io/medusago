// Package steps mirrors @medusajs/core-flows/src/notification/steps.
package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// NotificationService is the part of INotificationModuleService the steps use.
type NotificationService interface {
	CreateNotifications(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
}

func notificationService(c composer.Container) NotificationService {
	s, _ := c.Resolve(utilsmodulessdk.NOTIFICATION).(NotificationService)
	return s
}

// NotifyOnFailureStep mirrors notifyOnFailureStep: the notifications are
// sent only when the workflow is compensated.
var NotifyOnFailureStep = composer.CreateStep("notify-on-failure",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[any], error) {
		return composer.NewStepResponse[any](nil, any(data)), nil
	},
	func(ctx composer.StepExecutionContext, input any) error {
		data, _ := input.([]*dal.Record)
		if data == nil {
			return nil
		}
		_, err := notificationService(ctx.Container).CreateNotifications(ctx.Context, data)
		return err
	})

// SendNotificationsStep mirrors sendNotificationsStep (not compensated).
var SendNotificationsStep = composer.CreateStep("send-notifications",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := notificationService(ctx.Container).CreateNotifications(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		ids := make([]any, len(created))
		for i, n := range created {
			ids[i], _ = n.Get("id")
		}
		return composer.NewStepResponse(created, any(ids)), nil
	}, nil)
