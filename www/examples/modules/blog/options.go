package blog

// region: options
import (
	"fmt"
)

// Options are the options of the blog module, the `options` of its entry in
// medusa-config.go.
type Options struct {
	// DefaultTags are created by the module's loader.
	DefaultTags []string
	// MaxTitleLength is the longest title CreateDraft accepts.
	MaxTitleLength int
}

// ParseOptions validates the options map the module receives.
func ParseOptions(raw map[string]any) (Options, error) {
	opts := Options{MaxTitleLength: 120}

	switch v := raw["defaultTags"].(type) {
	case nil:
	case []string:
		opts.DefaultTags = v
	case []any:
		for _, tag := range v {
			s, ok := tag.(string)
			if !ok {
				return opts, fmt.Errorf("defaultTags must be a list of strings")
			}
			opts.DefaultTags = append(opts.DefaultTags, s)
		}
	default:
		return opts, fmt.Errorf("defaultTags must be a list of strings")
	}

	switch v := raw["maxTitleLength"].(type) {
	case nil:
	case int:
		opts.MaxTitleLength = v
	case float64:
		opts.MaxTitleLength = int(v)
	default:
		return opts, fmt.Errorf("maxTitleLength must be a number")
	}
	if opts.MaxTitleLength <= 0 {
		return opts, fmt.Errorf("maxTitleLength must be greater than 0")
	}
	return opts, nil
}

// endregion: options
