import { defineCollection } from 'astro:content';
import { z } from 'astro/zod';
import { glob } from 'astro/loaders';
import { docsSchema } from '@astrojs/starlight/schema';

// DESIGN-0003: the shared `docs/` tree lives at the repo root, not under
// `site/src/content/docs/`. Starlight's bundled `docsLoader()` is
// hardcoded to that path, so we use Astro's `glob()` loader with an
// explicit `base` pointing at `../docs` to mount the shared tree.
//
// Frontmatter compatibility: docz emits `id`, `status`, `author`, and
// `created`. Starlight's docsSchema doesn't know those fields, so we
// extend it with them as optional. `title` and the rest of Starlight's
// surface stay untouched.
export const collections = {
	docs: defineCollection({
		loader: glob({
			pattern: '**/[^_]*.{md,mdx}',
			base: '../docs',
		}),
		schema: docsSchema({
			extend: z.object({
				id: z.string().optional(),
				status: z.string().optional(),
				author: z.string().optional(),
				created: z.union([z.string(), z.date()]).optional(),
			}),
		}),
	}),
};
