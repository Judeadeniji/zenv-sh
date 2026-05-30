# zEnv Documentation Site

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](../../LICENSE-MIT)

The official documentation website for **zEnv**. Built with [Astro](https://astro.build) and [Starlight](https://starlight.astro.build).

## Development

```bash
# From the repo root
pnpm install

# Start the dev server
cd apps/docs
pnpm dev
```

The documentation site will be available at `http://localhost:4321`.

## Structure

```text
apps/docs/
├── src/
│   ├── content/
│   │   └── docs/          # Markdown and MDX pages
│   ├── assets/            # Images and static assets
│   └── components/        # Custom Astro/React components
├── public/                # Static files served at the root
└── astro.config.mjs       # Astro and Starlight configuration
```

## Adding Content

Documentation pages are written in standard Markdown (`.md`) or MDX (`.mdx`).

To add a new page:
1. Create a file in `src/content/docs/`. The file path dictates the URL route.
2. Add frontmatter to specify the title and metadata.
3. Update the `sidebar` configuration in `astro.config.mjs` if necessary to feature the page in navigation.

See the [Starlight Documentation](https://starlight.astro.build/guides/pages/) for details on writing content and using Starlight components.

## Building for Production

```bash
pnpm build
```

The generated static site will be placed in the `dist/` directory, ready to be deployed to any static host.

## License

MIT — see [LICENSE](../../LICENSE-MIT).
