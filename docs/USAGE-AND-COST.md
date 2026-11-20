# Running Vanguard and sharing it with others

## Default: local and without AI

Vanguard accepts normalized JSON and YAML files. It does not need an AI API for
data processing, verdicts, policy comparison, shadow evaluation or review export.
Other users provide their own reports and policies and run the binary on their
own machine or CI runner. No hosted account, application login, subscription,
database or service operated by this project's author is required.

The review command additionally needs local Git. Building from source needs Go;
downloading/building dependencies can need network access. Evaluation after setup
works offline. Existing optional Slack notification is the only external action
and is not used by review/shadow mode. CI hosting and upstream scanner costs are
separate choices made by the operator. Do not promise unlimited free hosted CI.

## ChatGPT Plus and APIs

ChatGPT Plus does not include OpenAI API usage. API access/billing is separate.
See [OpenAI's Plus guidance](https://help.openai.com/en/articles/6950777-what-is-chatgpt-plus)
and [API quickstart](https://developers.openai.com/api/docs/quickstart).

Eligible participating apps can separately support using a ChatGPT plan through
Sign in with ChatGPT. That is not an ordinary API key or a connection implemented
in Vanguard. See [OpenAI's participating-app guidance](https://help.openai.com/en/articles/20001542-using-your-chatgpt-plan-in-other-apps-and-sites).

For personal study, you can manually give ChatGPT a sanitized review and ask for
help understanding it. That does not connect Vanguard to an API or make the gate
an AI agent. Do not upload private reports or tokens without authorization.

No model provider, API-key field or paid fallback is implemented in this version.

## If an assistant is useful later

Keep it optional and separate from the deterministic gate. Possible choices are
a local model or an operator's own provider account/key. Hosted API usage would
be billed under that account, subject to the provider's current terms and rates.
Do not share the author's key or route other users' traffic through the author's
account. Keep credentials in process environment/secret storage, never source,
browser-visible settings, prompts, screenshots or exported reviews.

Require explicit consent before sending report content off the local machine.
Set request/time/cost limits and preserve deterministic results if assistance is
unavailable. An assistant may explain evidence; it must not override the gate,
alter the policy, grant approval or submit actions without separate controls.
This is future design guidance, not a shipped integration.

## Reuse and support

For now, distribute source/build instructions and an unverified-feature notice.
Do not imply new behavior passed testing until manual results are recorded. A
public repository is not a promise of support, hosted availability or security
certification. A reuse license still needs the author's explicit selection; this
change does not add or assume one.

The project is evidence of focused security engineering. A review demonstration
and a clear explanation of its limits are more useful than an unused AI connector.
