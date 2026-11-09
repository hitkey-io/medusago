package middlewares

import (
	"context"
	"encoding/base64"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

const (
	sessionAuth    = "session"
	bearerAuth     = "bearer"
	apiKeyAuth     = "api-key"
	adminActorType = "user"
)

// ApiKeyAuthenticator is the part of the api key module the middleware uses.
type ApiKeyAuthenticator interface {
	Authenticate(ctx context.Context, token string) (*dal.Record, error)
}

// ConfigModuleKey / ApiKeyModuleKey are the container keys resolved here.
const (
	ConfigModuleKey = "configModule"
	ApiKeyModuleKey = "api_key"
)

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func isActorTypePermitted(actorTypes []string, current string) bool {
	return contains(actorTypes, "*") || contains(actorTypes, current)
}

// Authenticate mirrors authenticate(actorType, authType, options).
func Authenticate(actorTypes []string, authTypes []string, opts medusahttp.AuthenticateOptions) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		isExclusivelyUser := len(actorTypes) == 1 && actorTypes[0] == adminActorType

		if contains(authTypes, apiKeyAuth) && isExclusivelyUser {
			if apiKey := getApiKeyInfo(req); apiKey != nil {
				id, _ := apiKey.Get("id")
				ctx := dal.NewRecord()
				ctx.Set("actor_id", id)
				ctx.Set("actor_type", "api-key")
				ctx.Set("auth_identity_id", "")
				ctx.Set("app_metadata", dal.NewRecord())
				ctx.Set("user_metadata", dal.NewRecord())
				req.AuthContext = ctx
				next(nil)
				return nil
			}
		}

		authContext := getAuthContextFromSession(req.Session, authTypes, actorTypes)
		if authContext == nil {
			secret := "supersecret"
			if c, ok := req.Scope.Resolve(ConfigModuleKey).(config.ConfigModule); ok {
				secret = c.ProjectConfig.HTTP.JwtSecret
			}
			authContext = GetAuthContextFromJwtToken(req.Header.Get("Authorization"), secret, authTypes, actorTypes)
		}

		requireMfa := !opts.AllowUnauthenticated
		if opts.RequireMfa != nil {
			requireMfa = *opts.RequireMfa
		}
		if requireMfa && authContext != nil && truthy(authContext, "mfa_enabled") {
			if msg := mfaRequirementError(authContext, opts.RequireMfaMaxAgeSeconds); msg != "" {
				return res.Status(401).JSON(messageBody(msg))
			}
		}

		if authContext != nil && stringOf(authContext, "actor_id") != "" {
			req.AuthContext = authContext
			next(nil)
			return nil
		}
		if authContext != nil && stringOf(authContext, "auth_identity_id") != "" && opts.AllowUnregistered && isActorTypePermitted(actorTypes, stringOf(authContext, "actor_type")) {
			req.AuthContext = authContext
			next(nil)
			return nil
		}
		if opts.AllowUnauthenticated {
			next(nil)
			return nil
		}
		if contains(authTypes, apiKeyAuth) && isExclusivelyUser && isBearerSecretApiKey(req.Header.Get("Authorization")) {
			return res.Status(401).JSON(messageBody("A secret API key was passed as a Bearer token. Secret API keys must be sent using HTTP Basic authentication instead (Authorization: Basic <secret-api-key>)."))
		}
		return res.Status(401).JSON(messageBody("Unauthorized"))
	}
}

func messageBody(msg string) *dal.Record {
	r := dal.NewRecord()
	r.Set("message", msg)
	return r
}

func stringOf(r *dal.Record, key string) string {
	v, _ := r.Get(key)
	s, _ := v.(string)
	return s
}

// truthy mirrors JS truthiness for a record value.
func truthy(r *dal.Record, key string) bool {
	v, ok := r.Get(key)
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	}
	return true
}

func mfaRequirementError(ctx *dal.Record, maxAgeSeconds int) string {
	completedAt, _ := ctx.Get("mfa_challenge_completed_at")
	s, _ := completedAt.(string)
	if !truthy(ctx, "mfa_challenge_completed_at") {
		return "MFA verification is required to complete this request"
	}
	if maxAgeSeconds == 0 {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil || time.Since(t) > time.Duration(maxAgeSeconds)*time.Second {
		return "MFA was verified too long ago to complete this request"
	}
	return ""
}

func getApiKeyInfo(req *medusahttp.MedusaRequest) *dal.Record {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return nil
	}
	parts := strings.Split(authHeader, " ")
	tokenType := parts[0]
	token := ""
	if len(parts) > 1 {
		token = parts[1]
	}
	if strings.ToLower(tokenType) != "basic" || token == "" {
		return nil
	}
	normalized := token
	if !strings.HasPrefix(token, "sk_") {
		b, _ := base64.StdEncoding.DecodeString(padBase64(token))
		normalized = string(b)
	}
	normalized = strings.TrimSuffix(normalized, ":")
	if !strings.HasPrefix(normalized, "sk_") {
		return nil
	}
	svc, ok := req.Scope.Resolve(ApiKeyModuleKey).(ApiKeyAuthenticator)
	if !ok {
		return nil
	}
	apiKey, err := svc.Authenticate(req.Ctx(), normalized)
	if err != nil {
		slog.Error(err.Error())
		return nil
	}
	return apiKey
}

// padBase64 makes Node's lenient base64 decoding work with Go's decoder.
func padBase64(s string) string {
	s = strings.TrimRight(s, "=")
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	return s
}

func isBearerSecretApiKey(authHeader string) bool {
	if authHeader == "" {
		return false
	}
	parts := strings.Split(authHeader, " ")
	return strings.ToLower(parts[0]) == bearerAuth && len(parts) > 1 && strings.HasPrefix(parts[1], "sk_")
}

func getAuthContextFromSession(session *medusahttp.Session, authTypes, actorTypes []string) *dal.Record {
	if !contains(authTypes, sessionAuth) || session == nil {
		return nil
	}
	v, ok := session.Get("auth_context")
	if !ok {
		return nil
	}
	ctx, ok := v.(*dal.Record)
	if !ok {
		return nil
	}
	if isActorTypePermitted(actorTypes, stringOf(ctx, "actor_type")) {
		return ctx
	}
	return nil
}

var bearerRe = regexp.MustCompile(`(\S+)\s+(\S+)`)

// GetAuthContextFromJwtToken mirrors getAuthContextFromJwtToken.
func GetAuthContextFromJwtToken(authHeader, secret string, authTypes, actorTypes []string) *dal.Record {
	if !contains(authTypes, bearerAuth) || authHeader == "" {
		return nil
	}
	m := bearerRe.FindStringSubmatch(authHeader)
	if m == nil || strings.ToLower(m[1]) != bearerAuth {
		return nil
	}
	payload, err := auth.VerifyJwtToken(m[2], secret)
	if err != nil {
		slog.Debug("Failed to verify JWT token: " + err.Error())
		return nil
	}
	if isActorTypePermitted(actorTypes, stringOf(payload, "actor_type")) {
		return payload
	}
	return nil
}
