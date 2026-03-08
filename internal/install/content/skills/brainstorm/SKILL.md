---
name: brainstorm
description: |
  Multi-agent divergent thinking: 3 specialist agents (Visionary, Pragmatist, Critic) generate
  perspectives in parallel, then debate to produce richer, more grounded output.
  Each agent researches independently via knowledge base and web search.
  Use when: (1) unsure what to think about, (2) ideas are few or thinking is rigid,
  (3) need to surface risks and issues, (4) need raw material for convergence (/alfred:refine).
user-invocable: true
disable-model-invocation: true
argument-hint: "<theme or rough prompt>"
allowed-tools: Read, Glob, Grep, AskUserQuestion, Agent, WebSearch, WebFetch, mcp__plugin_alfred_alfred__knowledge, mcp__plugin_alfred_alfred__spec
context: fork
---

# /alfred:brainstorm — Multi-Agent Divergent Thinking

3 specialist agents generate perspectives in parallel, then debate to produce
richer, more grounded divergent output. The goal is not "deciding" but "expanding."

## Key Principles
- This skill's role is **divergence**. It does not judge or decide (decisions are made by /alfred:refine).
- Where facts are insufficient, explicitly label as "hypothesis" — **never assert speculation as fact**.
- Prefer breadth over depth — surface many angles, don't deep-dive on one.

## Phase 0: Intake & Minimal Assumption Check (AskUserQuestion recommended)

Confirm with up to 3 questions (with choices):

1) What is the goal?
- a) Determine direction
- b) Expand options
- c) Surface risks/issues
- d) Reframe the question

2) Any constraints?
- Deadline / time / budget / team / tech restrictions / hard no's

3) What is the scope?
- Personal decision / team consensus / product / learning / career etc

*If the user says "you decide", proceed with reasonable defaults.*

## Phase 1: Parallel Perspective Generation (3 agents)

Search knowledge for relevant context first, then spawn **all 3 agents in a single
message** using the Agent tool. Pass each the theme, constraints, and any knowledge
base results.

**Agent 1: Visionary** — Possibilities, innovation, what-if scenarios
```
You are the Visionary. Your role is to think BIG and explore possibilities.

Theme: <theme>
Constraints: <constraints>
Knowledge context: <knowledge results if any>

Your job:
1. Search the web for innovative approaches, emerging trends, and unconventional
   solutions related to this theme
2. Generate 5-7 ideas in the "Experimental / Ambitious" category
3. For each idea: one-liner, 30-second explanation, when it works, biggest upside
4. Think about what's possible if constraints were loosened
5. Surface non-obvious connections and analogies from other domains

Format your output as a structured list. Be bold — this is divergence, not judgment.
```

**Agent 2: Pragmatist** — Feasibility, effort, trade-offs, proven solutions
```
You are the Pragmatist. Your role is to find workable, proven approaches.

Theme: <theme>
Constraints: <constraints>
Knowledge context: <knowledge results if any>

Your job:
1. Search the web for proven solutions, established patterns, and case studies
   related to this theme
2. Generate 5-7 ideas in the "Conservative / Realistic" category
3. For each idea: one-liner, 30-second explanation, effort estimate, fit with constraints
4. Identify trade-off axes (speed/quality, short-term/long-term, etc.)
5. Surface what solutions others have used successfully in similar situations

Format your output as a structured list. Be practical — ground everything in reality.
```

**Agent 3: Critic** — Risks, failure modes, edge cases, blind spots
```
You are the Critic. Your role is to find what could go wrong and what's missing.

Theme: <theme>
Constraints: <constraints>
Knowledge context: <knowledge results if any>

Your job:
1. Search the web for post-mortems, failure cases, and anti-patterns related
   to this theme
2. Identify 5-7 risks, failure patterns, and blind spots
3. For each: what goes wrong, why it's likely, how to detect early
4. Challenge the assumptions behind the theme itself — is the question right?
5. Surface hidden dependencies and second-order effects

Format your output as a structured list. Be thorough — find what others will miss.
```

## Phase 2: Cross-Critique (1 round, parent-mediated)

After collecting all 3 agents' output:

1. Identify **conflicts** between perspectives (Visionary vs Critic, etc.)
2. Identify **gaps** — topics none of the agents covered
3. Spawn a **single synthesis agent** with all 3 outputs:

```
You are a synthesis moderator. Three specialists have generated perspectives on a theme.

Visionary's output: <...>
Pragmatist's output: <...>
Critic's output: <...>

Your job:
1. Identify 3-5 key tension points where the specialists disagree
2. For each tension: state both sides and why the disagreement matters
3. Identify 2-3 blind spots that ALL THREE missed
4. Suggest 2-3 hybrid ideas that combine the best of multiple perspectives
5. Rank the top 5 ideas across all specialists by "most worth exploring further"

Be concise. Focus on what's NEW from the synthesis, not restating what was already said.
```

## Phase 3: Final Output (Markdown)

Compile everything into this structure:

```md
# Brainstorm Output: <Theme>

## Assumptions
- Goal:
- Constraints:
- Scope:

## Perspectives (3-agent synthesis)

### Visionary — Bold possibilities
- ... (top ideas from Visionary agent)

### Pragmatist — Proven approaches
- ... (top ideas from Pragmatist agent)

### Critic — Risks & blind spots
- ... (top risks from Critic agent)

## Key Tensions (where agents disagreed)
1. <Tension>: Visionary says X, Critic says Y. This matters because...
2. ...

## Hybrid Ideas (synthesis)
- ... (combined ideas from synthesis round)

## Blind Spots (what all 3 missed)
- ...

## Top Ideas (ranked by synthesis)
1.
2.
3.
4.
5.

## Questions to Answer for Convergence (priority order)
1.
2.
3.

## Recommended Next Step
- To converge: /alfred:refine
- To create a spec: /alfred:plan
- To explore: files to read in Plan Mode / commands to investigate
```

## Exit Criteria
- All 3 specialist agents completed
- Synthesis round completed
- At least 15 ideas generated across all agents
- Key tensions and blind spots identified
- Questions for convergence are ready
