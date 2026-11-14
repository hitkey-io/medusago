package configexamples

// region: log-script
import (
	"context"
	"log/slog"

	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// LogMessages is a script for `go run . exec log-messages`.
func LogMessages(ctx context.Context, args typescommon.ExecArgs) error {
	logger, ok := args.Container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	if !ok {
		return nil
	}

	logger.Info("I'm using the logger!")
	logger.Warn("Something looks off", "order_id", "order_123")
	logger.Error("Something failed", "error", "payment declined")
	logger.Debug("Only shown with LOG_LEVEL=debug")

	return nil
}

// endregion: log-script
