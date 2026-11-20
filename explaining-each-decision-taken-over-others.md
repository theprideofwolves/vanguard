# Vanguard: design decisions

1. **Keep normalized input.** Assessment and conversion stay upstream. The gate
   evaluates supplied evidence; it does not discover vulnerabilities or verify
   that an upstream scan completed.
2. **Compare policy semantics and sample outcomes separately.** Comparing only
   historical verdicts misses relaxations not exercised by the sample corpus.
   Any relaxed limit sets ReviewRequired; mixed changes also require review.
3. **Treat omitted and -1 limits identically.** Both mean unlimited. Raw hashes
   still distinguish their bytes without inventing a semantic change.
4. **Reuse one evaluator.** Ordinary decisions and paired comparison decisions
   share decisionFor/evaluator.Evaluate. The report is parsed once per comparison,
   then held fixed under both policies to isolate the policy's observed effect.
5. **Keep all four transitions.** Newly passing reports expose changed acceptance;
   newly failing reports show rollout impact. Equal aggregate totals can hide both.
   Unchanged failures can still have different violations, retained in paired evidence.
6. **Fail a corpus atomically.** Silently skipping an invalid report could bias the
   review toward a partial sample. No result is returned if either policy or any
   report fails validation. An empty corpus is not a useful impact analysis.
7. **Use explicit, bounded batches.** Nonrecursive local JSON intake is a report
   loader, not a folder scanner. Count/byte/enumeration caps make the simple
   in-memory design deliberate. Sorting labels produces stable ordering.
8. **Retain exact-byte hashes.** They identify compared inputs, not authenticity,
   completeness, policy approval or cryptographic provenance. Both paired decisions
   carry the same input hash. Timestamps vary between separate runs.
9. **Keep one current UI artifact.** Evaluation and comparison replace the same
   export slot; a new invalid evaluation request clears it before upload parsing.
   Guide/theme navigation preserves it. Restart loses inputs and decisions.
10. **Keep notifications narrow.** Optional failure-only Slack belongs to ordinary
    CLI evaluation. Comparing policies or using the browser cannot send messages.
    Delivery failures do not change an already-computed policy verdict.

Go's standard-library HTTP/templates and a single YAML dependency keep packaging
small and the control flow inspectable. The UI uses shared white/black styles,
a teal accent and a protected cookie-based theme switch without JavaScript.
Server-rendered full-page forms trade richer interaction for simpler runtime and
state handling. Report uploads override pasted text and may use temporary files.

The differentiator is explainable policy review, not a claim of new academic
research. A threshold increase may be justified; the tool exposes it for review
rather than asserting that every relaxation is unsafe. Authentication for shared
hosting, live scanner adapters, saved history, exception engines, baseline finding
identity and automatic PR approval remain outside this version's scope.

## Pinned-review extension (manual verification pending)

- Keep the deterministic evaluator unchanged. New context/export code is isolated
  in internal/review and an explicit CLI subcommand.
- Read the baseline from a full local commit, not a mutable worktree or arbitrary
  branch. Commit selection and workflow protection remain operator responsibilities.
- Use review exit status for relaxation, shadow status for the baseline gate.
  Candidate results never silently replace active enforcement.
- Prefer JSON/Markdown and a local evidence pack to a model-generated explanation.
  No provider account or ongoing API bill is required for data processing.
- Create a new pack directory exclusively, publish a hash manifest last, retain
  incomplete output for inspection and never reuse old evidence as a new result.
- Keep API keys, broad filesystem tools, case actions and AI out of this extension.
  Add them only for a demonstrated user need with separate controls.
- Defer test execution to the owner as requested. Historical results are preserved
  with their source commits; no new PASS/coverage/performance claim is inferred.
