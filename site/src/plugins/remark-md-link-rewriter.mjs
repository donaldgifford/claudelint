// Rewrites Markdown relative links that point at .md/.mdx files
// (e.g. `../design/0001-foo.md`) into absolute Starlight routes
// (`/design/0001-foo/`). DESIGN-0003 keeps the same docs/ tree feeding
// both MkDocs and Starlight, so we need authors to write one link
// style that works in both pipelines. MkDocs handles relative .md
// hrefs natively; Starlight needs help because its directory-index
// routes break relative-resolution math.
//
// astro-rehype-relative-markdown-links is the off-the-shelf option but
// it assumes content lives under `src/content/<collection>/`. Our
// content lives at `../docs/` (shared with MkDocs), so we resolve
// hrefs against the source file's absolute path and emit absolute
// routes.

import { visit } from 'unist-util-visit';
import { dirname, isAbsolute, posix, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const MD_LINK_RE = /\.mdx?(#.*)?$/;

/**
 * @typedef {Object} Options
 * @property {string} contentRoot Absolute path to the shared docs tree
 *   (e.g. the repo's `docs/` directory). Required.
 */

/**
 * @param {Options} options
 */
export default function remarkMdLinkRewriter(options) {
	const contentRoot = options?.contentRoot;
	if (!contentRoot || !isAbsolute(contentRoot)) {
		throw new Error(
			'remark-md-link-rewriter: `contentRoot` must be an absolute path to the shared docs tree.',
		);
	}

	return (tree, file) => {
		const sourcePath = file?.path ?? file?.history?.[file.history.length - 1];
		if (!sourcePath) return;

		const sourceAbs = sourcePath.startsWith('file:') ? fileURLToPath(sourcePath) : sourcePath;
		const sourceDir = dirname(sourceAbs);

		visit(tree, 'link', (node) => {
			const url = node.url;
			if (typeof url !== 'string') return;
			// Skip absolute URLs, anchors, protocol-relative, mailto, and
			// already-absolute site paths.
			if (/^([a-z]+:|\/\/|#|mailto:|\/)/i.test(url)) return;
			if (!MD_LINK_RE.test(url)) return;

			const [pathPart, hash = ''] = url.split('#');
			const targetAbs = resolve(sourceDir, pathPart);
			const rel = posix.relative(contentRoot, targetAbs.split('\\').join('/'));
			if (rel.startsWith('..')) return; // outside the docs tree — leave it alone

			// Strip extension and convert to Starlight route. Starlight
			// lowercases segments when generating IDs (e.g. `README.md` ->
			// slug `readme`), so we lowercase here to match.
			const withoutExt = rel.replace(/\.mdx?$/i, '').toLowerCase();
			node.url = `/${withoutExt}/${hash ? `#${hash}` : ''}`;
		});
	};
}
