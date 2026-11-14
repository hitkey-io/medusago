package events

import (
	"log/slog"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// SendOrderConfirmationInput is the input of SendOrderConfirmationWorkflow.
type SendOrderConfirmationInput struct {
	ID string
}

// SendOrderConfirmationWorkflow stands for a workflow that sends an email.
var SendOrderConfirmationWorkflow = composer.CreateWorkflow("send-order-confirmation",
	func(e *composer.Execution, input SendOrderConfirmationInput) (any, error) {
		logger, _ := e.Container().Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
		logger.Info("Confirmation sent", "order_id", input.ID)
		return nil, nil
	})

// region: workflow
// HelloWorldWorkflow emits the custom.created event once it finishes
// successfully.
var HelloWorldWorkflow = composer.CreateWorkflow("hello-world",
	func(e *composer.Execution, input struct{}) (any, error) {
		// ...

		payload := dal.NewRecord()
		payload.Set("id", "123")
		// other data payload

		_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{
			EventName: "custom.created",
			Data:      payload,
		})
		return nil, err
	})

// endregion: workflow

// region: multiple
// emitMany emits the event three times, once for each element of Data.
func emitMany(e *composer.Execution) error {
	payloads := make([]any, 0, 3)
	for _, id := range []string{"123", "456", "789"} {
		payload := dal.NewRecord()
		payload.Set("id", id)
		payloads = append(payloads, payload)
	}

	_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{
		EventName: "custom.created",
		Data:      payloads,
	})
	return err
}

// endregion: multiple

// region: priority
// emitCritical emits an event with the CRITICAL priority.
func emitCritical(e *composer.Execution) error {
	options := dal.NewRecord()
	options.Set("priority", eventbus.EventPriority.CRITICAL)

	payload := dal.NewRecord()
	payload.Set("id", "123")

	_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{
		EventName: "payment.processed",
		Data:      payload,
		Options:   options,
	})
	return err
}

// endregion: priority
