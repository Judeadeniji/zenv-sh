import { redisStorage } from "@better-auth/redis-storage"
import { betterAuth } from "better-auth"
import { drizzleAdapter } from "better-auth/adapters/drizzle"
import { admin, openAPI, organization, twoFactor } from "better-auth/plugins"
import { db } from "./db.js"
import * as schema from "./schema/index.js"
import { env } from "./env.js"
import { redis } from "./redis.js"

const trustedOrigins = env.TRUSTED_ORIGINS
	? env.TRUSTED_ORIGINS.split(",").map((s) => s.trim())
	: []

export const auth = betterAuth({
	appName: "zEnv",
	database: drizzleAdapter(db, {
		provider: "pg",
		schema: schema,
		usePlural: true,
		transaction: true,
	}),
	secondaryStorage: redisStorage({
		client: redis,
		keyPrefix: "better-auth:",
	}),
	emailAndPassword: {
		enabled: true,
	},
	socialProviders: {
		...(env.GITHUB_CLIENT_ID && env.GITHUB_CLIENT_SECRET
			? {
					github: {
						clientId: env.GITHUB_CLIENT_ID,
						clientSecret: env.GITHUB_CLIENT_SECRET,
					},
				}
			: {}),
		...(env.GOOGLE_CLIENT_ID && env.GOOGLE_CLIENT_SECRET
			? {
					google: {
						clientId: env.GOOGLE_CLIENT_ID,
						clientSecret: env.GOOGLE_CLIENT_SECRET,
					},
				}
			: {}),
	},
	trustedOrigins,
	session: {
		expiresIn: 60 * 60 * 24, // 24 hours (seconds)
		updateAge: 60 * 60, // refresh every hour
		cookieCache: {
			enabled: true,
			maxAge: 60 * 5, // 5 min cookie cache
		},
	},
	plugins: [
		openAPI(),
		admin({
			defaultRole: "user",
			adminRole: "admin",
		}),
		twoFactor(),
		organization({
			allowUserToCreateOrganization: true,
			organizationLimit: 5,
			membershipLimit: 100,
			schema: {
				organization: {
					additionalFields: {
						ownerId: {
							type: "string",
							fieldName: "owner_id",
							required: false,
							references: {
								model: "users",
								field: "id",
								onDelete: "cascade",
							},
						},
					},
				},
			},
			sendInvitationEmail: async (data) => {
				// TODO: wire up email provider (Resend, Postmark, etc.)
				// data.invitation.id  — the invitation ID used in the join URL
				// data.invitation.email — the invitee's email address
				// data.organization.name — the org they're being invited to
				// data.inviter.user.email — the person who sent the invite
				// const url = `${env.APP_URL}/join/${data.invitation.id}`
				const url = new URL(`/join/${data.invitation.id}`, env.APP_URL)
				console.log(
					`[invite] ${data.inviter.user.email} → ${data.invitation.email}` +
						` (${data.organization.name}) — ${url}`,
				)
			},
			organizationHooks: {
				beforeCreateOrganization: async ({ organization, user }) => {
					return {
						data: {
							...organization,
							ownerId: user.id,
						},
					}
				},
			},
		}),
	],
	advanced: {
		crossSubDomainCookies: {
			enabled: !!env.COOKIE_DOMAIN,
			domain: env.COOKIE_DOMAIN,
		},
		database: {
			generateId: "uuid",
		},
	},
})
