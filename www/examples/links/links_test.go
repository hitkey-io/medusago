package links

import (
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	"github.com/hitkey-io/medusago/www/examples/modules/blog"
)

func TestLinks(t *testing.T) {
	utilsmodulessdk.ClearCustomLinks()
	t.Cleanup(utilsmodulessdk.ClearCustomLinks)
	for _, define := range []func() error{
		DefinePostProduct, DefinePostVariant, DefineCustomerComment,
		DefinePostSalesChannel, DefinePostAuthor, DefineAuthorPosts,
	} {
		if err := define(); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefineConfig(config.InputConfig{Modules: []config.ModuleConfig{{Module: &blog.Module.ModuleExports}}})
	_, link, qc, err := loaders.NewQueryAndLink(nil, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range qc.Links {
		if l.Definition.IsReadOnlyLink {
			t.Logf("read-only: %+v", l.Definition.Extends[0].Relationship)
			continue
		}
		if l.Definition.DatabaseConfig != nil || true {
			t.Logf("link %s alias=%v rels=%s/%s -> %s/%s", l.Definition.ServiceName, l.Definition.Alias, l.Definition.Relationships[0].ServiceName, l.Definition.Relationships[0].ForeignKey, l.Definition.Relationships[1].ServiceName, l.Definition.Relationships[1].ForeignKey)
		}
	}
	for _, k := range [][4]string{
		{"blog", "post_id", "product", "product_id"},
		{"blog", "post_id", "product", "variant_id"},
		{"blog", "post_id", "sales_channel", "sales_channel_id"},
		{"customer", "customer_id", "blog", "comment_id"},
	} {
		t.Logf("%v -> %v", k, link.GetLinkModule(k[0], k[1], k[2], k[3]) != nil)
	}
}
