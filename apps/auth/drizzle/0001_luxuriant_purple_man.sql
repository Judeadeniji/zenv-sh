CREATE TYPE "public"."rotation_status" AS ENUM('staging', 'committing', 'complete', 'failed');--> statement-breakpoint
CREATE TYPE "public"."environment" AS ENUM('development', 'staging', 'production');--> statement-breakpoint
CREATE TYPE "public"."permission" AS ENUM('read', 'read_write');--> statement-breakpoint
CREATE TYPE "public"."recovery_status" AS ENUM('pending', 'approved', 'cancelled', 'expired', 'completed');--> statement-breakpoint
CREATE TYPE "public"."vault_key_type" AS ENUM('pin', 'passphrase');--> statement-breakpoint
CREATE TABLE "accounts" (
	"id" uuid PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid() NOT NULL,
	"account_id" text NOT NULL,
	"provider_id" text NOT NULL,
	"user_id" uuid NOT NULL,
	"access_token" text,
	"refresh_token" text,
	"id_token" text,
	"access_token_expires_at" timestamp,
	"refresh_token_expires_at" timestamp,
	"scope" text,
	"password" text,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp NOT NULL
);
--> statement-breakpoint
CREATE TABLE "invitations" (
	"id" uuid PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"email" text NOT NULL,
	"role" text,
	"status" text DEFAULT 'pending' NOT NULL,
	"expires_at" timestamp NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"inviter_id" uuid NOT NULL
);
--> statement-breakpoint
CREATE TABLE "members" (
	"id" uuid PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"user_id" uuid NOT NULL,
	"role" text DEFAULT 'member' NOT NULL,
	"created_at" timestamp NOT NULL
);
--> statement-breakpoint
CREATE TABLE "organizations" (
	"id" uuid PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid() NOT NULL,
	"name" text NOT NULL,
	"slug" text NOT NULL,
	"logo" text,
	"created_at" timestamp NOT NULL,
	"metadata" text,
	"owner_id" uuid,
	CONSTRAINT "organizations_slug_unique" UNIQUE("slug")
);
--> statement-breakpoint
CREATE TABLE "sessions" (
	"id" uuid PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid() NOT NULL,
	"expires_at" timestamp NOT NULL,
	"token" text NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp NOT NULL,
	"ip_address" text,
	"user_agent" text,
	"user_id" uuid NOT NULL,
	"impersonated_by" text,
	"active_organization_id" text,
	CONSTRAINT "sessions_token_unique" UNIQUE("token")
);
--> statement-breakpoint
CREATE TABLE "two_factors" (
	"id" uuid PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid() NOT NULL,
	"secret" text NOT NULL,
	"backup_codes" text NOT NULL,
	"user_id" uuid NOT NULL
);
--> statement-breakpoint
CREATE TABLE "users" (
	"id" uuid PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid() NOT NULL,
	"name" text NOT NULL,
	"email" text NOT NULL,
	"email_verified" boolean DEFAULT false NOT NULL,
	"image" text,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp DEFAULT now() NOT NULL,
	"role" text,
	"banned" boolean DEFAULT false,
	"ban_reason" text,
	"ban_expires" timestamp,
	"two_factor_enabled" boolean DEFAULT false,
	CONSTRAINT "users_email_unique" UNIQUE("email")
);
--> statement-breakpoint
CREATE TABLE "verifications" (
	"id" uuid PRIMARY KEY DEFAULT pg_catalog.gen_random_uuid() NOT NULL,
	"identifier" text NOT NULL,
	"value" text NOT NULL,
	"expires_at" timestamp NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "project_key_grants" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"project_id" uuid NOT NULL,
	"user_id" uuid NOT NULL,
	"wrapped_project_vault_key" "bytea" NOT NULL,
	"granted_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "project_key_grants_project_id_user_id_key" UNIQUE("project_id","user_id")
);
--> statement-breakpoint
CREATE TABLE "project_rotations" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"project_id" uuid NOT NULL,
	"rotation_id" uuid NOT NULL,
	"status" "rotation_status" DEFAULT 'staging' NOT NULL,
	"total_items" integer NOT NULL,
	"staged_items" integer DEFAULT 0 NOT NULL,
	"initiated_by" uuid NOT NULL,
	"started_at" timestamp with time zone DEFAULT now() NOT NULL,
	"completed_at" timestamp with time zone,
	CONSTRAINT "project_rotations_rotation_id_key" UNIQUE("rotation_id")
);
--> statement-breakpoint
CREATE TABLE "project_vault_keys" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"project_id" uuid NOT NULL,
	"project_salt" "bytea" NOT NULL,
	"wrapped_project_dek" "bytea" NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"dek_version" integer DEFAULT 1 NOT NULL,
	CONSTRAINT "project_vault_keys_project_id_key" UNIQUE("project_id")
);
--> statement-breakpoint
CREATE TABLE "projects" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" uuid NOT NULL,
	"name" text NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "projects_organization_id_name_key" UNIQUE("organization_id","name")
);
--> statement-breakpoint
CREATE TABLE "identities" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"auth_key_hash" "bytea" NOT NULL,
	"salt" "bytea" NOT NULL,
	"vault_key_type" "vault_key_type" DEFAULT 'passphrase' NOT NULL,
	"wrapped_dek" "bytea" NOT NULL,
	"public_key" "bytea" NOT NULL,
	"wrapped_private_key" "bytea" NOT NULL,
	"recovery_wrapped_dek" "bytea",
	"recovery_disabled" boolean DEFAULT false NOT NULL,
	"identity_id" uuid,
	"preferences" jsonb DEFAULT '{}'::jsonb NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "identities_external_id_key" UNIQUE("identity_id")
);
--> statement-breakpoint
CREATE TABLE "recovery_requests" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"user_id" uuid NOT NULL,
	"contact_user_id" uuid NOT NULL,
	"status" "recovery_status" DEFAULT 'pending' NOT NULL,
	"recovery_public_key" "bytea",
	"recovery_payload" "bytea",
	"requested_at" timestamp with time zone DEFAULT now() NOT NULL,
	"eligible_at" timestamp with time zone NOT NULL,
	"approved_at" timestamp with time zone,
	"completed_at" timestamp with time zone,
	"cancelled_at" timestamp with time zone
);
--> statement-breakpoint
CREATE TABLE "service_tokens" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"project_id" uuid NOT NULL,
	"name" text NOT NULL,
	"token_hash" "bytea" NOT NULL,
	"environment" "environment" NOT NULL,
	"permission" "permission" DEFAULT 'read' NOT NULL,
	"created_by" uuid,
	"expires_at" timestamp with time zone,
	"revoked_at" timestamp with time zone,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "service_tokens_token_hash_key" UNIQUE("token_hash")
);
--> statement-breakpoint
CREATE TABLE "trusted_contacts" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"user_id" uuid NOT NULL,
	"contact_user_id" uuid NOT NULL,
	"trusted_wrapped_dek" "bytea" NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "trusted_contacts_user_id_contact_user_id_key" UNIQUE("user_id","contact_user_id")
);
--> statement-breakpoint
CREATE TABLE "vault_item_rotations" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"project_id" uuid NOT NULL,
	"rotation_id" uuid NOT NULL,
	"vault_item_id" uuid NOT NULL,
	"new_ciphertext" "bytea" NOT NULL,
	"new_nonce" "bytea" NOT NULL,
	"staged_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "vault_item_versions" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"item_id" uuid NOT NULL,
	"version" integer NOT NULL,
	"ciphertext" "bytea" NOT NULL,
	"nonce" "bytea" NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "vault_items" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"project_id" uuid NOT NULL,
	"environment" "environment" NOT NULL,
	"name_hash" "bytea" NOT NULL,
	"ciphertext" "bytea" NOT NULL,
	"nonce" "bytea" NOT NULL,
	"version" integer DEFAULT 1 NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	"dek_version" integer DEFAULT 1 NOT NULL
);
--> statement-breakpoint
ALTER TABLE "accounts" ADD CONSTRAINT "accounts_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "invitations" ADD CONSTRAINT "invitations_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "invitations" ADD CONSTRAINT "invitations_inviter_id_users_id_fk" FOREIGN KEY ("inviter_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "members" ADD CONSTRAINT "members_organization_id_organizations_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "members" ADD CONSTRAINT "members_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "organizations" ADD CONSTRAINT "organizations_owner_id_users_id_fk" FOREIGN KEY ("owner_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "sessions" ADD CONSTRAINT "sessions_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "two_factors" ADD CONSTRAINT "two_factors_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "project_key_grants" ADD CONSTRAINT "project_key_grants_project_id_fkey" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "project_key_grants" ADD CONSTRAINT "project_key_grants_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "project_rotations" ADD CONSTRAINT "project_rotations_project_id_fkey" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "project_rotations" ADD CONSTRAINT "project_rotations_initiated_by_fkey" FOREIGN KEY ("initiated_by") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "project_vault_keys" ADD CONSTRAINT "project_vault_keys_project_id_fkey" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "projects" ADD CONSTRAINT "projects_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "identities" ADD CONSTRAINT "identities_identity_id_users_id_fk" FOREIGN KEY ("identity_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "recovery_requests" ADD CONSTRAINT "recovery_requests_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "recovery_requests" ADD CONSTRAINT "recovery_requests_contact_user_id_fkey" FOREIGN KEY ("contact_user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "service_tokens" ADD CONSTRAINT "service_tokens_project_id_fkey" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "service_tokens" ADD CONSTRAINT "service_tokens_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "trusted_contacts" ADD CONSTRAINT "trusted_contacts_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "trusted_contacts" ADD CONSTRAINT "trusted_contacts_contact_user_id_fkey" FOREIGN KEY ("contact_user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "vault_item_rotations" ADD CONSTRAINT "vault_item_rotations_project_id_fkey" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "vault_item_rotations" ADD CONSTRAINT "vault_item_rotations_rotation_id_fkey" FOREIGN KEY ("rotation_id") REFERENCES "public"."project_rotations"("rotation_id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "vault_item_rotations" ADD CONSTRAINT "vault_item_rotations_vault_item_id_fkey" FOREIGN KEY ("vault_item_id") REFERENCES "public"."vault_items"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "vault_item_versions" ADD CONSTRAINT "vault_item_versions_item_id_fkey" FOREIGN KEY ("item_id") REFERENCES "public"."vault_items"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "vault_items" ADD CONSTRAINT "vault_items_project_id_fkey" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "accounts_userId_idx" ON "accounts" USING btree ("user_id");--> statement-breakpoint
CREATE INDEX "invitations_organizationId_idx" ON "invitations" USING btree ("organization_id");--> statement-breakpoint
CREATE INDEX "invitations_email_idx" ON "invitations" USING btree ("email");--> statement-breakpoint
CREATE INDEX "members_organizationId_idx" ON "members" USING btree ("organization_id");--> statement-breakpoint
CREATE INDEX "members_userId_idx" ON "members" USING btree ("user_id");--> statement-breakpoint
CREATE UNIQUE INDEX "organizations_slug_uidx" ON "organizations" USING btree ("slug");--> statement-breakpoint
CREATE INDEX "sessions_userId_idx" ON "sessions" USING btree ("user_id");--> statement-breakpoint
CREATE INDEX "twoFactors_secret_idx" ON "two_factors" USING btree ("secret");--> statement-breakpoint
CREATE INDEX "twoFactors_userId_idx" ON "two_factors" USING btree ("user_id");--> statement-breakpoint
CREATE INDEX "verifications_identifier_idx" ON "verifications" USING btree ("identifier");--> statement-breakpoint
CREATE INDEX "idx_project_rotations_project" ON "project_rotations" USING btree ("project_id");--> statement-breakpoint
CREATE UNIQUE INDEX "idx_recovery_requests_active" ON "recovery_requests" USING btree ("user_id") WHERE (status = ANY (ARRAY['pending'::recovery_status, 'approved'::recovery_status]));--> statement-breakpoint
CREATE INDEX "idx_recovery_requests_contact" ON "recovery_requests" USING btree ("contact_user_id") WHERE (status = ANY (ARRAY['pending'::recovery_status, 'approved'::recovery_status]));--> statement-breakpoint
CREATE INDEX "idx_service_tokens_hash" ON "service_tokens" USING btree ("token_hash") WHERE (revoked_at IS NULL);--> statement-breakpoint
CREATE INDEX "idx_service_tokens_project" ON "service_tokens" USING btree ("project_id");--> statement-breakpoint
CREATE INDEX "idx_trusted_contacts_contact" ON "trusted_contacts" USING btree ("contact_user_id");--> statement-breakpoint
CREATE INDEX "idx_trusted_contacts_user" ON "trusted_contacts" USING btree ("user_id");--> statement-breakpoint
CREATE INDEX "idx_vault_item_rotations_rotation" ON "vault_item_rotations" USING btree ("rotation_id");--> statement-breakpoint
CREATE INDEX "idx_vault_item_versions_item" ON "vault_item_versions" USING btree ("item_id","version");--> statement-breakpoint
CREATE UNIQUE INDEX "idx_vault_items_lookup" ON "vault_items" USING btree ("project_id","environment","name_hash");--> statement-breakpoint
CREATE INDEX "idx_vault_items_project_env" ON "vault_items" USING btree ("project_id","environment");