package blog

// region: imports
import (
	"context"
	"fmt"

	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// endregion: imports

// region: loader
// SeedTags creates the default tags of the module options. It is safe to
// run many times: tags that exist are left as they are.
func SeedTags(ctx context.Context, opts typesmodulessdk.LoaderOptions) error {
	options, err := ParseOptions(opts.Options)
	if err != nil {
		return err
	}
	if len(options.DefaultTags) == 0 {
		return nil
	}

	// A loader gets the database handle, not the module service: build the
	// service of the models to use their internal services.
	tags := utilsmodulessdk.NewMedusaService(opts.DB, Models, nil).Internal("Tag")

	wanted := make([]any, len(options.DefaultTags))
	for i, value := range options.DefaultTags {
		wanted[i] = value
	}
	existing, err := tags.List(ctx, rec("value", rec("$in", wanted)), utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, tag := range existing {
		value, _ := tag.Get("value")
		have[value.(string)] = true
	}

	var missing []*dal.Record
	for _, value := range options.DefaultTags {
		if !have[value] {
			missing = append(missing, rec("value", value))
		}
	}
	if _, err := tags.Create(ctx, missing); err != nil {
		return err
	}
	opts.Logger.Info(fmt.Sprintf("blog: created %d default tags", len(missing)))
	return nil
}

// endregion: loader
