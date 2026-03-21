export interface SidebarLink {
	label: string;
	slug: string;
}

export interface SidebarSection {
	label: string;
	items: SidebarLink[];
}

export const sidebar: SidebarSection[] = [
	{
		label: 'Getting Started',
		items: [
			{ label: 'Introduction', slug: 'getting-started/introduction' },
			{ label: 'Quickstart', slug: 'getting-started/quickstart' },
			{ label: 'How It Works', slug: 'getting-started/how-it-works' },
		],
	},
	{
		label: 'Concepts',
		items: [
			{ label: 'Zero-Knowledge Architecture', slug: 'concepts/zero-knowledge' },
			{ label: 'Key Hierarchy', slug: 'concepts/key-hierarchy' },
			{ label: 'Projects & Environments', slug: 'concepts/environments' },
		],
	},
	{
		label: 'Guides',
		items: [
			{ label: 'Share secrets with your team', slug: 'guides/team-secrets' },
			{ label: 'Migrate from .env files', slug: 'guides/migrate-from-dotenv' },
			{ label: 'Secret versioning & rollback', slug: 'guides/versioning' },
			{ label: 'Managing access tokens', slug: 'guides/access-tokens' },
		],
	},
	{
		label: 'Examples',
		items: [
			{ label: 'CI/CD integration', slug: 'examples/ci-cd' },
			{ label: 'Framework integration', slug: 'examples/frameworks' },
			{ label: 'Docker & Kubernetes', slug: 'examples/docker-kubernetes' },
		],
	},
	{
		label: 'CLI',
		items: [
			{ label: 'Installation', slug: 'cli/installation' },
			{ label: 'Commands', slug: 'cli/commands' },
			{ label: 'Configuration', slug: 'cli/configuration' },
		],
	},
	{
		label: 'SDK',
		items: [
			{ label: 'Installation', slug: 'sdk/installation' },
			{ label: 'Usage', slug: 'sdk/usage' },
		],
	},
	{
		label: 'API Reference',
		items: [
			{ label: 'Authentication', slug: 'api/authentication' },
			{ label: 'Secrets', slug: 'api/secrets' },
			{ label: 'Projects', slug: 'api/projects' },
			{ label: 'Organizations', slug: 'api/organizations' },
			{ label: 'Tokens', slug: 'api/tokens' },
		],
	},
	{
		label: 'Self-Hosting',
		items: [
			{ label: 'Docker Compose', slug: 'self-hosting/docker-compose' },
			{ label: 'Environment Variables', slug: 'self-hosting/environment-variables' },
		],
	},
	{
		label: 'Security',
		items: [
			{ label: 'Threat Model', slug: 'security/threat-model' },
			{ label: 'Key Rotation', slug: 'security/key-rotation' },
		],
	},
	{
		label: 'Amnesia (Crypto Engine)',
		items: [
			{ label: 'Overview', slug: 'amnesia/overview' },
			{ label: 'Key Derivation', slug: 'amnesia/key-derivation' },
			{ label: 'Symmetric Encryption', slug: 'amnesia/symmetric' },
			{ label: 'Hashing', slug: 'amnesia/hashing' },
			{ label: 'Asymmetric Encryption', slug: 'amnesia/asymmetric' },
			{ label: 'Random Generation', slug: 'amnesia/random' },
		],
	},
];

/**
 * Given a current slug, find the previous and next pages across all sections.
 */
export function getPrevNext(currentSlug: string) {
	const flat = sidebar.flatMap((s) => s.items);
	const idx = flat.findIndex((item) => item.slug === currentSlug);
	return {
		prev: idx > 0 ? flat[idx - 1] : null,
		next: idx < flat.length - 1 ? flat[idx + 1] : null,
	};
}
