import { betterAuth } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { admin, openAPI, organization, twoFactor } from "better-auth/plugins";
import { db } from "./db.js";
import * as schema from "./schema/index.js";
import { env } from "./env.js";
import { syncOrgToZenv, syncMemberToZenv } from "./hooks.js";

const trustedOrigins = env.TRUSTED_ORIGINS
  ? env.TRUSTED_ORIGINS.split(",").map((s) => s.trim())
  : [];

export const auth = betterAuth({
  database: drizzleAdapter(db, { provider: "pg", schema: schema }),
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
      hooks: {
        organization: {
          afterCreate: async (ctx: {
            organization: { id: string; name: string };
            member: { userId: string };
          }) => {
            await syncOrgToZenv(ctx.organization, ctx.member);
          },
        },
        member: {
          afterCreate: async (ctx: {
            member: { userId: string; role: string };
            organization: { id: string };
          }) => {
            await syncMemberToZenv(ctx.member, ctx.organization);
          },
        },
      },
    }),
  ],
  advanced: {
    crossSubDomainCookies: {
      enabled: !!env.COOKIE_DOMAIN,
      domain: env.COOKIE_DOMAIN,
    },
  },
});
