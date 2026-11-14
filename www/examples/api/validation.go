package api

import (
	"encoding/json"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// region: body-schema
// PostSumSchema accepts an object with two required numbers, a and b.
var PostSumSchema = zod.Object(
	zod.F("a", zod.Number()),
	zod.F("b", zod.Number()),
)

// endregion: body-schema

// region: query-schema
// GetSumSchema accepts the query parameters a and b. Query parameters are
// strings, so they are converted into numbers before they're validated.
var GetSumSchema = zod.Object(
	zod.F("a", zod.Preprocess(parseNumber, zod.Number())),
	zod.F("b", zod.Preprocess(parseNumber, zod.Number())),
)

// endregion: query-schema

// region: middlewares
// SumMiddlewares validate the body of POST requests and the query parameters
// of GET requests to /custom/sum.
var SumMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher: "/custom/sum",
		Methods: []string{"POST"},
		Middlewares: []medusahttp.MiddlewareFunction{
			httputils.ValidateAndTransformBody(httputils.Schema(PostSumSchema)),
		},
	},
	{
		Matcher: "/custom/sum",
		Methods: []string{"GET"},
		Middlewares: []medusahttp.MiddlewareFunction{
			httputils.ValidateAndTransformQuery(GetSumSchema, httputils.QueryConfig{}),
		},
	},
}

// endregion: middlewares

// region: handlers
func sumPOST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	// ValidatedBody holds the output of the schema: an object whose
	// properties have the types the schema declares.
	validated := body(req.ValidatedBody)

	return res.JSON(map[string]any{
		"sum": numberOf(validated, "a") + numberOf(validated, "b"),
	})
}

func sumGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.JSON(map[string]any{
		"sum": numberOf(req.ValidatedQuery, "a") + numberOf(req.ValidatedQuery, "b"),
	})
}

// endregion: handlers

// region: route
// SumRoute is the route file of /custom/sum.
var SumRoute = medusahttp.RouteFile{
	Path:  "/custom/sum",
	Route: medusahttp.Route{GET: sumGET, POST: sumPOST},
}

// endregion: route

// region: richer-schema
// CreatePostSchema shows other common rules: optional properties, string
// length checks, enums, arrays and defaults.
var CreatePostSchema = zod.Object(
	zod.F("title", zod.String().Min(1)),
	zod.F("author_email", zod.String().Email()),
	zod.F("status", zod.Enum("draft", "published").Default("draft")),
	zod.F("tags", zod.Array(zod.String()).Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// endregion: richer-schema

// region: typed
// CreatePostInput is the type of the validated body of CreatePostSchema.
type CreatePostInput struct {
	Title       string         `json:"title"`
	AuthorEmail string         `json:"author_email"`
	Status      string         `json:"status"`
	Tags        []string       `json:"tags"`
	Metadata    map[string]any `json:"metadata"`
}

// decode converts a validated value into a Go type.
func decode[T any](validated any) (T, error) {
	var out T

	raw, err := dal.MarshalValue(validated)
	if err != nil {
		return out, err
	}

	err = json.Unmarshal(raw, &out)
	return out, err
}

func createPostPOST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	input, err := decode[CreatePostInput](req.ValidatedBody)
	if err != nil {
		return err
	}

	return res.Status(201).JSON(map[string]any{
		"title":  input.Title,
		"status": input.Status,
	})
}

// endregion: typed

// CreatePostRoute validates the body of POST /custom/posts with
// CreatePostSchema.
var CreatePostRoute = medusahttp.RouteFile{
	Path:  "/custom/posts",
	Route: medusahttp.Route{POST: createPostPOST},
}

// CreatePostMiddlewares validate the body of POST /custom/posts.
var CreatePostMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher: "/custom/posts",
		Methods: []string{"POST"},
		Middlewares: []medusahttp.MiddlewareFunction{
			httputils.ValidateAndTransformBody(httputils.Schema(CreatePostSchema)),
		},
	},
}
