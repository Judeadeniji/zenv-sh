FROM node:22-alpine AS builder

WORKDIR /build

# Install dependencies first for caching.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/auth/package.json ./apps/auth/
COPY packages/amnesia/package.json ./packages/amnesia/
COPY packages/sdk/package.json ./packages/sdk/

RUN corepack enable pnpm && pnpm install --frozen-lockfile

# Copy source.
COPY apps/auth/ ./apps/auth/

RUN cd apps/auth && npx tsc

# ---

FROM node:22-alpine

RUN apk add --no-cache tini

WORKDIR /app

COPY --from=builder /build/node_modules ./node_modules
COPY --from=builder /build/apps/auth/node_modules ./apps/auth/node_modules
COPY --from=builder /build/apps/auth/dist ./dist
COPY --from=builder /build/apps/auth/package.json ./package.json
COPY --from=builder /build/apps/auth/drizzle ./drizzle

EXPOSE 3000

ENTRYPOINT ["tini", "--"]
CMD ["node", "--env-file=.env", "dist/index.js"]
