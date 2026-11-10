package commands

import (
	"context"
	"fmt"
	"log/slog"

	inviteworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/invite/workflows"
	userworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/user/workflows"
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	authtypes "github.com/hitkey-io/medusago/packages/core/types/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	authservices "github.com/hitkey-io/medusago/packages/modules/auth/src/services"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// UserOptions mirrors the options of `medusa user`.
type UserOptions struct {
	Email    string
	Password string
	ID       string
	Invite   bool
	Config   config.ConfigModule
	// Directory and OnLoad are passed to the loaders.
	Directory string
	OnLoad    []func(ctx context.Context, app *loaders.MedusaApp) error
}

// User mirrors commands/user.ts: an invite (its token is logged) or a user
// with an emailpass auth identity. RBAC roles are Enterprise and skipped.
func User(opts UserOptions) error {
	ctx := context.Background()
	app, err := loaders.InitializeWithOptions(ctx, opts.Config, loaders.Options{Directory: opts.Directory, SkipAdmin: true, OnLoad: opts.OnLoad})
	if err != nil {
		return err
	}
	defer app.Shutdown()
	c := app.Container

	email := dal.NewRecord()
	email.Set("email", opts.Email)

	if opts.Invite {
		invites, err := inviteworkflows.CreateInvitesWorkflow.Run(ctx, c, inviteworkflows.CreateInvitesWorkflowInput{Invites: []*dal.Record{email}})
		if err != nil {
			return err
		}
		token, _ := invites[0].Get("token")
		slog.Info(fmt.Sprintf("\n      Invite token: %v\n      Open the invite in Medusa Admin at: [your-admin-url]/invite?token=%v", token, token))
		return nil
	}

	users, err := userworkflows.CreateUsersWorkflow.Run(ctx, c, userworkflows.CreateUsersWorkflowInput{Users: []*dal.Record{email}})
	if err != nil {
		return err
	}
	userID, _ := users[0].Get("id")

	auth, _ := c.Resolve(utilsmodulessdk.AUTH).(*authservices.AuthModuleService)
	body := dal.NewRecord()
	body.Set("email", opts.Email)
	body.Set("password", opts.Password)
	res := auth.Register(ctx, "emailpass", authtypes.AuthenticationInput{Body: body})
	if res.Error != "" {
		return fmt.Errorf("%s", res.Error)
	}
	identityID, _ := res.AuthIdentity.Get("id")
	meta := dal.NewRecord()
	meta.Set("user_id", userID)
	update := dal.NewRecord()
	update.Set("id", identityID)
	update.Set("app_metadata", meta)
	if _, err := auth.Internal("AuthIdentity").Update(ctx, []any{update}); err != nil {
		return err
	}
	slog.Info("User created successfully.")
	return nil
}
