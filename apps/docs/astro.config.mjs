// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
	integrations: [
		starlight({
			title: 'zEnv',
			description: 'Zero-knowledge encryption platform. Your secrets, your keys, your data.',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/Judeadeniji/zenv-sh' }],
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						{ label: 'Introduction', slug: 'getting-started/introduction' },
						{ label: 'Quickstart', slug: 'getting-started/quickstart' },
						{ label: 'How It Works', slug: 'getting-started/how-it-works' },
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
				{
					label: 'CLI',
					autogenerate: { directory: 'cli' },
				},
				{
					label: 'SDK',
					autogenerate: { directory: 'sdk' },
				},
				{
					label: 'Self-Hosting',
					autogenerate: { directory: 'self-hosting' },
				},
			],
			editLink: {
				baseUrl: 'https://github.com/Judeadeniji/zenv-sh/edit/main/docs/',
			},
		}),
	],
});
