package main

// region: main-imports
import (
	"log"
	"os"

	cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/www/examples/links"
	"github.com/hitkey-io/medusago/www/examples/modules/scripts"
)

// endregion: main-imports

// region: main
func main() {
	// Links must be defined before the application is loaded.
	if err := links.Define(); err != nil {
		log.Fatal(err)
	}

	os.Exit(cli.Run(os.Args[1:], cli.Options{
		Config: Config,
		Scripts: map[string]typescommon.ExecScript{
			"my-script": scripts.MyScript,
			"greet":     scripts.Greet,
			"seed":      scripts.Seed,
			"test-blog": scripts.TestBlog,
		},
	}))
}

// endregion: main
