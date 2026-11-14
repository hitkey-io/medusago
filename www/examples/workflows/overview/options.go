package overview

import "github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"

// region: options
// retriedStep is retried twice if it fails.
var retriedStep = step1.Config("", composer.StepOptions{MaxRetries: 2})

// endregion: options
