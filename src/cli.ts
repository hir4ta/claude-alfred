import { defineCommand, runMain } from 'citty';

const main = defineCommand({
  meta: {
    name: 'alfred',
    description: 'Development butler for Claude Code',
  },
  subCommands: {
    serve: defineCommand({
      meta: { description: 'Start MCP server (stdio)' },
      async run() {
        const { Store } = await import('./store/index.js');
        const { Embedder } = await import('./embedder/index.js');
        const { serveMCP } = await import('./mcp/server.js');
        const store = Store.openDefault();
        let emb = null;
        try { emb = Embedder.create(); } catch { /* no Voyage key */ }
        if (emb) store.expectedDims = emb.dims;
        const version = await resolveVersion();
        await serveMCP(store, emb, version);
      },
    }),
    dashboard: defineCommand({
      meta: { description: 'Open browser dashboard' },
      args: {
        port: { type: 'string', default: '7575', description: 'Port number' },
        'url-only': { type: 'boolean', default: false, description: 'Print URL only' },
      },
      async run({ args }) {
        const { Store } = await import('./store/index.js');
        const { Embedder } = await import('./embedder/index.js');
        const { startDashboard } = await import('./api/server.js');
        const projectPath = process.cwd();
        const store = Store.openDefault();
        let emb = null;
        try { emb = Embedder.create(); } catch { /* no Voyage key */ }
        if (emb) store.expectedDims = emb.dims;
        const version = await resolveVersion();
        await startDashboard(projectPath, store, emb, {
          port: parseInt(args.port, 10),
          urlOnly: args['url-only'],
          version,
        });
      },
    }),
    hook: defineCommand({
      meta: { description: 'Handle hook event' },
      args: {
        event: { type: 'positional', description: 'Event name' },
      },
      async run({ args }) {
        const { runHook } = await import('./hooks/dispatcher.js');
        await runHook(args.event as string);
      },
    }),
    'plugin-bundle': defineCommand({
      meta: { description: 'Generate plugin bundle' },
      args: {
        output: { type: 'positional', description: 'Output directory', default: 'plugin' },
      },
      async run({ args }) {
        // Placeholder — will copy content/ to output dir.
        console.log(`plugin-bundle: output=${args.output} (not yet implemented)`);
      },
    }),
    version: defineCommand({
      meta: { description: 'Show version' },
      args: {
        short: { type: 'boolean', default: false, description: 'Version only' },
      },
      async run({ args }) {
        const version = await resolveVersion();
        if (args.short) {
          console.log(version);
        } else {
          console.log(`alfred ${version}`);
        }
      },
    }),
  },
});

async function resolveVersion(): Promise<string> {
  try {
    const { readFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { fileURLToPath } = await import('node:url');
    const thisDir = fileURLToPath(new URL('.', import.meta.url));
    // Try package.json relative to dist/
    for (const rel of ['..', '../..']) {
      try {
        const pkg = JSON.parse(readFileSync(join(thisDir, rel, 'package.json'), 'utf-8'));
        if (pkg.version) return pkg.version;
      } catch { continue; }
    }
  } catch { /* ignore */ }
  return 'dev';
}

runMain(main);
