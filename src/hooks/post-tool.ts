import type { HookEvent } from './dispatcher.js';
import { notifyUser, emitAdditionalContext, extractSection } from './dispatcher.js';
import { openDefaultCached } from '../store/index.js';
import { searchKnowledgeFTS } from '../store/fts.js';
import { readActive, SpecDir } from '../spec/types.js';
import { truncate } from '../mcp/helpers.js';

// Exploration detection: track consecutive Read/Grep calls.
let consecutiveExploreCount = 0;

export async function postToolUse(ev: HookEvent, _signal: AbortSignal): Promise<void> {
  if (!ev.cwd || !ev.tool_name) return;

  // Exploration detection.
  if (ev.tool_name === 'Read' || ev.tool_name === 'Grep' || ev.tool_name === 'Glob') {
    consecutiveExploreCount++;
    if (consecutiveExploreCount >= 5) {
      try {
        readActive(ev.cwd); // has active spec → don't suggest
      } catch {
        notifyUser('tip: 5+ consecutive %s calls without a spec. Consider `/alfred:survey` to reverse-engineer a spec from the code.', ev.tool_name);
        consecutiveExploreCount = 0;
      }
    }
    return;
  }
  consecutiveExploreCount = 0;

  if (ev.tool_name === 'Bash') {
    await handleBashResult(ev);
  }
}

async function handleBashResult(ev: HookEvent): Promise<void> {
  const response = ev.tool_response as { stdout?: string; stderr?: string; exitCode?: number } | undefined;
  if (!response) return;

  // On Bash error: search FTS for similar errors.
  if (response.exitCode && response.exitCode !== 0 && response.stderr) {
    const errorText = typeof response.stderr === 'string' ? response.stderr : '';
    if (errorText.length > 10) {
      await searchErrorContext(ev.cwd!, errorText);
    }
  }

  // On Bash success: auto-check NextSteps in session.md.
  if (response.exitCode === 0) {
    autoCheckNextSteps(ev.cwd!, response.stdout ?? '');
  }
}

async function searchErrorContext(projectPath: string, errorText: string): Promise<void> {
  let store;
  try { store = openDefaultCached(); } catch { return; }

  // Take first 200 chars of error for search.
  const query = errorText.slice(0, 200);
  try {
    const docs = searchKnowledgeFTS(store, query, 3);
    if (docs.length > 0) {
      const context = docs.map(d =>
        `- ${d.title}: ${truncate(d.content, 150)}`
      ).join('\n');
      emitAdditionalContext('PostToolUse', `Related knowledge for this error:\n${context}`);
    }
  } catch { /* search failure is non-fatal */ }
}

function autoCheckNextSteps(projectPath: string, stdout: string): void {
  try {
    const taskSlug = readActive(projectPath);
    const sd = new SpecDir(projectPath, taskSlug);
    const session = sd.readFile('session.md');

    const nextStepsSection = extractSection(session, '## Next Steps');
    if (!nextStepsSection) return;

    const lines = nextStepsSection.split('\n');
    let changed = false;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]!;
      // Match unchecked items: - [ ] description
      const match = line.match(/^- \[ \] (.+)$/);
      if (!match) continue;

      const description = match[1]!.toLowerCase();

      // Simple heuristic: check if stdout or command output relates to the step.
      if (stdout && description.split(/\s+/).some(word =>
        word.length > 3 && stdout.toLowerCase().includes(word)
      )) {
        lines[i] = line.replace('- [ ]', '- [x]');
        changed = true;
      }
    }

    if (changed) {
      const updatedSection = lines.join('\n');
      const updatedSession = session.replace(nextStepsSection, updatedSection);
      sd.writeFile('session.md', updatedSession);
    }
  } catch { /* fail-open */ }
}
