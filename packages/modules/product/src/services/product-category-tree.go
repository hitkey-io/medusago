package services

import (
	"context"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// listCategories mirrors ProductCategoryService.list/listAndCount and
// ProductCategoryRepository.find/findAndCount with the
// include_descendants_tree / include_ancestors_tree filters.
func (s *ProductModuleService) listCategories(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig, withCount bool) ([]*dal.Record, int, error) {
	descendants, ancestors := false, false
	if filters != nil && (filters.Has("include_descendants_tree") || filters.Has("include_ancestors_tree")) {
		d, _ := filters.Get("include_descendants_tree")
		a, _ := filters.Get("include_ancestors_tree")
		descendants, ancestors = d == true, a == true
		out := dal.NewRecord()
		for _, k := range filters.Keys() {
			if k == "include_descendants_tree" || k == "include_ancestors_tree" {
				continue
			}
			v, _ := filters.Get(k)
			out.Set(k, v)
		}
		filters = out
	}
	if !descendants && !ancestors {
		if withCount {
			return s.MedusaService.ListAndCountBySuffix(ctx, "ProductCategories", filters, config)
		}
		rows, err := s.MedusaService.ListBySuffix(ctx, "ProductCategories", filters, config)
		return rows, 0, err
	}
	// buildFindOptions: tree building needs mpath and parent_category_id
	if len(config.Select) > 0 {
		config.Select = appendMissing(config.Select, "mpath", "parent_category_id")
	}
	if ancestors {
		config.Relations = appendMissing(config.Relations, "parent_category")
	}
	if descendants {
		config.Relations = appendMissing(config.Relations, "category_children")
	}
	var rows []*dal.Record
	var count int
	var err error
	if withCount {
		rows, count, err = s.MedusaService.ListAndCountBySuffix(ctx, "ProductCategories", filters, config)
	} else {
		rows, err = s.MedusaService.ListBySuffix(ctx, "ProductCategories", filters, config)
	}
	if err != nil {
		return nil, 0, err
	}
	tree, err := s.buildCategoriesTree(ctx, descendants, ancestors, rows, filters, config)
	if err != nil {
		return nil, 0, err
	}
	return sortCategoriesByRank(tree), count, nil
}

func appendMissing(list []string, items ...string) []string {
	out := append([]string{}, list...)
	for _, it := range items {
		found := false
		for _, x := range out {
			if x == it {
				found = true
				break
			}
		}
		if !found {
			out = append(out, it)
		}
	}
	return out
}

func rankOf(r *dal.Record) float64 {
	v, _ := r.Get("rank")
	f, _ := numberOf(v)
	return f
}

func sortCategoriesByRank(cats []*dal.Record) []*dal.Record {
	sort.SliceStable(cats, func(i, j int) bool { return rankOf(cats[i]) < rankOf(cats[j]) })
	for _, c := range cats {
		if ch, ok := c.Get("category_children"); ok {
			if list := records(ch); list != nil {
				sorted := sortCategoriesByRank(list)
				out := make([]any, len(sorted))
				for i, x := range sorted {
					out[i] = x
				}
				c.Set("category_children", out)
			}
		}
	}
	return cats
}

// buildCategoriesTree mirrors buildProductCategoriesWithTree.
func (s *ProductModuleService) buildCategoriesTree(ctx context.Context, descendants, ancestors bool, cats []*dal.Record, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	shouldPopulateParent := false
	var relations []string
	for _, r := range config.Relations {
		if r == "parent_category" {
			shouldPopulateParent = true
			if ancestors {
				continue
			}
		}
		if r == "category_children" && descendants {
			continue
		}
		relations = append(relations, r)
	}
	var or []any
	parentMpaths := []any{}
	seen := map[string]bool{}
	for _, c := range cats {
		mpath := strOf(c, "mpath")
		if descendants {
			like := dal.NewRecord()
			like.Set("$like", mpath+"%")
			r := dal.NewRecord()
			r.Set("mpath", like)
			or = append(or, r)
		}
		if ancestors && mpath != "" {
			parent := ""
			for _, part := range strings.Split(mpath, ".") {
				if !seen[parent+part] {
					seen[parent+part] = true
					parentMpaths = append(parentMpaths, parent+part)
				}
				parent += part + "."
			}
		}
	}
	r := dal.NewRecord()
	r.Set("mpath", parentMpaths)
	or = append(or, r)
	where := dal.NewRecord()
	if filters != nil {
		for _, k := range filters.Keys() {
			switch k {
			case "id", "handle", "mpath", "parent_category_id":
				continue
			}
			v, _ := filters.Get(k)
			where.Set(k, v)
		}
	}
	where.Set("$or", or)
	treeConfig := config
	treeConfig.Relations = relations
	// the tree query loads every scalar (upstream's fields projection does
	// not apply to it)
	treeConfig.Select = nil
	treeConfig.Take = nil
	zero := 0
	treeConfig.Skip = &zero
	inTree, err := s.MedusaService.ListBySuffix(ctx, "ProductCategories", where, treeConfig)
	if err != nil {
		return nil, err
	}
	byID := map[string]*dal.Record{}
	for _, c := range inTree {
		byID[strOf(c, "id")] = c
	}
	if ancestors {
		for _, c := range inTree {
			pid := strOf(c, "parent_category_id")
			if pid == "" {
				continue
			}
			if p := byID[pid]; p != nil {
				c.Set("parent_category", p)
				c.Set("parent_category_id", strOf(p, "id"))
			} else {
				c.Set("parent_category", nil)
				c.Delete("parent_category")
				c.Set("parent_category_id", nil)
				c.Delete("parent_category_id")
			}
		}
	}
	var populate func(c *dal.Record, level int) *dal.Record
	populate = func(c *dal.Record, level int) *dal.Record {
		if c == nil {
			return nil
		}
		if descendants {
			var children []any
			for _, child := range inTree {
				if strOf(child, "parent_category_id") == strOf(c, "id") {
					children = append(children, populate(byID[strOf(child, "id")], level+1))
				}
			}
			if children == nil {
				children = []any{}
			}
			c.Set("category_children", children)
		}
		if level == 0 {
			if !ancestors && !shouldPopulateParent {
				c.Delete("parent_category")
			} else if p, ok := c.Get("parent_category"); ok && p == nil {
				// a null parent is not serialized
				c.Delete("parent_category")
			}
			return c
		}
		if ancestors {
			c.Delete("category_children")
		}
		if descendants {
			c.Delete("parent_category")
		}
		return c
	}
	out := make([]*dal.Record, 0, len(cats))
	for _, c := range cats {
		out = append(out, populate(byID[strOf(c, "id")], 0))
	}
	return out, nil
}
