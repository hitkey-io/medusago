package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	queryjoiner "github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	typesjoiner "github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	"github.com/hitkey-io/medusago/www/examples/links"
	"github.com/hitkey-io/medusago/www/examples/modules/blog"
)

// recorder stands for a module service: it records the calls Query makes.
type recorder struct {
	mu    sync.Mutex
	calls *[]string
	name  string
	cfg   *typesjoiner.ModuleJoinerConfig
}

func (r *recorder) rows(filters *dal.Record) []*dal.Record {
	if r.cfg.IsLink {
		var out []*dal.Record
		for i, rel := range r.cfg.Relationships {
			v, ok := filters.Get(rel.ForeignKey)
			ids, _ := v.([]any)
			if !ok || len(ids) == 0 {
				continue
			}
			other := r.cfg.Relationships[1-i].ForeignKey
			for _, id := range ids {
				row := dal.NewRecord()
				row.Set("id", "link_1")
				row.Set(rel.ForeignKey, id)
				row.Set(other, strings.TrimSuffix(other, "_id")+"_1")
				out = append(out, row)
			}
		}
		return out
	}
	var ids []any
	if v, ok := filters.Get("id"); ok {
		switch t := v.(type) {
		case []any:
			ids = t
		case string:
			ids = []any{t}
		}
	}
	if len(ids) == 0 {
		ids = []any{r.name + "_1"}
	}
	if len(ids) == 1 && ids[0] == "missing" {
		return nil
	}
	var out []*dal.Record
	for _, id := range ids {
		row := dal.NewRecord()
		row.Set("id", id)
		row.Set("author_id", "user_1")
		row.Set("title", "A post")
		out = append(out, row)
	}
	return out
}

func (r *recorder) log(kind, suffix string, filters *dal.Record, cfg utilsmodulessdk.FindConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, _ := json.Marshal(filters)
	take, skip := "-", "-"
	if cfg.Take != nil {
		take = fmt.Sprint(*cfg.Take)
	}
	if cfg.Skip != nil {
		skip = fmt.Sprint(*cfg.Skip)
	}
	*r.calls = append(*r.calls, fmt.Sprintf("%s.%s%s filters=%s select=%v relations=%v order=%v take=%s skip=%s deleted=%v", r.name, kind, suffix, f, cfg.Select, cfg.Relations, cfg.Order, take, skip, cfg.WithDeleted))
}

func (r *recorder) List(_ context.Context, suffix string, filters *dal.Record, cfg utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	r.log("list", suffix, filters, cfg)
	return r.rows(filters), nil
}

func (r *recorder) ListAndCount(_ context.Context, suffix string, filters *dal.Record, cfg utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	r.log("listAndCount", suffix, filters, cfg)
	rows := r.rows(filters)
	return rows, len(rows), nil
}

type resolver map[string]any

func (r resolver) Resolve(key string) any { return r[key] }

func setup(t *testing.T, defs ...func() error) (*query.Query, *[]string) {
	utilsmodulessdk.ClearCustomLinks()
	t.Cleanup(utilsmodulessdk.ClearCustomLinks)
	for _, define := range defs {
		if err := define(); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefineConfig(config.InputConfig{Modules: []config.ModuleConfig{{Module: &blog.Module.ModuleExports}}})
	qc, err := loaders.BuildQueryConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var calls []string
	svcs := map[string]query.ModuleService{}
	for _, c := range qc.JoinerConfigs {
		if c.ServiceName != "" {
			svcs[c.ServiceName] = &recorder{calls: &calls, name: c.ServiceName, cfg: c}
		}
	}
	q, err := query.New(qc.JoinerConfigs, queryjoiner.RelationMap(qc.RelationMap), svcs)
	if err != nil {
		t.Fatal(err)
	}
	return q, &calls
}

func TestExamples(t *testing.T) {
	q, callsPtr := setup(t, links.DefinePostProduct, links.DefinePostVariant, links.DefineCustomerComment,
		links.DefinePostSalesChannel, links.DefinePostAuthor, links.DefineAuthorPosts)
	c := resolver{common.ContainerRegistrationKeys.QUERY: q}
	ctx := context.Background()

	run := func(name string, fn func() error) {
		*callsPtr = nil
		err := fn()
		t.Logf("%s: err=%v\n  %s", name, err, strings.Join(*callsPtr, "\n  "))
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	run("PostTitles", func() error { _, err := PostTitles(ctx, c); return err })
	run("PostWithAllFields", func() error { _, err := PostWithAllFields(ctx, c, "post_1"); return err })
	run("PostsWithLinkedRecords", func() error {
		d, err := PostsWithLinkedRecords(ctx, c)
		b, _ := json.Marshal(d)
		t.Logf("%s", b)
		return err
	})
	run("PublishedPosts", func() error { _, err := PublishedPosts(ctx, c, "user_1"); return err })
	run("PostsByCommenter", func() error { _, err := PostsByCommenter(ctx, c); return err })
	run("PostsOfProduct", func() error { _, err := PostsOfProduct(ctx, c, "shirt"); return err })
	run("PostsPage", func() error { p, err := PostsPage(ctx, c, 0, 10); t.Logf("page %+v", p); return err })
	run("CountPosts", func() error { n, err := CountPosts(ctx, c); t.Logf("count %d", n); return err })
	run("DeletedPosts", func() error { _, err := DeletedPosts(ctx, c); return err })
	run("PostsWithLinkRows", func() error {
		d, err := links.PostsWithLinkRows(ctx, c)
		b, _ := json.Marshal(d)
		t.Logf("%s", b)
		return err
	})
	run("PinnedPostLinks", func() error {
		d, err := links.PinnedPostLinks(ctx, c)
		b, _ := json.Marshal(d)
		t.Logf("%s", b)
		return err
	})
	run("PostsWithAuthors", func() error { _, err := PostsWithAuthors(ctx, c); return err })
	for _, f := range filterOperators() {
		b, _ := json.Marshal(f)
		*callsPtr = nil
		_, err := q.Graph(ctx, query.GraphInput{Entity: "post", Fields: []string{"id"}, Filters: f}, queryjoiner.Options{})
		t.Logf("filter %s err=%v\n  %s", b, err, strings.Join(*callsPtr, "\n  "))
		if err != nil {
			t.Error(err)
		}
	}
	run("RetrievePost", func() error {
		_, err := RetrievePost(ctx, c, "missing")
		if err == nil {
			t.Error("expected not found")
		}
		t.Logf("not found: %v", err)
		return nil
	})
}

func TestAlias(t *testing.T) {
	q, callsPtr := setup(t, links.DefinePostSalesChannelAlias)
	c := resolver{common.ContainerRegistrationKeys.QUERY: q}
	res, err := q.Graph(context.Background(), query.GraphInput{Entity: "sales_channel", Fields: []string{"id", "articles.*"}}, queryjoiner.Options{})
	b, _ := json.Marshal(res)
	t.Logf("%v %s\n%s", err, b, strings.Join(*callsPtr, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	_ = c
	res, err = q.Graph(context.Background(), query.GraphInput{Entity: "article_sales_channel", Fields: []string{"*"}}, queryjoiner.Options{})
	t.Logf("entry point: %v", err)
}

func TestRelations(t *testing.T) {
	q, callsPtr := setup(t)
	for _, fields := range [][]string{{"id", "tags.value"}, {"id", "comments.*", "tags.*"}} {
		*callsPtr = nil
		res, err := q.Graph(context.Background(), query.GraphInput{Entity: "post", Fields: fields}, queryjoiner.Options{})
		b, _ := json.Marshal(res)
		t.Logf("%v: %v %s\n%s", fields, err, b, strings.Join(*callsPtr, "\n"))
		if err != nil {
			t.Error(err)
		}
	}
	for _, entity := range []string{"comment", "tag"} {
		_, err := q.Graph(context.Background(), query.GraphInput{Entity: entity, Fields: []string{"id", "post.*", "posts.*"}}, queryjoiner.Options{})
		t.Logf("%s: %v", entity, err)
	}
}

func TestInverseReadOnly(t *testing.T) {
	q, callsPtr := setup(t, links.DefineAuthorPosts)
	for _, field := range []string{"posts.*", "post.*"} {
		*callsPtr = nil
		res, err := q.Graph(context.Background(), query.GraphInput{Entity: "user", Fields: []string{"id", field}}, queryjoiner.Options{})
		b, _ := json.Marshal(res)
		t.Logf("%s: %v %s\n%s", field, err, b, strings.Join(*callsPtr, "\n"))
	}
}
