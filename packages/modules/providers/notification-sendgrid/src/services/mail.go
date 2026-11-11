package services

import (
	"errors"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Port of the parts of @sendgrid/helpers (Mail, Personalization,
// EmailAddress) the provider uses: Mail.create(data).toJSON(). Validation
// errors only need to fail (the provider reports them as "undefined -
// unknown error").

var errInvalidMail = errors.New("invalid mail data")

// emailAddress mirrors EmailAddress.create(...).toJSON().
func emailAddress(v any) (*dal.Record, error) {
	var name, email any
	nameSet := false
	switch t := v.(type) {
	case string:
		name, email = "", t
		nameSet = true
		if strings.Contains(t, "<") {
			parts := strings.SplitN(t, "<", 3)
			name = strings.TrimSpace(parts[0])
			email = strings.TrimSpace(strings.Replace(parts[1], ">", "", 1))
		}
	case *dal.Record:
		email, _ = t.Get("email")
		name, nameSet = t.Get("name")
	default:
		return nil, errInvalidMail
	}
	es, ok := email.(string)
	if !ok {
		return nil, errInvalidMail
	}
	out := dal.NewRecord()
	out.Set("email", es)
	if nameSet {
		ns, ok := name.(string)
		if !ok {
			return nil, errInvalidMail
		}
		if ns != "" {
			out.Set("name", ns)
		}
	}
	return out, nil
}

// emailAddresses mirrors EmailAddress.create over a list (falsy items
// dropped).
func emailAddresses(v any) ([]any, error) {
	list, ok := v.([]any)
	if !ok {
		list = []any{v}
	}
	out := []any{}
	for _, item := range list {
		if item == nil || item == "" || item == false || item == float64(0) {
			continue
		}
		a, err := emailAddress(item)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// deepMerge mirrors the deepmerge package (arrays are concatenated).
func deepMerge(target, source any) any {
	sa, sIsArr := source.([]any)
	ta, tIsArr := target.([]any)
	if sIsArr != tIsArr {
		return source
	}
	if sIsArr {
		return append(append([]any{}, ta...), sa...)
	}
	tr, tok := target.(*dal.Record)
	sr, sok := source.(*dal.Record)
	if !tok || !sok {
		return source
	}
	out := dal.NewRecord()
	for _, k := range tr.Keys() {
		v, _ := tr.Get(k)
		out.Set(k, v)
	}
	for _, k := range sr.Keys() {
		sv, _ := sr.Get(k)
		if tv, ok := tr.Get(k); ok && isMergeable(sv) {
			out.Set(k, deepMerge(tv, sv))
		} else {
			out.Set(k, sv)
		}
	}
	return out
}

func isMergeable(v any) bool {
	switch v.(type) {
	case *dal.Record, []any:
		return true
	}
	return false
}

// personalization mirrors new Personalization(data) + the mail's dynamic
// template data, then toJSON (snake cased keys).
func personalization(data *dal.Record, dynamic bool, templateData any) (*dal.Record, error) {
	field := func(camel, snake string) (any, bool) {
		if v, ok := data.Get(snake); ok {
			return v, true
		}
		return data.Get(camel)
	}
	out := dal.NewRecord()
	to := []any{}
	if v, ok := field("to", "to"); ok {
		var err error
		if to, err = emailAddresses(v); err != nil {
			return nil, err
		}
	}
	out.Set("to", to)
	for _, k := range []string{"cc", "bcc"} {
		if v, ok := field(k, k); ok {
			list, err := emailAddresses(v)
			if err != nil {
				return nil, err
			}
			if len(list) > 0 {
				out.Set(k, list)
			}
		}
	}
	if v, ok := field("headers", "headers"); ok {
		r, isObj := v.(*dal.Record)
		if !isObj {
			return nil, errInvalidMail
		}
		if len(r.Keys()) > 0 {
			out.Set("headers", r)
		}
	}
	if !dynamic {
		if v, ok := field("substitutions", "substitutions"); ok {
			if r, isObj := v.(*dal.Record); isObj && len(r.Keys()) > 0 {
				wrapped := dal.NewRecord()
				for _, k := range r.Keys() {
					x, _ := r.Get(k)
					wrapped.Set("{{"+k+"}}", x)
				}
				out.Set("substitutions", wrapped)
			}
		}
	}
	var tail [][2]any
	if v, ok := field("customArgs", "custom_args"); ok {
		if r, isObj := v.(*dal.Record); isObj && len(r.Keys()) > 0 {
			tail = append(tail, [2]any{"custom_args", r})
		}
	}
	if dynamic {
		var own any = dal.NewRecord()
		if v, ok := field("dynamicTemplateData", "dynamic_template_data"); ok {
			own = v
		}
		merged := own
		if templateData != nil {
			merged = deepMerge(templateData, own)
		}
		if r, isObj := merged.(*dal.Record); isObj && len(r.Keys()) > 0 {
			tail = append(tail, [2]any{"dynamic_template_data", r})
		}
	}
	if v, ok := field("subject", "subject"); ok {
		if _, isStr := v.(string); !isStr {
			return nil, errInvalidMail
		}
		out.Set("subject", v)
	}
	if v, ok := field("sendAt", "send_at"); ok {
		tail = append(tail, [2]any{"send_at", v})
	}
	if v, ok := field("from", "from"); ok {
		a, err := emailAddress(v)
		if err != nil {
			return nil, err
		}
		out.Set("from", a)
	}
	for _, t := range tail {
		out.Set(t[0].(string), t[1])
	}
	return out, nil
}

// buildMail mirrors Mail.create(data).toJSON() for the provider's message
// (from, dynamicTemplateData, attachments, to | personalizations, subject,
// html, templateId) and returns the JSON body.
func buildMail(message *dal.Record) ([]byte, error) {
	json := dal.NewRecord()
	from, _ := message.Get("from")
	fromAddr, err := emailAddress(from)
	if err != nil {
		return nil, err
	}
	json.Set("from", fromAddr)
	if v, ok := message.Get("subject"); ok {
		if _, isStr := v.(string); !isStr {
			return nil, errInvalidMail
		}
		json.Set("subject", v)
	}

	dynamic := false
	templateID, hasTemplate := message.Get("templateId")
	if hasTemplate && templateID != nil {
		ts, isStr := templateID.(string)
		if !isStr {
			return nil, errInvalidMail
		}
		dynamic = strings.HasPrefix(ts, "d-")
	} else {
		hasTemplate = false
	}
	var templateData any
	if dynamic {
		if v, ok := message.Get("dynamicTemplateData"); ok {
			if _, isObj := v.(*dal.Record); !isObj {
				if _, isArr := v.([]any); !isArr {
					return nil, errInvalidMail
				}
			}
			templateData = v
		}
	}

	personalizations := []any{}
	if list, ok := message.Get("personalizations"); ok {
		items, _ := list.([]any)
		for _, item := range items {
			r, isObj := item.(*dal.Record)
			if !isObj {
				return nil, errInvalidMail
			}
			p, err := personalization(r, dynamic, templateData)
			if err != nil {
				return nil, err
			}
			personalizations = append(personalizations, p)
		}
	} else {
		to, ok := message.Get("to")
		if !ok {
			return nil, errInvalidMail
		}
		data := dal.NewRecord()
		if to != nil {
			data.Set("to", to)
		}
		p, err := personalization(data, dynamic, templateData)
		if err != nil {
			return nil, err
		}
		personalizations = append(personalizations, p)
	}
	json.Set("personalizations", personalizations)

	if v, ok := message.Get("attachments"); ok {
		list, _ := v.([]any)
		for _, a := range list {
			r, _ := a.(*dal.Record)
			c, _ := get(r, "content")
			f, _ := get(r, "filename")
			if _, ok := c.(string); !ok {
				return nil, errInvalidMail
			}
			if _, ok := f.(string); !ok {
				return nil, errInvalidMail
			}
		}
		if len(list) > 0 {
			json.Set("attachments", list)
		}
	}
	if v, ok := message.Get("html"); ok {
		hs, isStr := v.(string)
		if !isStr {
			return nil, errInvalidMail
		}
		c := dal.NewRecord()
		c.Set("value", hs)
		c.Set("type", "text/html")
		json.Set("content", []any{c})
	}
	if hasTemplate {
		json.Set("template_id", templateID)
	}
	return dal.MarshalValue(json)
}
