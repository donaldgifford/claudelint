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
					label: 'RFCs',
					items: [{ autogenerate: { directory: 'rfc' } }],
				},
				{
					label: 'ADRs',
					items: [{ autogenerate: { directory: 'adr' } }],
				},
				{
					label: 'Design',
					items: [{ autogenerate: { directory: 'design' } }],
				},
				{
					label: 'Implementation',
					items: [{ autogenerate: { directory: 'impl' } }],
				},
				{
					label: 'Plans',
					items: [{ autogenerate: { directory: 'plan' } }],
				},
				{
					label: 'Investigations',
					items: [{ autogenerate: { directory: 'investigation' } }],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'JSON output schema', slug: 'json-output-schema' },
						{ label: 'Rules JSON schema', slug: 'rules-json-schema' },
					],
				},
			],
		}),
	],
});
