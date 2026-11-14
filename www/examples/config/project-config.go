// Package configexamples holds the configuration snippets of the MedusaGo
// documentation. They are compile-checked against the repository.
package configexamples

import (
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
)

// ProjectConfigExample sets every supported projectConfig option.
func ProjectConfigExample() config.InputConfig {
	// region: project-config
	return config.InputConfig{
		ProjectConfig: config.ProjectConfig{
			DatabaseURL: os.Getenv("DATABASE_URL"),
			RedisURL:    os.Getenv("REDIS_URL"),
			RedisPrefix: "medusa:",
			HTTP: config.HttpConfig{
				StoreCors:           os.Getenv("STORE_CORS"),
				AdminCors:           os.Getenv("ADMIN_CORS"),
				AuthCors:            os.Getenv("AUTH_CORS"),
				JwtSecret:           os.Getenv("JWT_SECRET"),
				CookieSecret:        os.Getenv("COOKIE_SECRET"),
				JwtExpiresIn:        "7d",
				StoreRelationsLimit: 3,
			},
		},
	}
	// endregion: project-config
}

// CorsExample lists several origins and an origin pattern.
func CorsExample() config.HttpConfig {
	// region: cors
	return config.HttpConfig{
		StoreCors: "http://localhost:8000,https://storefront.example.com",
		AdminCors: "http://localhost:9000,/vercel\\.app$/",
		AuthCors:  "http://localhost:8000,http://localhost:9000",
	}
	// endregion: cors
}

// AuthMethodsExample allows the emailpass and Google providers for admin
// users and the emailpass provider for customers.
func AuthMethodsExample() config.HttpConfig {
	// region: auth-methods
	return config.HttpConfig{
		AuthMethodsPerActor: map[string][]string{
			"user":     {"emailpass", "google"},
			"customer": {"emailpass"},
		},
	}
	// endregion: auth-methods
}

// SessionExample stores the sessions in Redis and tunes the cookie.
func SessionExample() config.ProjectConfig {
	// region: session
	return config.ProjectConfig{
		RedisURL:    os.Getenv("REDIS_URL"),
		RedisPrefix: "medusa:",
		SessionOptions: &config.SessionOptions{
			Name:   "connect.sid",
			Secret: os.Getenv("COOKIE_SECRET"),
			// Ttl is the lifetime of the session cookie in milliseconds.
			Ttl:     new(int64(10 * 60 * 60 * 1000)),
			Resave:  new(true),
			Rolling: new(false),
		},
		CookieOptions: &config.CookieOptions{
			SameSite: "lax",
			Secure:   new(true),
			Domain:   ".example.com",
		},
	}
	// endregion: session
}

// DatabaseURLExample sets the database URL directly.
func DatabaseURLExample() config.ProjectConfig {
	// region: database-url
	return config.ProjectConfig{
		DatabaseURL: "postgres://postgres:postgres@localhost/medusa-store",
	}
	// endregion: database-url
}

// LocalProductionCookies lets the session cookie work over plain HTTP, to try
// a production build on localhost.
func LocalProductionCookies() config.InputConfig {
	// region: local-cookies
	return config.InputConfig{
		ProjectConfig: config.ProjectConfig{
			// ...
			CookieOptions: &config.CookieOptions{
				SameSite: "lax",
				Secure:   new(false),
			},
		},
	}
	// endregion: local-cookies
}

// RedisSessionExample stores the sessions in Redis.
func RedisSessionExample() config.ProjectConfig {
	// region: redis-session
	return config.ProjectConfig{
		RedisURL: os.Getenv("REDIS_URL"),
	}
	// endregion: redis-session
}
