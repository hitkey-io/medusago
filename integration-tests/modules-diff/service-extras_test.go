package modulesdiff

import (
	"context"
	"testing"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	taxtypes "github.com/hitkey-io/medusago/packages/core/types/src/tax"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	auth "github.com/hitkey-io/medusago/packages/modules/auth/src"
	authservices "github.com/hitkey-io/medusago/packages/modules/auth/src/services"
	taxmodule "github.com/hitkey-io/medusago/packages/modules/tax/src"
	taxservices "github.com/hitkey-io/medusago/packages/modules/tax/src/services"
)

// Module service methods without an upstream golden (they need a custom
// provider on both sides, or secrets that cannot be recorded).

// customTaxProvider returns one 7% line per item, whatever the rates.
type customTaxProvider struct{ seen int }

func (p *customTaxProvider) GetIdentifier() string { return "custom" }

func (p *customTaxProvider) GetTaxLines(_ context.Context, itemLines []taxtypes.ItemTaxCalculationLine, shippingLines []taxtypes.ShippingTaxCalculationLine, _ *dal.Record) ([]*dal.Record, error) {
	p.seen = len(itemLines) + len(shippingLines)
	out := []*dal.Record{}
	for _, l := range itemLines {
		id, _ := l.LineItem.Get("id")
		out = append(out, rec("rate", float64(7), "code", "CUSTOM", "name", "Custom", "line_item_id", id, "provider_id", "custom"))
	}
	return out, nil
}

func TestTaxModuleUsesTheRegionProvider(t *testing.T) {
	pool := setupDatabase(t, "medusa_diff_go_tax_provider")
	ctx := context.Background()
	provider := &customTaxProvider{}
	inst, err := taxmodule.Module.Service(modulessdk.ServiceOptions{DB: pool, Providers: []modulessdk.ProviderInstance{{Identifier: "custom", ID: "custom", Instance: provider}}})
	if err != nil {
		t.Fatal(err)
	}
	svc := inst.(*taxservices.TaxModuleService)
	if _, err := svc.Internal("TaxProvider").Create(ctx, []*dal.Record{rec("id", "tp_custom_custom", "is_enabled", true)}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTaxRegions(ctx, []*dal.Record{
		rec("country_code", "fr", "provider_id", "tp_custom_custom", "default_tax_rate", rec("name", "FR", "code", "FR20", "rate", 20)),
		rec("country_code", "it", "default_tax_rate", rec("name", "IT", "code", "IT22", "rate", 22)),
	}); err != nil {
		t.Fatal(err)
	}
	items := []*dal.Record{rec("id", "item_1", "product_id", "prod_1")}

	lines, err := svc.GetTaxLines(ctx, items, rec("address", rec("country_code", "FR")))
	if err != nil {
		t.Fatal(err)
	}
	if provider.seen != 1 || len(lines) != 1 || field(lines[0], "provider_id") != "custom" || field(lines[0], "rate") != float64(7) {
		t.Fatalf("custom provider lines = %v (seen %d)", lines, provider.seen)
	}

	// a region without a provider: the provider "null" is not registered
	if _, err := svc.GetTaxLines(ctx, items, rec("address", rec("country_code", "IT"))); err == nil {
		t.Fatal("expected an error for a region without a tax provider")
	}

	if p, err := svc.GetProvider("tp_system"); err != nil || p.GetIdentifier() != "system" {
		t.Fatalf("GetProvider(tp_system) = %v, %v", p, err)
	}
	if _, err := svc.GetProvider("tp_missing"); err == nil {
		t.Fatal("expected an error for an unknown provider")
	}
}

func TestUseAuthMfaRecoveryCode(t *testing.T) {
	pool := setupDatabase(t, "medusa_diff_go_auth_recovery")
	ctx := context.Background()
	inst, err := auth.Module.Service(modulessdk.ServiceOptions{DB: pool})
	if err != nil {
		t.Fatal(err)
	}
	svc := inst.(*authservices.AuthModuleService)
	identities, err := svc.Internal("AuthIdentity").Create(ctx, []*dal.Record{rec("app_metadata", rec())})
	if err != nil {
		t.Fatal(err)
	}
	id := field(identities[0], "id")
	err = svc.UseAuthMfaRecoveryCode(ctx, id, "aaaa-bbbb-cccc-dddd")
	me, ok := common.AsMedusaError(err)
	if !ok || me.Type != common.MedusaErrorTypes.NOT_ALLOWED || me.Message != "Recovery code is invalid or already used" {
		t.Fatalf("err = %v", err)
	}
	two := 2
	codes, err := svc.GenerateAuthMfaRecoveryCodes(ctx, id, &two)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 2 {
		t.Fatalf("codes = %v", codes)
	}
	if err := svc.UseAuthMfaRecoveryCode(ctx, id, codes[0]); err != nil {
		t.Fatal(err)
	}
	// a recovery code is used once
	if err := svc.UseAuthMfaRecoveryCode(ctx, id, codes[0]); err == nil {
		t.Fatal("a used recovery code was accepted again")
	}
	left, err := svc.Internal("AuthMfaRecoveryCode").List(ctx, rec("auth_identity_id", id), utilsmodulessdk.FindConfig{})
	if err != nil || len(left) != 1 {
		t.Fatalf("recovery codes left = %d, %v", len(left), err)
	}
}
