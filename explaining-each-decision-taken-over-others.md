# Vanguard: accepted decisions

1. Retain Go and normalized input: no scanner dependencies; native scanner adapters deferred.
2. Reject identical normalized findings: avoid accidental double-counting, while distinct normalized finding objects still count.
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

## Vanguard onboarding decisions

Keep example report/policy downloads because those are the actual evaluation
inputs. Remove unused account/connection checks and filesystem selection: local
policy evaluation works offline. Keep optional Slack exclusively in the CLI so
a browser demo or evaluation cannot unexpectedly send a notification.

Use an intentionally failing demo to explain a valid gate outcome versus a tool
error. Complete the guide after any valid evaluation, including FAIL. Store only
completion; keep sensitive reports and decisions in session memory. Invalid
evaluation clears the decision so export cannot silently return old evidence.

Use server-rendered multipart forms for pasted/file input without a frontend
runtime. Retain Referrer-Policy same-origin so ordinary local form posts work
while strict guards still reject foreign and opaque origins. Replace stale
native-scanner diagrams with the source-backed flow in architecture.md.
