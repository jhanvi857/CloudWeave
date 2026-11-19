import { docs } from '@/.source/server';
import { loader } from 'fumadocs-core/source';
import { llms } from 'fumadocs-core/source/llms';
import { lucideIconsPlugin } from 'fumadocs-core/source/lucide-icons';
import { docsRoute } from './shared';

// Source loader powered by Fumadocs Content Collections (fully compatible with Turbopack & Webpack)
export const source = loader({
  baseUrl: docsRoute,
  source: docs.toFumadocsSource(),
  plugins: [lucideIconsPlugin()],
});

export const docsLlms = llms(source, {
  renderPage: (page: any) => page.data.description ?? '',
}) as any;


