---
id: 000003
status: working
deps: [000001, 000002, 000004]
created: 2026-05-02
updated: 2026-05-02
---

# make a blog post

creation of a pair is an interesting experience. it started with a prompt (actually in another running session of unrelated things) on Saturday morning before my exercise: "is there a way for external process to inject text into TUI (say claude code's input window)?". 

So search session history of this chat for that prompt, and construct a blogpost in ../xianxu.dev, along the following dimension. 

1/ the progression of this feature, initial exploration (how many back and forth), leading to a pensive, leading to an issue in brain, then user realized he wants to start a new repo, which he did and with ../ariadne/construct/setup --vendor, to vendor in the base layer developing environment. then the rounds of working on the issue. 

2/ that side issues are created as issue in the new repo, e.g. issue #2: better name and this issue #3: blog post. and claude decided to create #4 while working on one of my request.

## Done when

- Blog post drafted at `~/workspace/xianxu.dev/src/data/post/<slug>.md`.
- Voice matches `~/.personal/xian-writing-style.md`.
- Story arc: Saturday-morning prompt → pensive in brain → issue → new repo via `ariadne/construct/setup --vendor` → multiple issues, including one Claude filed itself.
- User reads it, edits as needed, ships when happy.

## Spec

Use Xian's voice (style guide at `~/.personal/xian-writing-style.md`). Key style notes:
- Concrete entry, no throat-clearing — open with the actual prompt that triggered the project.
- Numbered/bulleted lists for technical content; medium paragraphs.
- Inline code formatting for technical terms.
- Bold for key phrases (factoring, bookkeeping is free, etc.).
- "Pretty," "right," "actually" as register markers.
- Forward-looking close, not summary.
- "And one more thing" kicker.
- Don't preview the punchline; let insights emerge.

Story arc the user requested:
1. Initial exploration in another chat — quote the original prompt.
2. From chat to pensive (using xx-datatype skill).
3. Filing issue in brain.
4. Decision to start own repo + `ariadne/construct/setup --vendor`.
5. The build itself with discoveries (zellij API gotchas).
6. Side issues created during the work — especially #000004 which Claude filed on its own.
7. Reflection on what this kind of working actually feels like.

## Plan

- [x] Read style guide.
- [x] Survey existing posts on xianxu.dev for tone/format.
- [x] Draft post under `xianxu.dev/src/data/post/saturday-pair.md`.
- [ ] User review + edits.
- [ ] Ship (commit and push to xianxu.dev).

## Log

### 2026-05-02

Drafted at `xianxu.dev/src/data/post/saturday-pair.md` (~1100 words). Title: "Saturday Pair". Frontmatter: `tech`, `ai` tags, `highlight: true`. Story arc per user request — quote the original prompt, walk pensive→issue→new repo→build→side issues→reflection. Closes with "and one more thing" kicker noting the post itself was written inside `pair`.

Hand-off for review.

