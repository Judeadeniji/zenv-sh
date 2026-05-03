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
COPY packages/amnesia/ ./packages/amnesia/
COPY apps/web/ ./apps/web/

# Build-time env vars — Vite bakes VITE_* into the bundle at build time.
ARG VITE_AUTH_URL
ARG VITE_API_URL
ENV VITE_AUTH_URL=${VITE_AUTH_URL}
ENV VITE_API_URL=${VITE_API_URL}

# Build workspace dep first so its dist/ is present for the web build.
RUN cd packages/amnesia && pnpm build

RUN cd apps/web && pnpm build

# ---

FROM node:22-alpine

RUN apk add --no-cache tini

WORKDIR /app

COPY --from=builder /build/apps/web/.output ./

EXPOSE 3000

ENTRYPOINT ["tini", "--"]
CMD ["node", "server/index.mjs"]
