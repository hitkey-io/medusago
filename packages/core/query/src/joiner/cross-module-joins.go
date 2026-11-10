package joiner

import (
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// This file mirrors cross-module-joins/{types,build-chain,spec-registry,
// rewrite-filters,index}.ts (one Go package: the upstream sub directory
// would import the joiner types it is imported by).

type hop struct {
	kind          string // "relationship" | "internal"
	property      string
	serviceConfig *ServiceConfig
	entity        string
	relationship  *Relationship
	internal      *InternalRelation
	internalTrail []string
}

type chainLevel struct {
	realPathKey string
	entity      string
	spec        *dal.CrossModuleJoinSpec
}

type chainResolution struct {
	outcome string // "native" | "residual" | "pushable"
	levels  []chainLevel
}

type pushdownCandidate struct {
	levels  []chainLevel
	filters *dal.Record
}

// specRegistry keeps the specs by real path key in insertion order.
type specRegistry struct {
	keys  []string
	specs map[string]*dal.CrossModuleJoinSpec
}

func newSpecRegistry() *specRegistry {
	return &specRegistry{specs: map[string]*dal.CrossModuleJoinSpec{}}
}

// ResidualCrossModuleFilter mirrors ResidualCrossModuleFilter.
type ResidualCrossModuleFilter struct {
	Path    string
	Filters *dal.Record
}

type chainPosition struct {
	correlateKey string
	currentTable string
	serviceName  string
}

type chainStep struct {
	spec         *dal.CrossModuleJoinSpec
	entity       string
	properties   []string
	serviceName  string
	consumedHops int
}

// resolveChain mirrors resolveChain.
func resolveChain(rootConfig *ServiceConfig, pathSegments []string, catalog *GraphCatalog) chainResolution {
	hops := resolvePathHops(rootConfig, pathSegments, catalog)
	if hops == nil || !crossesModules(hops, rootConfig) {
		return chainResolution{outcome: "native"}
	}
	levels := buildChainLevels(hops, rootConfig, catalog)
	if levels == nil {
		return chainResolution{outcome: "residual"}
	}
	return chainResolution{outcome: "pushable", levels: levels}
}

// resolvePathHops mirrors resolvePathHops.
func resolvePathHops(rootConfig *ServiceConfig, pathSegments []string, catalog *GraphCatalog) []hop {
	var hops []hop
	currentConfig := rootConfig
	entity := rootConfig.Entity
	var internalTrail []string

	for _, segment := range pathSegments {
		realSegments := []string{segment}
		if entry := resolveFieldAliasEntry(currentConfig.FieldAlias[segment], entity); entry != nil {
			realSegments = strings.Split(entry.Path, ".")
		}
		for _, realSegment := range realSegments {
			var internalRelation *InternalRelation
			if md := catalog.GetAliasMetadata(entity); md != nil {
				if rel, ok := md.Relations[realSegment]; ok {
					r := rel
					internalRelation = &r
				}
			}
			if internalRelation != nil {
				if entity != "" {
					if schemaEntity := catalog.GetEntity(entity, realSegment); schemaEntity != "" && schemaEntity != internalRelation.Entity {
						return nil
					}
				}
				hops = append(hops, hop{kind: "internal", property: realSegment, serviceConfig: currentConfig, entity: internalRelation.Entity,
					internal: internalRelation, internalTrail: append([]string{}, internalTrail...)})
				internalTrail = append(internalTrail, realSegment)
				entity = internalRelation.Entity
				continue
			}

			hopEntity := entity
			if entity != "" {
				if e := catalog.GetEntity(entity, realSegment); e != "" {
					hopEntity = e
				}
			}
			relationship := catalog.GetEntityRelationship(currentConfig, realSegment, hopEntity)
			if relationship == nil {
				return nil
			}
			if relationship.Entity != "" {
				hopEntity = relationship.Entity
			}
			nextConfig := catalog.GetServiceConfig(relationship.ServiceName, "", relationship.Entity)
			if nextConfig == nil {
				return nil
			}
			hops = append(hops, hop{kind: "relationship", property: realSegment, relationship: relationship, serviceConfig: nextConfig,
				entity: hopEntity, internalTrail: append([]string{}, internalTrail...)})
			internalTrail = nil
			currentConfig = nextConfig
			entity = hopEntity
		}
	}
	return hops
}

func crossesModules(hops []hop, rootConfig *ServiceConfig) bool {
	for _, h := range hops {
		if h.serviceConfig.ServiceName != rootConfig.ServiceName {
			return true
		}
	}
	return false
}

// buildChainLevels mirrors buildChainLevels.
func buildChainLevels(hops []hop, rootConfig *ServiceConfig, catalog *GraphCatalog) []chainLevel {
	if len(hops) == 0 {
		return nil
	}
	var levels []chainLevel
	var realPath []string
	position := &chainPosition{correlateKey: "id", serviceName: rootConfig.ServiceName}
	if md := catalog.GetAliasMetadata(rootConfig.Entity); md != nil {
		position.currentTable = md.TableName
	}
	parentTable := ""
	for i := 0; i < len(hops); {
		h := hops[i]
		var next *hop
		if i+1 < len(hops) {
			next = &hops[i+1]
		}
		var step *chainStep
		switch {
		case h.kind == "internal":
			step = buildInternalRelationStep(h, next, position, rootConfig, catalog)
		case h.serviceConfig.IsLink:
			step = buildLinkModuleStep(h, next, position, rootConfig, catalog)
		default:
			step = buildReadOnlyLinkStep(h, position, rootConfig, catalog)
		}
		if step == nil {
			return nil
		}
		realPath = append(realPath, step.properties...)
		spec := *step.spec
		if parentTable != "" {
			spec.Parent = parentTable
		}
		levels = append(levels, chainLevel{realPathKey: strings.Join(realPath, "."), entity: step.entity, spec: &spec})
		parentTable = step.spec.Target.Table
		position.correlateKey = step.spec.Target.PrimaryKey
		if position.correlateKey == "" {
			position.correlateKey = "id"
		}
		position.currentTable = parentTable
		position.serviceName = step.serviceName
		i += step.consumedHops
	}
	return levels
}

func isSimpleKey(key string) bool { return !strings.Contains(key, ",") && !strings.Contains(key, ".") }

func targetOf(md *AliasMetadata) dal.CrossModuleJoinTarget {
	return dal.CrossModuleJoinTarget{Table: md.TableName, Schema: md.Schema, PrimaryKey: "id"}
}

func buildInternalRelationStep(h hop, next *hop, position *chainPosition, rootConfig *ServiceConfig, catalog *GraphCatalog) *chainStep {
	relation := h.internal
	md := catalog.GetAliasMetadata(h.entity)
	if md == nil || md.TableName == "" || !isSimpleKey(relation.ForeignKey) {
		return nil
	}
	if relation.ForeignKeyOwner == "target" {
		if position.correlateKey != "id" {
			return nil
		}
		if next != nil && next.kind == "relationship" && !next.serviceConfig.IsLink {
			fk := resolveFkHopParts(*next, position, rootConfig, catalog)
			if fk == nil {
				return nil
			}
			if fk.direction == "forward" {
				return &chainStep{
					spec:   &dal.CrossModuleJoinSpec{Link: dal.CrossModuleJoinLink{Table: md.TableName, SourceKey: relation.ForeignKey, TargetKey: fk.column}, Target: fk.target},
					entity: next.entity, properties: []string{h.property, next.property}, serviceName: next.serviceConfig.ServiceName, consumedHops: 2,
				}
			}
		}
		return &chainStep{
			spec:   &dal.CrossModuleJoinSpec{Link: dal.CrossModuleJoinLink{Table: md.TableName, SourceKey: relation.ForeignKey, TargetKey: "id"}, Target: targetOf(md)},
			entity: h.entity, properties: []string{h.property}, serviceName: position.serviceName, consumedHops: 1,
		}
	}
	if position.currentTable == "" {
		return nil
	}
	return &chainStep{
		spec:   &dal.CrossModuleJoinSpec{Link: dal.CrossModuleJoinLink{Table: position.currentTable, SourceKey: position.correlateKey, TargetKey: relation.ForeignKey}, Target: targetOf(md)},
		entity: h.entity, properties: []string{h.property}, serviceName: position.serviceName, consumedHops: 1,
	}
}

func buildLinkModuleStep(h hop, targetHop *hop, position *chainPosition, rootConfig *ServiceConfig, catalog *GraphCatalog) *chainStep {
	if targetHop == nil || targetHop.kind != "relationship" {
		return nil
	}
	linkRel, targetRel := h.relationship, targetHop.relationship
	if linkRel.Inverse || targetRel.Inverse {
		return nil
	}
	if !isSimpleKey(linkRel.PrimaryKey) || !isSimpleKey(linkRel.ForeignKey) || !isSimpleKey(targetRel.PrimaryKey) || !isSimpleKey(targetRel.ForeignKey) {
		return nil
	}
	if linkRel.ForeignKey != position.correlateKey {
		return nil
	}
	md := catalog.GetAliasMetadata(targetHop.entity)
	if md == nil || md.TableName == "" {
		return nil
	}
	linkTable := resolveLinkTableName(h.serviceConfig)
	if linkTable == "" {
		return nil
	}
	target := dal.CrossModuleJoinTarget{Table: md.TableName, Schema: md.Schema, PrimaryKey: targetRel.PrimaryKey}
	return &chainStep{
		spec:   &dal.CrossModuleJoinSpec{Link: dal.CrossModuleJoinLink{Table: linkTable, SourceKey: linkRel.PrimaryKey, TargetKey: targetRel.ForeignKey}, Target: target},
		entity: targetHop.entity, properties: []string{h.property, targetHop.property}, serviceName: targetHop.serviceConfig.ServiceName, consumedHops: 2,
	}
}

func buildReadOnlyLinkStep(h hop, position *chainPosition, rootConfig *ServiceConfig, catalog *GraphCatalog) *chainStep {
	fk := resolveFkHopParts(h, position, rootConfig, catalog)
	if fk == nil {
		return nil
	}
	if fk.direction == "inverse" {
		return &chainStep{
			spec:   &dal.CrossModuleJoinSpec{Link: dal.CrossModuleJoinLink{Table: fk.target.Table, Schema: fk.target.Schema, SourceKey: fk.targetColumn, TargetKey: "id"}, Target: fk.target},
			entity: h.entity, properties: []string{h.property}, serviceName: h.serviceConfig.ServiceName, consumedHops: 1,
		}
	}
	if position.currentTable == "" {
		return nil
	}
	return &chainStep{
		spec:   &dal.CrossModuleJoinSpec{Link: dal.CrossModuleJoinLink{Table: position.currentTable, SourceKey: position.correlateKey, TargetKey: fk.column}, Target: fk.target},
		entity: h.entity, properties: []string{h.property}, serviceName: h.serviceConfig.ServiceName, consumedHops: 1,
	}
}

type fkHopParts struct {
	direction    string // "forward" | "inverse"
	column       string
	targetColumn string
	target       dal.CrossModuleJoinTarget
}

func resolveFkHopParts(h hop, position *chainPosition, rootConfig *ServiceConfig, catalog *GraphCatalog) *fkHopParts {
	relationship := h.relationship
	if relationship.Inverse || relationship.ServiceName == position.serviceName {
		return nil
	}
	fkSegments := strings.Split(relationship.ForeignKey, ".")
	column := fkSegments[len(fkSegments)-1]
	prefix := strings.Join(fkSegments[:len(fkSegments)-1], ".")
	if prefix != strings.Join(h.internalTrail, ".") {
		return nil
	}
	if strings.Contains(column, ",") || !isSimpleKey(relationship.PrimaryKey) {
		return nil
	}
	md := catalog.GetAliasMetadata(h.entity)
	if md == nil || md.TableName == "" {
		return nil
	}
	if h.entity != "" {
		if cfg := catalog.GetServiceConfig("", "", h.entity); cfg == nil || cfg.ServiceName != relationship.ServiceName {
			return nil
		}
	} else {
		return nil
	}
	target := targetOf(md)
	if relationship.PrimaryKey == "id" {
		return &fkHopParts{direction: "forward", column: column, target: target}
	}
	if column != position.correlateKey {
		return nil
	}
	return &fkHopParts{direction: "inverse", targetColumn: relationship.PrimaryKey, target: target}
}

var composeTableNameRe = regexp.MustCompile(`(?i)(_id|Service)$`)

func resolveLinkTableName(linkConfig *ServiceConfig) string {
	if linkConfig.DatabaseConfig != nil && linkConfig.DatabaseConfig.TableName != "" {
		return linkConfig.DatabaseConfig.TableName
	}
	var moduleRels []*Relationship
	for _, key := range linkConfig.Relationships.keys {
		for _, rel := range linkConfig.Relationships.values[key] {
			if rel.ServiceName != linkConfig.ServiceName {
				moduleRels = append(moduleRels, rel)
			}
		}
	}
	if len(moduleRels) != 2 {
		return ""
	}
	parts := []string{moduleRels[0].ServiceName, moduleRels[0].ForeignKey, moduleRels[1].ServiceName, moduleRels[1].ForeignKey}
	for i, p := range parts {
		parts[i] = composeTableNameRe.ReplaceAllString(p, "")
	}
	return strings.ToLower(strings.Join(parts, "_"))
}

// registerCandidates mirrors registerCandidates.
func registerCandidates(registry *specRegistry, candidates []pushdownCandidate) bool {
	additions := map[string]*dal.CrossModuleJoinSpec{}
	var additionKeys []string
	used := map[string]bool{}
	for _, k := range registry.keys {
		used[registry.specs[k].Target.Table] = true
	}
	for _, c := range candidates {
		for _, level := range c.levels {
			if registry.specs[level.realPathKey] != nil || additions[level.realPathKey] != nil {
				continue
			}
			if used[level.spec.Target.Table] {
				return false
			}
			additions[level.realPathKey] = level.spec
			additionKeys = append(additionKeys, level.realPathKey)
			used[level.spec.Target.Table] = true
		}
	}
	for _, k := range additionKeys {
		registry.keys = append(registry.keys, k)
		registry.specs[k] = additions[k]
	}
	for _, c := range candidates {
		if len(c.levels) == 0 {
			continue
		}
		leaf := c.levels[len(c.levels)-1]
		spec := registry.specs[leaf.realPathKey]
		if spec != nil && c.filters.Len() > 0 {
			if spec.Target.Filters != nil {
				and := dal.NewRecord()
				and.Set("$and", []any{spec.Target.Filters, c.filters})
				spec.Target.Filters = and
			} else {
				spec.Target.Filters = c.filters
			}
		}
	}
	return true
}

var supportedOperators = map[string]bool{"$eq": true, "$ne": true, "$in": true, "$nin": true, "$gt": true, "$gte": true, "$lt": true, "$lte": true, "$like": true, "$ilike": true, "$is": true}

func getFiltersArg(args []Arg) *Arg {
	for i := range args {
		if args[i].Name == "filters" {
			return &args[i]
		}
	}
	return nil
}

func isRecordValue(v any) bool {
	_, ok := v.(*dal.Record)
	return ok
}

// rewriteRootFilters mirrors rewriteRootFilters: pushable cross-module root
// filters move into the registry, others are kept as residual.
func rewriteRootFilters(query *RemoteJoinerQuery, serviceConfig *ServiceConfig, catalog *GraphCatalog, registry *specRegistry, residual *[]ResidualCrossModuleFilter) {
	arg := getFiltersArg(query.Args)
	if arg == nil {
		return
	}
	rootFilters, ok := arg.Value.(*dal.Record)
	if !ok || rootFilters == nil {
		return
	}
	for _, key := range rootFilters.Keys() {
		value, _ := rootFilters.Get(key)
		filters, isRec := value.(*dal.Record)
		if strings.HasPrefix(key, "$") || !isRec {
			continue
		}
		res := resolveChain(serviceConfig, []string{key}, catalog)
		if res.outcome == "native" {
			continue
		}
		if res.outcome == "residual" {
			*residual = append(*residual, ResidualCrossModuleFilter{Path: key, Filters: filters})
			continue
		}
		candidates := collectCandidates([]string{key}, res.levels, filters, serviceConfig, catalog)
		if candidates == nil || !registerCandidates(registry, candidates) {
			*residual = append(*residual, ResidualCrossModuleFilter{Path: key, Filters: filters})
			continue
		}
		rootFilters.Delete(key)
	}
}

// rewriteExpandFilters mirrors rewriteExpandFilters.
func rewriteExpandFilters(query *RemoteJoinerQuery, serviceConfig *ServiceConfig, catalog *GraphCatalog, registry *specRegistry, residual *[]ResidualCrossModuleFilter) {
	var withFilters []int
	for i, e := range query.Expands {
		if a := getFiltersArg(e.Args); a != nil {
			if r, ok := a.Value.(*dal.Record); ok && r.Len() > 0 {
				withFilters = append(withFilters, i)
			}
		}
	}
	var pruned []string
	for _, idx := range withFilters {
		expand := &query.Expands[idx]
		filtersArg := getFiltersArg(expand.Args)
		pathSegments := strings.Split(expand.Property, ".")
		filters := unwrapFieldAliasNestedFilters(serviceConfig, pathSegments, filtersArg.Value.(*dal.Record), catalog)
		filtersArg.Value = filters
		res := resolveChain(serviceConfig, pathSegments, catalog)
		if res.outcome == "native" {
			continue
		}
		if res.outcome == "residual" {
			*residual = append(*residual, ResidualCrossModuleFilter{Path: expand.Property, Filters: filters})
			continue
		}
		candidates := collectCandidates(pathSegments, res.levels, filters, serviceConfig, catalog)
		if candidates == nil || !registerCandidates(registry, candidates) {
			*residual = append(*residual, ResidualCrossModuleFilter{Path: expand.Property, Filters: filters})
			continue
		}
		pruned = append(pruned, expand.Property)
	}
	for _, property := range pruned {
		pruneFiltersFromExpand(query, property)
	}
}

func unwrapFieldAliasNestedFilters(rootConfig *ServiceConfig, pathSegments []string, filters *dal.Record, catalog *GraphCatalog) *dal.Record {
	last := pathSegments[len(pathSegments)-1]
	keys := filters.Keys()
	wrappedV, _ := filters.Get(last)
	wrapped, ok := wrappedV.(*dal.Record)
	if len(keys) != 1 || keys[0] != last || !ok {
		return filters
	}
	parentSegments := pathSegments[:len(pathSegments)-1]
	var parentHops []hop
	if len(parentSegments) > 0 {
		parentHops = resolvePathHops(rootConfig, parentSegments, catalog)
		if parentHops == nil {
			return filters
		}
	}
	parentConfig := rootConfig
	parentEntity := rootConfig.Entity
	if len(parentHops) > 0 {
		ph := parentHops[len(parentHops)-1]
		parentConfig = ph.serviceConfig
		if ph.entity != "" {
			parentEntity = ph.entity
		}
	}
	if resolveFieldAliasEntry(parentConfig.FieldAlias[last], parentEntity) == nil {
		return filters
	}
	hops := resolvePathHops(rootConfig, pathSegments, catalog)
	if len(hops) == 0 || hops[len(hops)-1].kind != "internal" {
		return filters
	}
	return wrapped
}

func collectCandidates(pathSegments []string, levels []chainLevel, filters *dal.Record, rootConfig *ServiceConfig, catalog *GraphCatalog) []pushdownCandidate {
	leaf := levels[len(levels)-1]
	crossjoinable := map[string]bool{}
	if md := catalog.GetAliasMetadata(leaf.entity); md != nil {
		for _, f := range md.Crossjoinable {
			crossjoinable[f] = true
		}
	}
	fieldFilters := dal.NewRecord()
	var candidates []pushdownCandidate
	for _, key := range filters.Keys() {
		value, _ := filters.Get(key)
		if !strings.HasPrefix(key, "$") && !crossjoinable[key] && isRecordValue(value) {
			deeper := append(append([]string{}, pathSegments...), key)
			res := resolveChain(rootConfig, deeper, catalog)
			if res.outcome != "pushable" {
				return nil
			}
			nested := collectCandidates(deeper, res.levels, value.(*dal.Record), rootConfig, catalog)
			if nested == nil {
				return nil
			}
			candidates = append(candidates, nested...)
			continue
		}
		fieldFilters.Set(key, value)
	}
	if !areFiltersPushable(fieldFilters, crossjoinable) {
		return nil
	}
	return append([]pushdownCandidate{{levels: levels, filters: fieldFilters}}, candidates...)
}

func areFiltersPushable(filters *dal.Record, crossjoinable map[string]bool) bool {
	for _, key := range filters.Keys() {
		value, _ := filters.Get(key)
		if key == "$and" || key == "$or" {
			list, ok := value.([]any)
			if !ok {
				return false
			}
			for _, c := range list {
				r, ok := c.(*dal.Record)
				if !ok || !areFiltersPushable(r, crossjoinable) {
					return false
				}
			}
			continue
		}
		if strings.HasPrefix(key, "$") || !crossjoinable[key] || !isPushableFilterValue(value) {
			return false
		}
	}
	return true
}

func isPushableFilterValue(value any) bool {
	if list, ok := value.([]any); ok {
		for _, e := range list {
			if isRecordValue(e) {
				return false
			}
		}
		return true
	}
	if r, ok := value.(*dal.Record); ok {
		if r.Len() == 0 {
			return false
		}
		for _, op := range r.Keys() {
			v, _ := r.Get(op)
			if !supportedOperators[op] || !isPushableOperatorValue(v) {
				return false
			}
		}
		return true
	}
	return true
}

func isPushableOperatorValue(value any) bool {
	if list, ok := value.([]any); ok {
		for _, e := range list {
			if isRecordValue(e) {
				return false
			}
		}
		return true
	}
	return !isRecordValue(value)
}

func pruneFiltersFromExpand(query *RemoteJoinerQuery, property string) {
	indexOf := func(p string) int {
		for i, e := range query.Expands {
			if e.Property == p {
				return i
			}
		}
		return -1
	}
	if i := indexOf(property); i >= 0 {
		var remaining []Arg
		for _, a := range query.Expands[i].Args {
			if a.Name != "filters" {
				remaining = append(remaining, a)
			}
		}
		query.Expands[i].Args = remaining
	}
	for property != "" {
		i := indexOf(property)
		if i < 0 {
			break
		}
		node := query.Expands[i]
		prefix := property + "."
		hasDescendants := false
		for j, e := range query.Expands {
			if j != i && strings.HasPrefix(e.Property, prefix) {
				hasDescendants = true
				break
			}
		}
		if hasDescendants || len(node.Fields) > 0 || len(node.Args) > 0 {
			break
		}
		query.Expands = append(query.Expands[:i], query.Expands[i+1:]...)
		if k := strings.LastIndex(property, "."); k >= 0 {
			property = property[:k]
		} else {
			property = ""
		}
	}
}

// extractCrossModuleJoins mirrors extractCrossModuleJoins.
func extractCrossModuleJoins(query *RemoteJoinerQuery, serviceConfig *ServiceConfig, catalog *GraphCatalog) ([]*dal.CrossModuleJoinSpec, []ResidualCrossModuleFilter, []ResidualOrderBy) {
	registry := newSpecRegistry()
	var residual []ResidualCrossModuleFilter
	rewriteRootFilters(query, serviceConfig, catalog, registry, &residual)
	rewriteExpandFilters(query, serviceConfig, catalog, registry, &residual)
	residualOrder := rewriteRootOrder(query, serviceConfig, catalog, registry)
	out := make([]*dal.CrossModuleJoinSpec, 0, len(registry.keys))
	for _, k := range registry.keys {
		out = append(out, registry.specs[k])
	}
	return out, residual, residualOrder
}

// ResidualOrderBy mirrors ResidualOrderBy.
type ResidualOrderBy struct {
	Segments  []string
	Direction string
}

type orderEntry struct {
	segments  []string
	direction string
	owner     *dal.Record
	key       string
}

func flattenOrder(order *dal.Record, prefix []string, entries []orderEntry) []orderEntry {
	for _, key := range order.Keys() {
		value, _ := order.Get(key)
		segments := append(append([]string{}, prefix...), strings.Split(key, ".")...)
		if r, ok := value.(*dal.Record); ok {
			entries = flattenOrder(r, segments, entries)
			continue
		}
		if s, ok := value.(string); ok && (strings.EqualFold(s, "asc") || strings.EqualFold(s, "desc")) {
			entries = append(entries, orderEntry{segments: segments, direction: strings.ToUpper(s), owner: order, key: key})
		}
	}
	return entries
}

type orderClassification struct {
	kind   string // native | pushable | residual
	levels []chainLevel
	field  string
}

func (r *specRegistry) clone() *specRegistry {
	c := newSpecRegistry()
	c.keys = append(c.keys, r.keys...)
	for k, v := range r.specs {
		c.specs[k] = v
	}
	return c
}

func classifyOrderPath(segments []string, serviceConfig *ServiceConfig, registry *specRegistry, catalog *GraphCatalog) orderClassification {
	if len(segments) < 2 {
		return orderClassification{kind: "native"}
	}
	prefixLength := 0
	crosses := false
	for end := len(segments) - 1; end >= 1; end-- {
		if hops := resolvePathHops(serviceConfig, segments[:end], catalog); hops != nil {
			prefixLength = end
			crosses = crossesModules(hops, serviceConfig)
			break
		}
	}
	if prefixLength == 0 || !crosses {
		return orderClassification{kind: "native"}
	}
	if prefixLength == len(segments)-1 {
		res := resolveChain(serviceConfig, segments[:len(segments)-1], catalog)
		if res.outcome == "pushable" {
			field := segments[len(segments)-1]
			leaf := res.levels[len(res.levels)-1]
			if md := catalog.GetAliasMetadata(leaf.entity); md != nil {
				for _, f := range md.Crossjoinable {
					if f == field && registerCandidates(registry, []pushdownCandidate{{levels: res.levels, filters: dal.NewRecord()}}) {
						return orderClassification{kind: "pushable", levels: res.levels, field: field}
					}
				}
			}
		}
	}
	return orderClassification{kind: "residual"}
}

func pruneEmptyObjects(obj *dal.Record) {
	for _, k := range obj.Keys() {
		v, _ := obj.Get(k)
		if r, ok := v.(*dal.Record); ok {
			pruneEmptyObjects(r)
			if r.Len() == 0 {
				obj.Delete(k)
			}
		}
	}
}

// rewriteRootOrder mirrors rewriteRootOrder.
func rewriteRootOrder(query *RemoteJoinerQuery, serviceConfig *ServiceConfig, catalog *GraphCatalog, registry *specRegistry) []ResidualOrderBy {
	argIndex := -1
	for i, a := range query.Args {
		if a.Name == "order" {
			argIndex = i
			break
		}
	}
	if argIndex < 0 {
		return nil
	}
	order, ok := query.Args[argIndex].Value.(*dal.Record)
	if !ok {
		return nil
	}
	entries := flattenOrder(order, nil, nil)
	if len(entries) == 0 {
		return nil
	}
	scratch := registry.clone()
	classified := make([]orderClassification, len(entries))
	residual := false
	for i, e := range entries {
		classified[i] = classifyOrderPath(e.segments, serviceConfig, scratch, catalog)
		if classified[i].kind == "residual" {
			residual = true
		}
	}
	if residual {
		query.Args = append(query.Args[:argIndex:argIndex], query.Args[argIndex+1:]...)
		out := make([]ResidualOrderBy, len(entries))
		for i, e := range entries {
			out[i] = ResidualOrderBy{Segments: e.segments, Direction: e.direction}
		}
		return out
	}
	type rewrite struct{ key, dir string }
	var rewrites []rewrite
	for i, e := range entries {
		c := classified[i]
		if c.kind != "pushable" {
			continue
		}
		registerCandidates(registry, []pushdownCandidate{{levels: c.levels, filters: dal.NewRecord()}})
		e.owner.Delete(e.key)
		leaf := c.levels[len(c.levels)-1]
		rewrites = append(rewrites, rewrite{leaf.spec.Target.Table + "." + c.field, e.direction})
	}
	pruneEmptyObjects(order)
	for _, r := range rewrites {
		order.Set(r.key, r.dir)
	}
	return nil
}
