// Command api-routes prints every HTTP route (method, path, feature flag)
// that a MedusaGo server registers by default (core routes plus the loyalty
// plugin). www/scripts/generate-api-reference.mjs uses it to publish only the
// operations the Go server implements.
package main

import (
	"fmt"
	"sort"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api"
	loyalty "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api"
)

func main() {
	var lines []string
	add := func(src string, files []medusahttp.RouteFile) {
		for _, f := range files {
			for _, m := range []struct {
				name string
				h    medusahttp.RouteHandler
			}{{"GET", f.Route.GET}, {"POST", f.Route.POST}, {"PUT", f.Route.PUT}, {"PATCH", f.Route.PATCH}, {"DELETE", f.Route.DELETE}} {
				if m.h != nil {
					lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s", m.name, f.Path, f.FeatureFlag, src))
				}
			}
		}
	}
	add("core", api.Routes())
	add("loyalty", loyalty.Routes())
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Println(l)
	}
}
