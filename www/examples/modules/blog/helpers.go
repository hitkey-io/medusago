package blog

// region: rec
import "github.com/hitkey-io/medusago/packages/core/utils/src/dal"

// rec builds a *dal.Record, an ordered map that encodes to JSON with its
// keys in insertion order, like a JavaScript object.
func rec(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// endregion: rec
