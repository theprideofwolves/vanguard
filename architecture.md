# Vanguard: implemented architecture

Browser/CLI -> bounded policy/report intake -> parser -> severity evaluator ->
deterministic decision and hashes -> UI/JSON. Optional CLI Slack occurs after
the decision and cannot change it. No stored source, global background jobs,
database, scanner engine, or cloud credentials. UI decisions are session-local.
The parser retains the capped report in memory; this is not an unbounded stream.

## Operational boundaries
Private single-user tool. Browser access requires the launcher token. No remote
deployment, authentication roles, telemetry, continuous watch, or automatic
containment. Data corruption stops load instead of silently overwriting history.
Backups and third-party credentials remain the user's responsibility.

## Tests and evidence
Feature tests are alongside implementation, local-request tests in
internal/desktop, and integration contract tests beside the adapters. CI verifies
Windows/Linux builds and race checks. Mocks verify defined contracts; live
permissions, provider behavior and clean-machine interactive UI require manual
validation. Release packaging does not establish enterprise production readiness.
