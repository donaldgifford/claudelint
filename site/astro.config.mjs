// @ts-check
import { defineConfig } from 'astro/config';
import { fileURLToPath } from 'node:url';
import starlight from '@astrojs/starlight';
import remarkMdLinkRewriter from './src/plugins/remark-md-link-rewriter.mjs';

// DESIGN-0003: dual-output docs site. This config wires Starlight to
// the shared root `docs/` tree via the content collection in
// src/content.config.ts. Sidebar groups mirror the docz doc-type
// directories (RFC / ADR / DESIGN / IMPL / PLAN / Investigation), each
// using `autogenerate` so dropping a new `.md` into `docs/<type>/`
// surfaces it without touching this file.
//
// `remark-md-link-rewriter` is a small custom plugin (under
// src/plugins/) that rewrites Markdown-style cross-doc links like
// `[DESIGN-0001](../design/0001-foo.md)` into absolute Starlight
// routes (`/design/0001-foo/`). MkDocs handles the .md suffix
// natively; we share the same link style in both pipelines so authors
// only learn one convention. The off-the-shelf
// `astro-rehype-relative-markdown-links` assumes content lives at
// `src/content/<collection>/`, which our shared-source setup
// (`../docs/`) breaks.
const docsContentRoot = fileURLToPath(new URL('../docs', import.meta.url));

export default defineConfig({
	site: 'https://claudelint.dev',
	markdown: {
		remarkPlugins: [[remarkMdLinkRewriter, { contentRoot: docsContentRoot }]],
	},
	integrations: [
		starlight({
			title: 'claudelint',
			description:
				'Linter for Claude Code artifacts — CLAUDE.md, skills, commands, agents, hooks, plugins, marketplaces, and MCP servers.',
			social: [
				{
					icon: 'cloud-download',
					label: 'Releases',
					href: 'https://github.com/donaldgifford/claudelint/releases/latest',
				},
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/donaldgifford/claudelint',
				},
			],
			editLink: {
				baseUrl: 'https://github.com/donaldgifford/claudelint/edit/main/docs/',
			},
			sidebar: [
				{
					label: 'Install',
					items: [{ autogenerate: { directory: 'install' } }],
				},
				{
					label: 'Rules',
					items: [{ autogenerate: { directory: 'rules' } }],
				},
				{
					label: 'Development',
					collapsed: true,
					items: [
						{ autogenerate: { directory: 'development' } },
						{
							label: 'RFCs',
							collapsed: true,
							items: [{ autogenerate: { directory: 'rfc' } }],
						},
						{
							label: 'ADRs',
							collapsed: true,
							items: [{ autogenerate: { directory: 'adr' } }],
						},
						{
							label: 'Design',
							collapsed: true,
							items: [{ autogenerate: { directory: 'design' } }],
						},
						{
							label: 'Implementation',
							collapsed: true,
							items: [{ autogenerate: { directory: 'impl' } }],
						},
						{
							label: 'Plans',
							collapsed: true,
							items: [{ autogenerate: { directory: 'plan' } }],
						},
						{
							label: 'Investigations',
							collapsed: true,
							items: [{ autogenerate: { directory: 'investigation' } }],
						},
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'JSON output schema', slug: 'json-output-schema' },
						{ label: 'Rules JSON schema', slug: 'rules-json-schema' },
					],
				},
				{
					label: 'Changelog',
					slug: 'changelog',
				},
			],
		}),
	],
});
