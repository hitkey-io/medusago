package longrunning

import (
	"context"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: subscribe
// Start runs the workflow and listens to the changes in the status of its
// execution. It returns the transaction ID of the execution, which you
// use to set the status of its async steps.
func Start(ctx context.Context, container composer.Container, done func(result any)) (string, error) {
	// Notify delivers the events of the run to the subscribers
	_, info, err := MyWorkflow.RunWithInfo(ctx, container, nil, composer.RunOptions{Notify: true})
	if err != nil {
		return "", err
	}

	const subscriberID = "hello-world-subscriber"
	composer.Subscribe("hello-world", info.TransactionID, subscriberID, func(event composer.WorkflowEvent) {
		switch event.EventType {
		case "onFinish":
			// the result is the output of the workflow as a JSON-like value
			done(event.Result)

			// unsubscribe
			composer.Unsubscribe("hello-world", info.TransactionID, subscriberID)
		case "onStepFailure":
			slog.Error("Workflow failed", "transaction_id", event.TransactionID, "errors", event.Errors)
		}
	})

	return info.TransactionID, nil
}

// endregion: subscribe
