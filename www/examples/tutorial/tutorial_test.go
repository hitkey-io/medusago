package main

import (
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/api"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/api/admin/brands"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/api/admin/products"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/links"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/modules/brand"
)

func TestBrandValidator(t *testing.T) {
	cases := map[string]string{
		`{}`:                 "Invalid request: Field 'name' is required",
		`{"name": 1}`:        "Invalid request: Expected type: 'string' for field 'name', got: '1'",
		`{"name":"a","x":1}`: "Invalid request: Unrecognized fields: 'x'",
		`{"name":"Acme"}`:    "",
	}
	for body, want := range cases {
		v, _ := dal.DecodeOrderedJSON([]byte(body))
		_, err := zod.ZodValidator(brands.PostAdminCreateBrand, v)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != want {
			t.Errorf("%s: %q, want %q", body, got, want)
		}
	}
}

func TestAdditionalDataMiddleware(t *testing.T) {
	run := func(body string) error {
		v, _ := dal.DecodeOrderedJSON([]byte(body))
		req := medusahttp.NewMedusaRequest(httptest.NewRequest(nethttp.MethodPost, "/admin/products", nil))
		req.ValidatedBody = v
		called := false
		err := products.ValidateAdditionalData(req, nil, func(error) { called = true })
		if err == nil && !called {
			t.Error("next not called")
		}
		return err
	}
	if err := run(`{"title":"a"}`); err != nil {
		t.Error(err)
	}
	if err := run(`{"title":"a","additional_data":{"brand_id":"b1"}}`); err != nil {
		t.Error(err)
	}
	if err := run(`{"title":"a","additional_data":{"brand_id":1}}`); err == nil {
		t.Error("expected an error")
	}
	if err := run(`{"title":"a","additional_data":{"other":1}}`); err == nil {
		t.Error("expected an error")
	}
}

func TestLinkAndRegistration(t *testing.T) {
	if err := links.Define(); err != nil {
		t.Fatal(err)
	}
	cfgs, err := utilsmodulessdk.ResolveCustomLinks([]*joiner.ModuleJoinerConfig{modules.Product.JoinerConfig, brand.Module.JoinerConfig})
	if err != nil || len(cfgs) != 1 || cfgs[0].ServiceName != "ProductProductBrandBrandLink" {
		t.Fatalf("%v %v", cfgs, err)
	}
	if len(api.Routes) != 1 || len(api.Middlewares) != 2 {
		t.Fatalf("%d routes, %d middlewares", len(api.Routes), len(api.Middlewares))
	}
	cfg := medusaConfig()
	if len(cfg.Modules) != 1 || cfg.Modules[0].Module == nil {
		t.Fatal("module not configured")
	}
}
