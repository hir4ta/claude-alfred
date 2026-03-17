import { Hono } from 'hono';
import { serve } from '@hono/node-server';
import { serveStatic } from '@hono/node-server/serve-static';
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import type { Store } from '../store/index.js';
import type { Embedder } from '../embedder/index.js';
import {
  SpecDir, readActiveState, VALID_SLUG, filesForSize,
} from '../spec/types.js';
import type { SpecFile, SpecSize, SpecType } from '../spec/types.js';
import { listAllEpics } from '../epic/index.js';
import {
  listAllKnowledge, getKnowledgeStats, setKnowledgeEnabled,
} from '../store/knowledge.js';
import { searchKnowledgeFTS } from '../store/fts.js';
import { detectProject } from '../store/project.js';

export interface DashboardOptions {
  port: number;
  urlOnly: boolean;
  version: string;
}

export function createApp(
  projectPath: string,
  store: Store,
  emb: Embedder | null,
  version: string,
): Hono {
  const app = new Hono();
  const proj = detectProject(projectPath);

  // --- API Routes ---

  app.get('/api/version', (c) => c.json({ version }));
  app.get('/api/project', (c) => c.json({ path: projectPath, name: proj.name }));

  app.get('/api/tasks', (c) => {
    try {
      const state = readActiveState(projectPath);
      return c.json({ active: state.primary, tasks: state.tasks });
    } catch {
      return c.json({ active: '', tasks: [] });
    }
  });

  app.get('/api/tasks/:slug/specs/:file', (c) => {
    const slug = c.req.param('slug');
    const file = c.req.param('file');
    if (!VALID_SLUG.test(slug)) return c.json({ error: 'invalid slug' }, 400);

    const sd = new SpecDir(projectPath, slug);
    try {
      const content = sd.readFile(file as SpecFile);
      return c.json({ content });
    } catch {
      return c.json({ error: 'spec file not found' }, 404);
    }
  });

  app.get('/api/tasks/:slug/specs', (c) => {
    const slug = c.req.param('slug');
    if (!VALID_SLUG.test(slug)) return c.json({ error: 'invalid slug' }, 400);

    const sd = new SpecDir(projectPath, slug);
    const sections = sd.exists() ? sd.allSections() : [];
    return c.json({ specs: sections });
  });

  app.get('/api/tasks/:slug/validation', (c) => {
    const slug = c.req.param('slug');
    if (!VALID_SLUG.test(slug)) return c.json({ error: 'invalid slug' }, 400);

    const sd = new SpecDir(projectPath, slug);
    if (!sd.exists()) return c.json({ error: 'not found' }, 404);

    // Basic validation (same as dossier validate).
    let state;
    try { state = readActiveState(projectPath); } catch { return c.json({ checks: [] }); }
    const task = state.tasks.find(t => t.slug === slug);
    const size = (task?.size ?? 'L') as SpecSize;
    const specType = (task?.spec_type ?? 'feature') as SpecType;
    const expectedFiles = filesForSize(size, specType);
    const checks = expectedFiles.map(f => {
      try { sd.readFile(f); return { name: f, status: 'pass' }; }
      catch { return { name: f, status: 'fail' }; }
    });
    return c.json({ checks });
  });

  app.get('/api/knowledge', (c) => {
    const limit = Math.min(parseInt(c.req.query('limit') ?? '50', 10) || 50, 500);
    const entries = listAllKnowledge(store, proj.remote, proj.path, limit);
    return c.json({ entries });
  });

  app.get('/api/knowledge/search', (c) => {
    const query = c.req.query('q');
    if (!query) return c.json({ error: "query parameter 'q' is required" }, 400);
    const limit = Math.min(parseInt(c.req.query('limit') ?? '10', 10) || 10, 500);
    const entries = searchKnowledgeFTS(store, query, limit);
    return c.json({ entries, method: 'fts5' });
  });

  app.get('/api/knowledge/stats', (c) => {
    const stats = getKnowledgeStats(store);
    return c.json(stats);
  });

  app.patch('/api/knowledge/:id/enabled', async (c) => {
    const id = parseInt(c.req.param('id'), 10);
    if (isNaN(id)) return c.json({ error: 'invalid id' }, 400);
    const body = await c.req.json<{ enabled: boolean }>();
    setKnowledgeEnabled(store, id, body.enabled);
    return c.json({ ok: true });
  });

  app.get('/api/activity', (c) => {
    const auditPath = join(projectPath, '.alfred', 'audit.jsonl');
    const entries: unknown[] = [];
    try {
      const content = readFileSync(auditPath, 'utf-8');
      for (const line of content.split('\n')) {
        if (line.trim()) {
          try { entries.push(JSON.parse(line)); } catch { /* skip */ }
        }
      }
    } catch { /* no audit file */ }
    return c.json({ entries: entries.reverse().slice(0, 100) });
  });

  app.get('/api/epics', (c) => {
    const epics = listAllEpics(projectPath);
    return c.json({ epics });
  });

  app.get('/api/health', (c) => {
    const stats = getKnowledgeStats(store);
    return c.json({ total: stats.total, bySubType: stats.bySubType });
  });

  // --- SSE ---
  app.get('/api/events', (c) => {
    return c.newResponse(
      new ReadableStream({
        start(controller) {
          const encoder = new TextEncoder();
          controller.enqueue(encoder.encode('event: connected\ndata: {}\n\n'));

          // Poll .alfred/ for changes every 5s.
          const alfredDir = join(projectPath, '.alfred');
          let lastMtime = dirMaxMtime(alfredDir);
          const interval = setInterval(() => {
            const mtime = dirMaxMtime(alfredDir);
            if (mtime > lastMtime) {
              lastMtime = mtime;
              controller.enqueue(encoder.encode('event: refresh\ndata: {}\n\n'));
            }
          }, 5000);

          const signal = c.req.raw.signal;
          if (signal) {
            signal.addEventListener('abort', () => {
              clearInterval(interval);
              controller.close();
            });
          }
        },
      }),
      {
        headers: {
          'Content-Type': 'text/event-stream',
          'Cache-Control': 'no-cache',
          'Connection': 'keep-alive',
        },
      },
    );
  });

  // --- SPA serving ---
  if (process.env['ALFRED_DEV'] === '1') {
    // Dev mode: proxy to Vite.
    app.all('/*', async (c) => {
      const url = new URL(c.req.url);
      url.host = 'localhost:5173';
      url.protocol = 'http:';
      const resp = await fetch(url.toString(), {
        method: c.req.method,
        headers: c.req.raw.headers,
      });
      return new Response(resp.body, {
        status: resp.status,
        headers: resp.headers,
      });
    });
  } else {
    // Production: serve from web/dist/.
    const webDistPath = resolveWebDist();
    if (webDistPath && existsSync(webDistPath)) {
      app.use('/*', serveStatic({ root: webDistPath }));
      // SPA fallback: serve index.html for client-side routing.
      app.get('*', (c) => {
        const indexPath = join(webDistPath, 'index.html');
        try {
          const html = readFileSync(indexPath, 'utf-8');
          return c.html(html);
        } catch {
          return c.text('Dashboard not built. Run: npm run build:web', 404);
        }
      });
    }
  }

  return app;
}

export async function startDashboard(
  projectPath: string,
  store: Store,
  emb: Embedder | null,
  opts: DashboardOptions,
): Promise<void> {
  const app = createApp(projectPath, store, emb, opts.version);
  const addr = `http://localhost:${opts.port}`;

  if (opts.urlOnly) {
    console.log(addr);
  } else {
    console.error(`alfred dashboard: ${addr}`);
    openBrowser(addr);
  }

  serve({ fetch: app.fetch, port: opts.port });

  // Wait for signal.
  await new Promise<void>((resolve) => {
    process.on('SIGINT', () => { console.error('\nshutting down...'); resolve(); });
    process.on('SIGTERM', () => { console.error('\nshutting down...'); resolve(); });
  });
}

function resolveWebDist(): string {
  // Try relative to this file (npm package layout).
  const thisDir = fileURLToPath(new URL('.', import.meta.url));
  const candidates = [
    join(thisDir, '..', 'web', 'dist'),
    join(thisDir, '..', '..', 'web', 'dist'),
  ];
  for (const p of candidates) {
    if (existsSync(join(p, 'index.html'))) return p;
  }
  return join(process.cwd(), 'web', 'dist');
}

function dirMaxMtime(dir: string): number {
  let maxT = 0;
  try {
    for (const entry of readdirSync(dir)) {
      try {
        const info = statSync(join(dir, entry));
        if (info.mtimeMs > maxT) maxT = info.mtimeMs;
        if (info.isDirectory()) {
          for (const sub of readdirSync(join(dir, entry))) {
            try {
              const si = statSync(join(dir, entry, sub));
              if (si.mtimeMs > maxT) maxT = si.mtimeMs;
            } catch { continue; }
          }
        }
      } catch { continue; }
    }
  } catch { /* dir doesn't exist */ }
  return maxT;
}

function openBrowser(url: string): void {
  import('node:child_process').then(({ execSync }) => {
    if (process.platform === 'darwin') {
      execSync(`open "${url}"`, { stdio: 'ignore' });
    } else if (process.platform === 'linux') {
      execSync(`xdg-open "${url}"`, { stdio: 'ignore' });
    }
  }).catch(() => { /* ignore */ });
}
