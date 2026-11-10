package joiner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func dedupeStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func joinPath(parts ...string) string { return strings.Join(parts, ".") }

// CompileQuery mirrors compileQuery (cross module join pushdown and seed
// data are handled separately).
func CompileQuery(query *RemoteJoinerQuery, serviceConfig *ServiceConfig, catalog *GraphCatalog) (*QueryPlan, error) {
	crossModuleJoins, residualFilters, residualOrder := extractCrossModuleJoins(query, serviceConfig, catalog)
	if len(crossModuleJoins) > 0 {
		query.Args = append(query.Args, Arg{Name: "__internal", Value: crossModuleJoins})
	}
	var residualHidden []ResidualHiddenProperty
	if len(residualFilters) > 0 || len(residualOrder) > 0 {
		residualHidden = consumeResiduals(query, serviceConfig, catalog, residualFilters, residualOrder)
	}
	primaryKeyArg, otherArgs, pkName := getPrimaryKeysAndOtherFilters(serviceConfig, query)

	root := &ExpandProperty{
		Property:      "",
		Parent:        "",
		ServiceConfig: serviceConfig,
		Entity:        serviceConfig.Entity,
		Fields:        query.Fields,
		Args:          otherArgs,
	}
	var shortcuts []ShortcutSpec
	parsed, err := parseProperties(root, query, serviceConfig, &shortcuts, catalog)
	if err != nil {
		return nil, err
	}
	grouped := groupSameServiceExpands(parsed)
	attachExecutionStages(parsed, grouped)

	return &QueryPlan{
		Root:          grouped.Get(BasePath),
		Expands:       grouped,
		Shortcuts:     shortcuts,
		PkName:        pkName,
		PrimaryKeyArg: primaryKeyArg,
		OtherArgs:     otherArgs,

		ResidualFilters: residualFilters,
		ResidualOrder:   residualOrder,
		ResidualHidden:  residualHidden,
	}, nil
}

func parseProperties(root *ExpandProperty, query *RemoteJoinerQuery, serviceConfig *ServiceConfig, shortcuts *[]ShortcutSpec, catalog *GraphCatalog) (*ExpandMap, error) {
	aliasRealPathMap := map[string][]string{}
	parsed := newExpandMap()
	parsed.set(BasePath, root)
	var forwardArgumentsOnPath []string

	expands := append([]ExpandSpec(nil), query.Expands...)
	for i := 0; i < len(expands); i++ {
		expand := expands[i]
		properties := strings.Split(expand.Property, ".")
		var currentPath, currentAliasPath []string
		currentServiceConfig := serviceConfig

		for _, prop := range properties {
			currentPathEntity := currentServiceConfig.Entity
			if e := parsed.Get(joinPath(append([]string{BasePath}, currentPath...)...)); e != nil && e.Entity != "" {
				currentPathEntity = e.Entity
			}
			if aliasEntry := resolveFieldAliasEntry(currentServiceConfig.FieldAlias[prop], currentPathEntity); aliasEntry != nil {
				aliasPath := joinPath(append(append([]string{BasePath}, currentPath...), prop)...)
				next, err := rewriteFieldAlias(rewriteParams{
					aliasPath:              aliasPath,
					aliasRealPathMap:       aliasRealPathMap,
					expands:                &expands,
					expand:                 expand,
					property:               prop,
					aliasEntry:             *aliasEntry,
					parsed:                 parsed,
					currentServiceConfig:   currentServiceConfig,
					currentPath:            &currentPath,
					shortcuts:              shortcuts,
					forwardArgumentsOnPath: &forwardArgumentsOnPath,
				}, catalog)
				if err != nil {
					return nil, err
				}
				currentServiceConfig = next
				currentAliasPath = append(currentAliasPath, prop)
				continue
			}

			fullPath := joinPath(append(append([]string{BasePath}, currentPath...), prop)...)
			fullAliasPath := joinPath(append(append([]string{BasePath}, currentAliasPath...), prop)...)

			entity := currentServiceConfig.Entity
			if entity != "" {
				complete := strings.Split(fullPath, ".")
				for j := 1; j < len(complete); j++ {
					if e := catalog.GetEntity(entity, complete[j]); e != "" {
						entity = e
					}
				}
			}

			relationship := catalog.GetEntityRelationship(currentServiceConfig, prop, entity)
			isCurrentProp := fullPath == BasePath+"."+expand.Property || fullAliasPath == BasePath+"."+expand.Property

			fields := []string{}
			var args []Arg
			if isCurrentProp {
				if expand.Fields != nil {
					fields = append(fields, expand.Fields...)
				}
				args = expand.Args
			}

			if relationship != nil {
				parentExpand := parsed.Get(joinPath(append([]string{BasePath}, currentPath...)...))
				if parentExpand != nil {
					injectJoinKeys(parentExpand, relationship)
					fields = append(fields, joinKeyFields(relationship, "child")...)
				}
				if relationship.Entity != "" {
					entity = relationship.Entity
				}
				currentServiceConfig = catalog.GetServiceConfig(relationship.ServiceName, "", relationship.Entity)
				if currentServiceConfig == nil {
					return nil, fmt.Errorf("Target service not found: %s", relationship.ServiceName)
				}
			}

			isAliasMapping := expand.IsAliasMapping
			if !parsed.has(fullPath) {
				parentPath := joinPath(append([]string{BasePath}, currentPath...)...)
				if real, ok := aliasRealPathMap[parentPath]; ok {
					parentPath = joinPath(real[:len(real)-1]...)
				}
				exp := &ExpandProperty{
					Property:       prop,
					ServiceConfig:  currentServiceConfig,
					Entity:         entity,
					Fields:         fields,
					IsAliasMapping: isAliasMapping,
					Parent:         parentPath,
					ParentConfig:   parsed.Get(parentPath).ServiceConfig,
				}
				if !isAliasMapping || slices.Contains(forwardArgumentsOnPath, fullPath) {
					exp.Args = args
				}
				parsed.set(fullPath, exp)
			} else {
				exp := parsed.Get(fullPath)
				if slices.Contains(forwardArgumentsOnPath, fullPath) && args != nil {
					exp.Args = append(exp.Args, args...)
				}
				exp.IsAliasMapping = exp.IsAliasMapping || isAliasMapping
				exp.Fields = dedupeStrings(append(append([]string{}, exp.Fields...), fields...))
			}
			currentPath = append(currentPath, prop)
			currentAliasPath = append(currentAliasPath, prop)
		}
	}
	return parsed, nil
}

type rewriteParams struct {
	aliasPath              string
	aliasRealPathMap       map[string][]string
	expands                *[]ExpandSpec
	expand                 ExpandSpec
	property               string
	aliasEntry             FieldAliasEntry
	parsed                 *ExpandMap
	currentServiceConfig   *ServiceConfig
	currentPath            *[]string
	shortcuts              *[]ShortcutSpec
	forwardArgumentsOnPath *[]string
}

// rewriteFieldAlias mirrors rewriteFieldAlias.
func rewriteFieldAlias(p rewriteParams, catalog *GraphCatalog) (*ServiceConfig, error) {
	currentServiceConfig := p.currentServiceConfig
	serviceConfig := currentServiceConfig
	path := p.aliasEntry.Path
	pathParts := strings.Split(path, ".")
	fullPath := append(append([]string{}, *p.currentPath...), pathParts...)

	if _, ok := p.aliasRealPathMap[p.aliasPath]; ok {
		*p.currentPath = append(*p.currentPath, pathParts...)
		existing := joinPath(append([]string{BasePath}, *p.currentPath...)...)
		return p.parsed.Get(existing).ServiceConfig, nil
	}

	parentPath := joinPath(append([]string{BasePath}, *p.currentPath...)...)
	parentExpands := p.parsed.Get(parentPath)
	parentExpands.Fields = slices.DeleteFunc(append([]string{}, parentExpands.Fields...), func(f string) bool { return f == p.property })

	for _, forPath := range p.aliasEntry.ForwardArgumentsOnPath {
		*p.forwardArgumentsOnPath = append(*p.forwardArgumentsOnPath, BasePath+"."+joinPath(append(append([]string{}, *p.currentPath...), forPath)...))
	}

	parentFieldAlias := fullPath[max(len(fullPath)-2, 0)]
	parentRelIsList := false
	if rels := serviceConfig.Relationships.Get(parentFieldAlias); len(rels) == 1 {
		parentRelIsList = rels[0].IsList
	}
	*p.shortcuts = append(*p.shortcuts, ShortcutSpec{
		Location: append([]string{}, *p.currentPath...),
		Property: p.property,
		Path:     fullPath,
		IsList:   p.aliasEntry.IsList || parentRelIsList,
	})

	curMiddlePath := append([]string{}, *p.currentPath...)
	for _, mid := range pathParts {
		curMiddlePath = append(curMiddlePath, mid)
		midProp := joinPath(curMiddlePath...)
		var existing *ExpandSpec
		for i := range *p.expands {
			if (*p.expands)[i].Property == midProp {
				existing = &(*p.expands)[i]
				break
			}
		}
		extra := ExpandSpec{Property: midProp, IsAliasMapping: existing == nil}
		if existing != nil {
			extra.Fields = existing.Fields
			extra.Args = existing.Args
		}
		if midProp == joinPath(fullPath...) {
			extra.Fields = p.expand.Fields
			extra.Args = p.expand.Args
			extra.IsAliasMapping = p.expand.IsAliasMapping || existing == nil
		}
		if slices.Contains(*p.forwardArgumentsOnPath, BasePath+"."+midProp) {
			var forwarded []Arg
			if existing != nil {
				forwarded = append(forwarded, existing.Args...)
			}
			forwarded = append(forwarded, p.expand.Args...)
			if len(forwarded) > 0 {
				extra.Args = forwarded
			}
		}
		*p.expands = append(*p.expands, extra)
	}

	var partialPath []string
	for _, partial := range pathParts {
		complete := append(append(append([]string{BasePath}, *p.currentPath...), partialPath...), partial)
		parent := joinPath(complete[:len(complete)-1]...)
		entity := serviceConfig.Entity
		if entity != "" {
			for j := 1; j < len(complete); j++ {
				if e := catalog.GetEntity(entity, complete[j]); e != "" {
					entity = e
				}
			}
		}
		relationship := catalog.GetEntityRelationship(currentServiceConfig, partial, entity)
		if relationship != nil {
			if relationship.Entity != "" {
				entity = relationship.Entity
			}
			currentServiceConfig = catalog.GetServiceConfig(relationship.ServiceName, "", relationship.Entity)
			if currentServiceConfig == nil {
				return nil, fmt.Errorf("Target service not found: %s", relationship.ServiceName)
			}
		}
		partialPath = append(partialPath, partial)
		p.parsed.set(joinPath(complete...), &ExpandProperty{
			Property:      partial,
			ServiceConfig: currentServiceConfig,
			Entity:        entity,
			Parent:        parent,
			ParentConfig:  p.parsed.Get(parent).ServiceConfig,
		})
	}

	*p.currentPath = append(*p.currentPath, pathParts...)
	p.aliasRealPathMap[p.aliasPath] = append([]string{BasePath}, *p.currentPath...)
	return currentServiceConfig, nil
}

// joinKeyFields mirrors joinKeyFields.
func joinKeyFields(rel *Relationship, side string) []string {
	lastFK := func() string {
		parts := strings.Split(rel.ForeignKey, ".")
		return parts[len(parts)-1]
	}
	var field string
	if side == "parent" {
		if rel.Inverse {
			field = rel.PrimaryKey
		} else {
			field = lastFK()
		}
	} else {
		if rel.Inverse {
			field = lastFK()
		} else {
			field = rel.PrimaryKey
		}
	}
	return strings.Split(field, ",")
}

// injectJoinKeys mirrors injectJoinKeys.
func injectJoinKeys(parent *ExpandProperty, rel *Relationship) {
	fields := append(append([]string{}, parent.Fields...), joinKeyFields(rel, "parent")...)
	fields = slices.DeleteFunc(fields, func(f string) bool { return f == rel.Alias })
	parent.Fields = dedupeStrings(fields)
}

// groupSameServiceExpands mirrors groupSameServiceExpands: expands served
// by the same service as their parent are nested into it.
func groupSameServiceExpands(parsed *ExpandMap) *ExpandMap {
	merged := parsed.clone()
	mergedPaths := map[string]*ExpandProperty{}

	for _, path := range append([]string{}, parsed.keys...) {
		expand := merged.Get(path)
		if expand == nil {
			continue
		}
		service := expand.ServiceConfig.ServiceName
		parentPath := expand.Parent
		for parentPath != "" {
			parentExpand := merged.Get(parentPath)
			if parentExpand == nil {
				parentExpand = mergedPaths[parentPath]
			}
			if parentExpand == nil || parentExpand.ServiceConfig.ServiceName != service {
				break
			}
			nestedKeys := strings.Split(path, ".")[len(strings.Split(parentPath, ".")):]
			target := parentExpand
			for _, key := range nestedKeys {
				if target.Expands == nil {
					target.Expands = &NestedExpands{Values: map[string]*ExpandProperty{}}
				}
				target = target.Expands.getOrCreate(key)
			}
			target.Fields = dedupeStrings(append(append([]string{}, target.Fields...), expand.Fields...))
			if len(expand.Args) > 0 {
				target.Args = append(target.Args, expand.Args...)
			}
			merged.delete(path)
			mergedPaths[path] = expand
			parentPath = parentExpand.Parent
		}
	}
	return merged
}

// attachExecutionStages mirrors attachExecutionStages.
func attachExecutionStages(full, grouped *ExpandMap) {
	root := grouped.Get(BasePath)
	stages := [][]ExecutionStage{{{Service: root.ServiceConfig.ServiceName, Entity: root.Entity, Depth: 0}}}

	sequenceLength := func(path string) int {
		n := 0
		for current := path; current != "" && current != BasePath; {
			e := full.Get(current)
			if e == nil {
				break
			}
			n++
			current = e.Parent
		}
		return n
	}

	type depthGroup struct {
		services []string
		paths    map[string][]string
	}
	byDepth := map[int]*depthGroup{}
	maxDepth := 0
	for _, path := range grouped.keys {
		if path == BasePath {
			continue
		}
		depth := sequenceLength(path)
		service := grouped.Get(path).ServiceConfig.ServiceName
		g, ok := byDepth[depth]
		if !ok {
			g = &depthGroup{paths: map[string][]string{}}
			byDepth[depth] = g
		}
		if _, ok := g.paths[service]; !ok {
			g.services = append(g.services, service)
		}
		g.paths[service] = append(g.paths[service], path)
		maxDepth = max(maxDepth, depth)
	}
	for depth := 1; depth <= maxDepth; depth++ {
		g, ok := byDepth[depth]
		if !ok {
			continue
		}
		var stage []ExecutionStage
		for _, service := range g.services {
			stage = append(stage, ExecutionStage{Service: service, Paths: g.paths[service], Depth: depth})
		}
		if len(stage) > 0 {
			stages = append(stages, stage)
		}
	}
	root.ExecutionStages = stages
}

// getPrimaryKeysAndOtherFilters mirrors getPrimaryKeysAndOtherFilters: a
// filter on a primary key becomes the id list of the root fetch.
func getPrimaryKeysAndOtherFilters(serviceConfig *ServiceConfig, query *RemoteJoinerQuery) (*Arg, []Arg, string) {
	pkName := ""
	if len(serviceConfig.PrimaryKeys) > 0 {
		pkName = serviceConfig.PrimaryKeys[0]
	}
	var primaryKeyArg *Arg
	var otherArgs []Arg
	for i, arg := range query.Args {
		if slices.Contains(serviceConfig.PrimaryKeys, arg.Name) {
			if primaryKeyArg == nil {
				a := query.Args[i]
				primaryKeyArg = &a
				pkName = arg.Name
			}
			continue
		}
		otherArgs = append(otherArgs, arg)
	}

	if primaryKeyArg == nil {
		for i, arg := range otherArgs {
			if arg.Name != "filters" {
				continue
			}
			rec := asRecord(arg.Value)
			if rec == nil {
				continue
			}
			for _, key := range rec.Keys() {
				if !slices.Contains(serviceConfig.PrimaryKeys, key) {
					continue
				}
				v, _ := rec.Get(key)
				pkName = key
				primaryKeyArg = &Arg{Name: key, Value: v}
				// The filter is moved out of the filters.
				clone := cloneRecordWithout(rec, key)
				otherArgs[i] = Arg{Name: "filters", Value: clone}
				break
			}
			break
		}
	}
	if len(otherArgs) == 0 {
		otherArgs = nil
	}
	return primaryKeyArg, otherArgs, pkName
}

func asRecord(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

func cloneRecordWithout(r *dal.Record, without string) *dal.Record {
	out := dal.NewRecord()
	for _, k := range r.Keys() {
		if k == without {
			continue
		}
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}
