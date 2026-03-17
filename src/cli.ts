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
        await serveMCP(store, emb, '0.1.0-alpha.0');
      },
    }),
    hook: defineCommand({
      meta: { description: 'Handle hook event' },
      args: {
        event: { type: 'positional', description: 'Event name (SessionStart, PreCompact, UserPromptSubmit, PostToolUse)' },
      },
      async run({ args }) {
        const { runHook } = await import('./hooks/dispatcher.js');
        await runHook(args.event as string);
      },
    }),
    version: defineCommand({
      meta: { description: 'Show version' },
      run() {
        console.log('alfred 0.1.0-alpha.0');
      },
    }),
  },
});

runMain(main);
