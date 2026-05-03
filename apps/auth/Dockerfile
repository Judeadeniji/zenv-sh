FROM node:22-alpine AS builder

WORKDIR /build

# Install dependencies first for caching.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/auth/package.json ./apps/auth/
COPY apps/docs/package.json ./apps/docs/
COPY apps/web/package.json ./apps/web/
COPY packages/amnesia/package.json ./packages/amnesia/
COPY packages/sdk/package.json ./packages/sdk/

RUN corepack enable pnpm && pnpm install --frozen-lockfile

# Copy source.
COPY apps/auth/ ./apps/auth/

RUN cd apps/auth && pnpm build

# ---

FROM node:22-alpine

RUN apk add --no-cache tini

# Preserve the workspace directory structure so pnpm symlinks resolve correctly.
# apps/auth/node_modules symlinks point 4 levels up to node_modules/.pnpm —
# that depth must match between builder (/build) and runtime (/app).
WORKDIR /app

COPY --from=builder /build/node_modules ./node_modules
COPY --from=builder /build/apps/auth/node_modules ./apps/auth/node_modules
COPY --from=builder /build/apps/auth/dist ./apps/auth/dist
COPY --from=builder /build/apps/auth/package.json ./apps/auth/package.json
COPY --from=builder /build/apps/auth/drizzle ./apps/auth/drizzle
COPY --from=builder /build/apps/auth/drizzle.config.ts ./apps/auth/drizzle.config.ts

WORKDIR /app/apps/auth

EXPOSE 3000

ENTRYPOINT ["tini", "--"]
CMD ["node", "dist/index.js"]
