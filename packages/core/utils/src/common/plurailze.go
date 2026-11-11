package common

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The pluralization rules mirror the "pluralize" npm package (v8) used by
// plurailze.ts, with the "info" uncountable rule Medusa adds.

type pluralizeRule struct {
	re          *regexp.Regexp
	replacement string
}

var (
	pluralRules       []pluralizeRule
	singularRules     []pluralizeRule
	pluralUncountable = map[string]bool{}
	irregularPlurals  = map[string]string{}
	irregularSingles  = map[string]string{}
	interpolateRe     = regexp.MustCompile(`\$(\d{1,2})`)
)

func sanitizePluralizeRule(rule string) *regexp.Regexp {
	return regexp.MustCompile("(?i)" + rule)
}

func addPluralRule(rule, replacement string) {
	pluralRules = append(pluralRules, pluralizeRule{sanitizePluralizeRule(rule), replacement})
}

func addSingularRule(rule, replacement string) {
	singularRules = append(singularRules, pluralizeRule{sanitizePluralizeRule(rule), replacement})
}

func addIrregularRule(single, plural string) {
	plural = strings.ToLower(plural)
	single = strings.ToLower(single)
	irregularSingles[single] = plural
	irregularPlurals[plural] = single
}

// addUncountableRule takes a word or, prefixed with "re:", a pattern.
func addUncountableRule(word string) {
	if strings.HasPrefix(word, "re:") {
		addPluralRule(word[3:], "$0")
		addSingularRule(word[3:], "$0")
		return
	}
	pluralUncountable[strings.ToLower(word)] = true
}

func restoreCase(word, token string) string {
	if word == token {
		return token
	}
	if word == strings.ToLower(word) {
		return strings.ToLower(token)
	}
	if word == strings.ToUpper(word) {
		return strings.ToUpper(token)
	}
	first, size := utf8.DecodeRuneInString(word)
	if size > 0 && string(first) == strings.ToUpper(string(first)) {
		t, tsize := utf8.DecodeRuneInString(token)
		if tsize == 0 {
			return token
		}
		return strings.ToUpper(string(t)) + strings.ToLower(token[tsize:])
	}
	return strings.ToLower(token)
}

func pluralizeReplace(word string, rule pluralizeRule) string {
	loc := rule.re.FindStringSubmatchIndex(word)
	if loc == nil {
		return word
	}
	args := make([]string, len(loc)/2)
	for i := range args {
		if loc[2*i] >= 0 {
			args[i] = word[loc[2*i]:loc[2*i+1]]
		}
	}
	match := args[0]
	result := interpolateRe.ReplaceAllStringFunc(rule.replacement, func(m string) string {
		idx, _ := strconv.Atoi(m[1:])
		if idx < len(args) {
			return args[idx]
		}
		return ""
	})
	var replaced string
	if match == "" {
		// Upstream passes word[index - 1], index being the match offset for
		// rules without capture groups.
		prev := ""
		if loc[0] > 0 {
			r, _ := utf8.DecodeLastRuneInString(word[:loc[0]])
			prev = string(r)
		}
		replaced = restoreCase(prev, result)
	} else {
		replaced = restoreCase(match, result)
	}
	return word[:loc[0]] + replaced + word[loc[1]:]
}

func sanitizePluralizeWord(token, word string, rules []pluralizeRule) string {
	if token == "" || pluralUncountable[token] {
		return word
	}
	for i := len(rules) - 1; i >= 0; i-- {
		if rules[i].re.MatchString(word) {
			return pluralizeReplace(word, rules[i])
		}
	}
	return word
}

func replacePluralizeWord(replaceMap, keepMap map[string]string, rules []pluralizeRule, word string) string {
	token := strings.ToLower(word)
	if _, ok := keepMap[token]; ok {
		return restoreCase(word, token)
	}
	if r, ok := replaceMap[token]; ok {
		return restoreCase(word, r)
	}
	return sanitizePluralizeWord(token, word, rules)
}

// Pluralize mirrors pluralize from plurailze.ts.
func Pluralize(word string) string {
	return replacePluralizeWord(irregularSingles, irregularPlurals, pluralRules, word)
}

// Singularize mirrors singularize from plurailze.ts.
func Singularize(word string) string {
	return replacePluralizeWord(irregularPlurals, irregularSingles, singularRules, word)
}

func init() {
	for _, r := range [][2]string{
		{"I", "we"}, {"me", "us"}, {"he", "they"}, {"she", "they"}, {"them", "them"},
		{"myself", "ourselves"}, {"yourself", "yourselves"}, {"itself", "themselves"},
		{"herself", "themselves"}, {"himself", "themselves"}, {"themself", "themselves"},
		{"is", "are"}, {"was", "were"}, {"has", "have"}, {"this", "these"}, {"that", "those"},
		{"echo", "echoes"}, {"dingo", "dingoes"}, {"volcano", "volcanoes"}, {"tornado", "tornadoes"},
		{"torpedo", "torpedoes"}, {"genus", "genera"}, {"viscus", "viscera"}, {"stigma", "stigmata"},
		{"stoma", "stomata"}, {"dogma", "dogmata"}, {"lemma", "lemmata"}, {"schema", "schemata"},
		{"anathema", "anathemata"}, {"ox", "oxen"}, {"axe", "axes"}, {"die", "dice"}, {"yes", "yeses"},
		{"foot", "feet"}, {"eave", "eaves"}, {"goose", "geese"}, {"tooth", "teeth"}, {"quiz", "quizzes"},
		{"human", "humans"}, {"proof", "proofs"}, {"carve", "carves"}, {"valve", "valves"},
		{"looey", "looies"}, {"thief", "thieves"}, {"groove", "grooves"}, {"pickaxe", "pickaxes"},
		{"passerby", "passersby"},
	} {
		addIrregularRule(r[0], r[1])
	}
	for _, r := range [][2]string{
		{`s?$`, "s"},
		{`[^\x{0000}-\x{007F}]$`, "$0"},
		{`([^aeiou]ese)$`, "$1"},
		{`(ax|test)is$`, "$1es"},
		{`(alias|[^aou]us|t[lm]as|gas|ris)$`, "$1es"},
		{`(e[mn]u)s?$`, "$1s"},
		{`([^l]ias|[aeiou]las|[ejzr]as|[iu]am)$`, "$1"},
		{`(alumn|syllab|vir|radi|nucle|fung|cact|stimul|termin|bacill|foc|uter|loc|strat)(?:us|i)$`, "$1i"},
		{`(alumn|alg|vertebr)(?:a|ae)$`, "$1ae"},
		{`(seraph|cherub)(?:im)?$`, "$1im"},
		{`(her|at|gr)o$`, "$1oes"},
		{`(agend|addend|millenni|dat|extrem|bacteri|desiderat|strat|candelabr|errat|ov|symposi|curricul|automat|quor)(?:a|um)$`, "$1a"},
		{`(apheli|hyperbat|periheli|asyndet|noumen|phenomen|criteri|organ|prolegomen|hedr|automat)(?:a|on)$`, "$1a"},
		{`sis$`, "ses"},
		{`(?:(kni|wi|li)fe|(ar|l|ea|eo|oa|hoo)f)$`, "$1$2ves"},
		{`([^aeiouy]|qu)y$`, "$1ies"},
		{`([^ch][ieo][ln])ey$`, "$1ies"},
		{`(x|ch|ss|sh|zz)$`, "$1es"},
		{`(matr|cod|mur|sil|vert|ind|append)(?:ix|ex)$`, "$1ices"},
		{`\b((?:tit)?m|l)(?:ice|ouse)$`, "$1ice"},
		{`(pe)(?:rson|ople)$`, "$1ople"},
		{`(child)(?:ren)?$`, "$1ren"},
		{`eaux$`, "$0"},
		{`m[ae]n$`, "men"},
		{`^thou$`, "you"},
	} {
		addPluralRule(r[0], r[1])
	}
	for _, r := range [][2]string{
		{`s$`, ""},
		{`(ss)$`, "$1"},
		{`(wi|kni|(?:after|half|high|low|mid|non|night|[^\w]|^)li)ves$`, "$1fe"},
		{`(ar|(?:wo|[ae])l|[eo][ao])ves$`, "$1f"},
		{`ies$`, "y"},
		{`\b([pl]|zomb|(?:neck|cross)?t|coll|faer|food|gen|goon|group|lass|talk|goal|cut)ies$`, "$1ie"},
		{`\b(mon|smil)ies$`, "$1ey"},
		{`\b((?:tit)?m|l)ice$`, "$1ouse"},
		{`(seraph|cherub)im$`, "$1"},
		{`(x|ch|ss|sh|zz|tto|go|cho|alias|[^aou]us|t[lm]as|gas|(?:her|at|gr)o|[aeiou]ris)(?:es)?$`, "$1"},
		{`(analy|diagno|parenthe|progno|synop|the|empha|cri|ne)(?:sis|ses)$`, "$1sis"},
		{`(movie|twelve|abuse|e[mn]u)s$`, "$1"},
		{`(test)(?:is|es)$`, "$1is"},
		{`(alumn|syllab|vir|radi|nucle|fung|cact|stimul|termin|bacill|foc|uter|loc|strat)(?:us|i)$`, "$1us"},
		{`(agend|addend|millenni|dat|extrem|bacteri|desiderat|strat|candelabr|errat|ov|symposi|curricul|quor)a$`, "$1um"},
		{`(apheli|hyperbat|periheli|asyndet|noumen|phenomen|criteri|organ|prolegomen|hedr|automat)a$`, "$1on"},
		{`(alumn|alg|vertebr)ae$`, "$1a"},
		{`(cod|mur|sil|vert|ind)ices$`, "$1ex"},
		{`(matr|append)ices$`, "$1ix"},
		{`(pe)(rson|ople)$`, "$1rson"},
		{`(child)ren$`, "$1"},
		{`(eau)x?$`, "$1"},
		{`men$`, "man"},
	} {
		addSingularRule(r[0], r[1])
	}
	for _, w := range []string{
		"adulthood", "advice", "agenda", "aid", "aircraft", "alcohol", "ammo", "analytics", "anime",
		"athletics", "audio", "bison", "blood", "bream", "buffalo", "butter", "carp", "cash", "chassis",
		"chess", "clothing", "cod", "commerce", "cooperation", "corps", "debris", "diabetes", "digestion",
		"elk", "energy", "equipment", "excretion", "expertise", "firmware", "flounder", "fun", "gallows",
		"garbage", "graffiti", "hardware", "headquarters", "health", "herpes", "highjinks", "homework",
		"housework", "information", "jeans", "justice", "kudos", "labour", "literature", "machinery",
		"mackerel", "mail", "media", "mews", "moose", "music", "mud", "manga", "news", "only", "personnel",
		"pike", "plankton", "pliers", "police", "pollution", "premises", "rain", "research", "rice",
		"salmon", "scissors", "series", "sewage", "shambles", "shrimp", "software", "species", "staff",
		"swine", "tennis", "traffic", "transportation", "trout", "tuna", "wealth", "welfare", "whiting",
		"wildebeest", "wildlife", "you",
		`re:pok[eé]mon$`, `re:[^aeiou]ese$`, `re:deer$`, `re:fish$`, `re:measles$`, `re:o[iu]s$`,
		`re:pox$`, `re:sheep$`,
	} {
		addUncountableRule(w)
	}
	// plurailze.ts
	addUncountableRule("info")
}
