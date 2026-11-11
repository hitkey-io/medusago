const { Migration } = require("@medusajs/framework/mikro-orm/migrations");

class Migration20261111064001 extends Migration {

  async up() {
    this.addSql(`alter table if exists "brand_profile" drop constraint if exists "brand_profile_brand_id_unique";`);
    this.addSql(`alter table if exists "brand" drop constraint if exists "brand_handle_unique";`);
    this.addSql(`create table if not exists "brand" ("id" text not null, "name" text not null, "handle" text not null, "description" text null, "is_active" boolean not null default true, "rank" integer not null default 0, "rating" real null, "revenue" numeric null, "status" text check ("status" in ('draft', 'published', 'archived')) not null default 'draft', "tags" text[] not null default '{}', "currency_code" text null, "metadata" jsonb null, "launched_at" timestamptz null, "raw_revenue" jsonb null, "created_at" timestamptz not null default now(), "updated_at" timestamptz not null default now(), "deleted_at" timestamptz null, constraint "brand_pkey" primary key ("id"));`);
    this.addSql(`CREATE UNIQUE INDEX IF NOT EXISTS "IDX_brand_handle_unique" ON "brand" ("handle") WHERE deleted_at IS NULL;`);
    this.addSql(`CREATE INDEX IF NOT EXISTS "IDX_brand_deleted_at" ON "brand" ("deleted_at") WHERE deleted_at IS NULL;`);
    this.addSql(`CREATE INDEX IF NOT EXISTS "IDX_brand_name_status" ON "brand" ("name", "status") WHERE deleted_at IS NULL;`);

    this.addSql(`create table if not exists "brand_logo" ("id" text not null, "url" text not null, "position" integer not null default 0, "brand_id" text not null, "created_at" timestamptz not null default now(), "updated_at" timestamptz not null default now(), "deleted_at" timestamptz null, constraint "brand_logo_pkey" primary key ("id"));`);
    this.addSql(`CREATE INDEX IF NOT EXISTS "IDX_brand_logo_brand_id" ON "brand_logo" ("brand_id") WHERE deleted_at IS NULL;`);
    this.addSql(`CREATE INDEX IF NOT EXISTS "IDX_brand_logo_deleted_at" ON "brand_logo" ("deleted_at") WHERE deleted_at IS NULL;`);

    this.addSql(`create table if not exists "brand_profile" ("id" text not null, "website" text null, "brand_id" text not null, "created_at" timestamptz not null default now(), "updated_at" timestamptz not null default now(), "deleted_at" timestamptz null, constraint "brand_profile_pkey" primary key ("id"));`);
    this.addSql(`CREATE UNIQUE INDEX IF NOT EXISTS "IDX_brand_profile_brand_id_unique" ON "brand_profile" ("brand_id") WHERE deleted_at IS NULL;`);
    this.addSql(`CREATE INDEX IF NOT EXISTS "IDX_brand_profile_deleted_at" ON "brand_profile" ("deleted_at") WHERE deleted_at IS NULL;`);

    this.addSql(`create table if not exists "brand_stat" ("brand_id" text not null, "period" text not null, "views" integer not null default 0, "score" real null, "created_at" timestamptz not null default now(), "updated_at" timestamptz not null default now(), "deleted_at" timestamptz null, constraint "brand_stat_pkey" primary key ("brand_id", "period"));`);
    this.addSql(`CREATE INDEX IF NOT EXISTS "IDX_brand_stat_deleted_at" ON "brand_stat" ("deleted_at") WHERE deleted_at IS NULL;`);

    this.addSql(`alter table if exists "brand_logo" add constraint "brand_logo_brand_id_foreign" foreign key ("brand_id") references "brand" ("id") on update cascade on delete cascade;`);

    this.addSql(`alter table if exists "brand_profile" add constraint "brand_profile_brand_id_foreign" foreign key ("brand_id") references "brand" ("id") on update cascade on delete cascade;`);
  }

  async down() {
    this.addSql(`alter table if exists "brand_logo" drop constraint if exists "brand_logo_brand_id_foreign";`);

    this.addSql(`alter table if exists "brand_profile" drop constraint if exists "brand_profile_brand_id_foreign";`);

    this.addSql(`drop table if exists "brand" cascade;`);

    this.addSql(`drop table if exists "brand_logo" cascade;`);

    this.addSql(`drop table if exists "brand_profile" cascade;`);

    this.addSql(`drop table if exists "brand_stat" cascade;`);
  }

}

exports.Migration20261111064001 = Migration20261111064001;
