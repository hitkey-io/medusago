// Package services mirrors @medusajs/user/src/services.
package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/user/src/models"
)

// DefaultValidInviteDurationSeconds mirrors DEFAULT_VALID_INVITE_DURATION_SECONDS.
const DefaultValidInviteDurationSeconds = 60 * 60 * 24

// UserModuleService mirrors UserModuleService.
type UserModuleService struct {
	*utilsmodulessdk.MedusaService
	container modulessdk.Resolver
}

// NewUserModuleService builds the service.
func NewUserModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*UserModuleService, error) {
	return &UserModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig), container: opts.Container}, nil
}

func (s *UserModuleService) jwtSecret() string {
	if s.container != nil {
		if cfg, ok := s.container.Resolve(common.ContainerRegistrationKeys.CONFIG_MODULE).(config.ConfigModule); ok {
			return cfg.ProjectConfig.HTTP.JwtSecret
		}
	}
	return "supersecret"
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func uuid4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// generateToken mirrors generateToken: jsonwebtoken.sign({ id, email },
// secret, { expiresIn, jwtid }).
func (s *UserModuleService) generateToken(id, email any) (string, error) {
	p := dal.NewRecord()
	p.Set("id", id)
	p.Set("email", email)
	return auth.GenerateJwtTokenWithID(p, s.jwtSecret(), fmt.Sprint(DefaultValidInviteDurationSeconds), uuid4())
}

func expiresAt() time.Time {
	return time.Now().UTC().Add(DefaultValidInviteDurationSeconds * time.Second)
}

// CreateInvites mirrors createInvites.
func (s *UserModuleService) CreateInvites(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	emails := make([]any, len(data))
	for i, d := range data {
		emails[i] = get(d, "email")
	}
	f := dal.NewRecord()
	f.Set("email", emails)
	existing, err := s.Internal("User").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		list := make([]string, len(existing))
		for i, u := range existing {
			list[i] = fmt.Sprint(get(u, "email"))
		}
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "User account for following email(s) already exist: "+strings.Join(list, ", "))
	}
	exp := expiresAt()
	toCreate := make([]*dal.Record, len(data))
	for i, d := range data {
		id := common.GenerateEntityID(fmt.Sprint(get(d, "id")), "invite")
		if get(d, "id") == nil {
			id = common.GenerateEntityID("", "invite")
		}
		tok, err := s.generateToken(id, get(d, "email"))
		if err != nil {
			return nil, err
		}
		n := dal.NewRecord()
		for _, k := range d.Keys() {
			if k == "roles" {
				continue
			}
			v, _ := d.Get(k)
			n.Set(k, v)
		}
		n.Set("id", id)
		n.Set("expires_at", exp)
		n.Set("token", tok)
		toCreate[i] = n
	}
	created, err := s.Internal("Invite").Create(ctx, toCreate)
	if err != nil {
		return nil, err
	}
	s.inviteTokenGenerated(ctx, "token_generated", created)
	return created, nil
}

// inviteTokenGenerated mirrors the moduleEventBuilderFactory call
// emitting UserEvents.INVITE_TOKEN_GENERATED for the invites.
func (s *UserModuleService) inviteTokenGenerated(ctx context.Context, action string, invites []*dal.Record) {
	ids := make([]any, len(invites))
	for i, inv := range invites {
		ids[i] = get(inv, "id")
	}
	s.SaveEvent(ctx, utilsmodulessdk.ModuleEventBuilderFactory(utilsmodulessdk.USER, action, "invite", "user.user.invite.token_generated")(ids...))
}

// RefreshInviteTokens mirrors refreshInviteTokens.
func (s *UserModuleService) RefreshInviteTokens(ctx context.Context, ids []any) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", ids)
	invites, err := s.Internal("Invite").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if len(invites) != len(ids) {
		found := map[string]bool{}
		for _, inv := range invites {
			found[fmt.Sprint(get(inv, "id"))] = true
		}
		var missing []string
		for _, id := range ids {
			if !found[fmt.Sprint(id)] {
				missing = append(missing, fmt.Sprint(id))
			}
		}
		if len(missing) > 0 {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "The following invites do not exist: "+strings.Join(missing, ", "))
		}
	}
	exp := expiresAt()
	updates := make([]any, len(invites))
	for i, inv := range invites {
		tok, err := s.generateToken(get(inv, "id"), get(inv, "email"))
		if err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		u.Set("id", get(inv, "id"))
		u.Set("expires_at", exp)
		u.Set("token", tok)
		updates[i] = u
	}
	res, err := s.Internal("Invite").Update(ctx, updates)
	if err != nil {
		return nil, err
	}
	out, _ := res.([]*dal.Record)
	if r, ok := res.(*dal.Record); ok {
		out = []*dal.Record{r}
	}
	s.inviteTokenGenerated(ctx, "created", out)
	return out, nil
}

// ValidateInviteToken mirrors validateInviteToken.
func (s *UserModuleService) ValidateInviteToken(ctx context.Context, token string) (*dal.Record, error) {
	payload, err := auth.VerifyJwtToken(token, s.jwtSecret())
	if err != nil {
		return nil, err
	}
	invite, err := s.Internal("Invite").Retrieve(ctx, get(payload, "id"), utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if fmt.Sprint(get(invite, "token")) != token {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "The invite token is invalid")
	}
	if at, ok := get(invite, "expires_at").(time.Time); ok && at.Before(time.Now()) {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "The invite has expired")
	}
	return invite, nil
}
