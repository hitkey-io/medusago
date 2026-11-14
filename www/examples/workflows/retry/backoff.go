package retry

import (
	"errors"
	"time"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: backoff
// captureStep waits before it tries again: a retry executes the step
// immediately, so the step waits for as long as it needs.
var captureStep = composer.CreateStep("capture-payment",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		if ctx.Attempt > 1 {
			// 2 seconds before the second attempt, 4 before the third one
			time.Sleep(time.Duration(ctx.Attempt) * time.Second)
		}

		// TODO capture the payment
		return composer.StepResponse[string]{}, errors.New("the payment provider is unavailable")
	}, nil).Config("", composer.StepOptions{MaxRetries: 2})

// endregion: backoff
