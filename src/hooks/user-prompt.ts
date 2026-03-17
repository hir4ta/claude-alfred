import type { HookEvent } from './dispatcher.js';
import { notifyUser, emitAdditionalContext } from './dispatcher.js';
import { openDefaultCached } from '../store/index.js';
import { Embedder } from '../embedder/index.js';
import { searchPipeline, trackHitCounts, truncate } from '../mcp/helpers.js';

// Intent classification for skill nudge.
const INTENT_KEYWORDS: Record<string, string[]> = {
  research: ['research', 'investigate', 'understand', 'explore', 'learn', 'pattern', '調べ', '調査', '理解', '質問'],
  plan: ['plan', 'design', 'architect', 'how to', 'approach', 'アーキテクチャ', '設計', '計画'],
  implement: ['implement', 'add', 'create', 'build', 'refactor', '追加', '実装', 'リファクタ'],
  bugfix: ['fix', 'bug', 'error', 'broken', 'failing', '修正', 'バグ', 'エラー'],
  review: ['review', 'check', 'audit', 'inspect', 'レビュー', '確認'],
  tdd: ['test', 'tdd', 'spec', 'テスト'],
  'save-knowledge': ['remember', 'save', 'note', 'record', '覚え', '保存', 'メモ'],
};

const INTENT_TO_SKILL: Record<string, string> = {
  research: '/alfred:brief',
  plan: '/alfred:attend',
  implement: '/alfred:attend',
  bugfix: '/alfred:mend',
  review: '/alfred:inspect',
  tdd: '/alfred:tdd',
};

export async function userPromptSubmit(ev: HookEvent, signal: AbortSignal): Promise<void> {
  if (!ev.prompt || !ev.cwd) return;

  const prompt = ev.prompt.trim();
  if (!prompt) return;

  let store;
  try { store = openDefaultCached(); } catch { return; }

  // Semantic search for relevant knowledge.
  let emb: Embedder | null = null;
  try { emb = Embedder.create(); } catch { /* no Voyage key — FTS fallback */ }

  const limit = 5;
  const result = await searchPipeline(store, emb, prompt, limit, limit * 3);

  const parts: string[] = [];

  // Skill nudge.
  const intent = classifyIntent(prompt);
  if (intent && intent !== 'save-knowledge') {
    const skill = INTENT_TO_SKILL[intent];
    if (skill) {
      parts.push(`Skill suggestion: ${skill} — ${intentDescription(intent)}`);
    }
  }

  // Knowledge context.
  if (result.docs.length > 0) {
    trackHitCounts(store, result.docs);
    const contextLines = result.docs.map(d => {
      const label = d.subType !== 'general' ? `[${d.subType}] ` : '';
      return `- ${label}${d.title}: ${truncate(d.content, 150)}`;
    });
    parts.push('Related knowledge:\n' + contextLines.join('\n'));
  }

  if (parts.length > 0) {
    emitAdditionalContext('UserPromptSubmit', parts.join('\n\n'));
  }
}

function classifyIntent(prompt: string): string | null {
  const lower = prompt.toLowerCase();
  let bestIntent = '';
  let bestScore = 0;

  for (const [intent, keywords] of Object.entries(INTENT_KEYWORDS)) {
    let score = 0;
    for (const kw of keywords) {
      if (lower.includes(kw)) score++;
    }
    if (score > bestScore) {
      bestScore = score;
      bestIntent = intent;
    }
  }

  // save-knowledge suppresses research when both match.
  if (bestIntent === 'research') {
    let saveScore = 0;
    for (const kw of INTENT_KEYWORDS['save-knowledge']!) {
      if (lower.includes(kw)) saveScore++;
    }
    if (saveScore > 0) return 'save-knowledge';
  }

  return bestScore > 0 ? bestIntent : null;
}

function intentDescription(intent: string): string {
  switch (intent) {
    case 'research': return '調査・リサーチの構造化';
    case 'plan': return '仕様策定→承認→実装の自律フロー';
    case 'implement': return '仕様策定→承認→実装の自律フロー';
    case 'bugfix': return '再現→分析→修正→検証の自律バグ修正';
    case 'review': return '6プロファイル品質レビュー';
    case 'tdd': return 'Red→Green→Refactor の自律TDD';
    default: return '';
  }
}
