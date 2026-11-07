# Vanguard: accepted decisions

1. Retain Go and normalized input: no scanner dependencies; native scanner adapters deferred.
2. Reject identical normalized findings: avoid accidental double-counting, while distinct occurrences still count.
3. Capped in-memory reports: simpler deterministic evaluation; not an unlimited streaming claim.
4. Hash exact policy/input bytes: reproducibility identification, not authenticity or compliance proof.
5. CLI Slack after evaluation: optional notification cannot turn a failed gate into success.
6. Session-only UI: avoids storing potentially sensitive reports without a retention requirement.

Common choice: standard-library embedded UI over a SPA/framework. Fewer runtime
dependencies and straightforward packaging; trades rich asynchronous interactions
for full-page forms and serialized actions. Revisit after measured user need.

Alternatives deferred: ServiceNow adapters, cloud hosting, user accounts, automatic
remediation, generalized rule engines and AI. They expand operational scope without
improving the two selected workflows. Revisit only with a specific requirement.

Performance tradeoffs: bounds are intentional; startup and real-workload latency
must be measured before making claims. Source and tests are authoritative.
