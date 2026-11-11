package middlewares

import (
	"time"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// PublishableKeyHeader mirrors PUBLISHABLE_KEY_HEADER.
const PublishableKeyHeader = "x-publishable-api-key"

// EnsurePublishableApiKey mirrors ensurePublishableApiKeyMiddleware: the
// publishable key context (key, sales_channel_ids) of a valid key.
func EnsurePublishableApiKey(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	key := req.Header.Get(PublishableKeyHeader)
	if key == "" {
		next(common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Publishable API key required in the request header: "+PublishableKeyHeader+". You can manage your keys in settings in the dashboard."))
		return nil
	}
	q, _ := req.Scope.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
	f := dal.NewRecord()
	f.Set("token", key)
	f.Set("type", "publishable")
	r, err := q.Graph(req.Ctx(), query.GraphInput{Entity: "api_key", Fields: []string{"id", "token", "revoked_at", "sales_channels_link.sales_channel_id"}, Filters: f}, joiner.Options{
		// The api key itself is computed from the response. The linked sales
		// channels are not.
		Cache: caching.WithTags(true, "LinkPublishableApiKeySalesChannel:list:*"),
	})
	if err != nil {
		next(err)
		return nil
	}
	var apiKey *dal.Record
	if len(r.Data) > 0 {
		k, _ := r.Data[0].(*dal.Record)
		revoked := false
		if v, _ := k.Get("revoked_at"); v != nil {
			switch t := v.(type) {
			case time.Time:
				revoked = !t.After(time.Now())
			case string:
				if at, ok := common.ParseJSDate(t); ok {
					revoked = !at.After(time.Now())
				}
			}
		}
		if !revoked {
			apiKey = k
		}
	}
	if apiKey == nil {
		next(common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "A valid publishable key is required to proceed with the request"))
		return nil
	}
	ids := []any{}
	links, _ := apiKey.Get("sales_channels_link")
	if list, ok := links.([]any); ok {
		for _, l := range list {
			if lr, ok := l.(*dal.Record); ok {
				id, _ := lr.Get("sales_channel_id")
				ids = append(ids, id)
			}
		}
	}
	ctx := dal.NewRecord()
	token, _ := apiKey.Get("token")
	ctx.Set("key", token)
	ctx.Set("sales_channel_ids", ids)
	req.PublishableKeyContext = ctx
	next(nil)
	return nil
}
