// Package migrationexamples holds hand-written migrations for the chapter on
// writing migrations. They are not part of the blog module.
package migrationexamples

// region: imports
import (
	"context"

	"github.com/jackc/pgx/v5"

	m "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// endregion: imports

// region: add-column
// AddSubtitle adds a nullable column to the post table.
var AddSubtitle = m.Migration{
	Name: "Migration20261115090000",
	Up: []string{
		`alter table if exists "post" add column if not exists "subtitle" text null;`,
	},
	Down: []string{
		`alter table if exists "post" drop column if exists "subtitle";`,
	},
}

// endregion: add-column

// region: rename-column
// RenameContent renames post.content to post.body.
var RenameContent = m.Migration{
	Name: "Migration20261115091000",
	Up: []string{
		`alter table if exists "post" rename column "content" to "body";`,
	},
	Down: []string{
		`alter table if exists "post" rename column "body" to "content";`,
	},
}

// endregion: rename-column

// region: drop-column
// DropAuthorID removes post.author_id. The Down statements recreate the
// column, without its data.
var DropAuthorID = m.Migration{
	Name: "Migration20261115092000",
	Up: []string{
		`alter table if exists "post" drop column if exists "author_id";`,
	},
	Down: []string{
		`alter table if exists "post" add column if not exists "author_id" text null;`,
	},
}

// endregion: drop-column

// region: create-index
// IndexAuthor creates an index on post.author_id for the posts that are not
// soft-deleted.
var IndexAuthor = m.Migration{
	Name: "Migration20261115093000",
	Up: []string{
		`CREATE INDEX IF NOT EXISTS "IDX_post_author_id" ON "post" ("author_id") WHERE deleted_at IS NULL;`,
	},
	Down: []string{
		`DROP INDEX IF EXISTS "IDX_post_author_id";`,
	},
}

// endregion: create-index

// region: relationship
// CreateReaction adds a table with a foreign key to post.
var CreateReaction = m.Migration{
	Name: "Migration20261115094000",
	Up: []string{
		`create table if not exists "reaction" ("id" text not null, "emoji" text not null, "post_id" text not null, "created_at" timestamptz not null default now(), "updated_at" timestamptz not null default now(), "deleted_at" timestamptz null, constraint "reaction_pkey" primary key ("id"));`,
		`CREATE INDEX IF NOT EXISTS "IDX_reaction_post_id" ON "reaction" ("post_id") WHERE deleted_at IS NULL;`,
		`CREATE INDEX IF NOT EXISTS "IDX_reaction_deleted_at" ON "reaction" ("deleted_at") WHERE deleted_at IS NULL;`,
		`alter table if exists "reaction" add constraint "reaction_post_id_foreign" foreign key ("post_id") references "post" ("id") on update cascade on delete cascade;`,
	},
	Down: []string{
		`alter table if exists "reaction" drop constraint if exists "reaction_post_id_foreign";`,
		`drop table if exists "reaction" cascade;`,
	},
}

// endregion: relationship

// region: data-migration
// PublishOldDrafts changes the data, not the schema. UpFunc runs in the
// migration's transaction instead of the Up statements.
var PublishOldDrafts = m.Migration{
	Name: "Migration20261115095000",
	UpFunc: func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update "post" set "status" = 'archived' where "status" = 'draft' and "created_at" < now() - interval '1 year'`)
		return err
	},
	// without DownFunc or Down, the migration cannot be rolled back
}

// endregion: data-migration
