# AI-summary evaluation

## Decision

**Do not add AI summaries to OpsMosaic now.** This task was an evaluation, not
authorization to integrate a model.

The existing product provides a practical tip or one random news/release item
with its publisher and original link. The user can open the source for context.
The `AI` interest selects AI-related headlines; it does not imply a desire for
generated summaries. The user selected:

- Evaluation only; no implementation or provider.
- No content sent to an external service and a `$0` budget.
- No AI calls, credentials, or automatic summaries.
- If separately approved in future, environment-variable credential supply,
  per-run explicit opt-in, headline/link fallback on provider failure, and only
  public headline/topic/source/link metadata as the maximum input. Article text
  is out of scope without separate approval.

## Why defer

Summarizing a headline and short feed metadata adds little beyond the title and
link and can invent context. Sending these values to a hosted model would still
disclose a user's selected topics and reading choices, create provider and
credential obligations, and potentially incur usage costs. No provider,
endpoint, model, or pricing plan was selected, so there is no responsible
provider-cost estimate; current runtime cost is zero.

Useful summaries require article text. That adds another fetch/extraction path
with site-specific failures, content and copyright considerations, potentially
larger data transfers to a provider, and more sanitization and attribution
work. It would weaken the current small, cacheable headline/link workflow.
Local models avoid external content sharing but bring model installation,
hardware, package size and lifecycle complexity that conflicts with the
standalone CLI goal.

## Revisit gate

Do not implement until the user explicitly chooses a provider and confirms the
data-sharing boundary, budget, credential mechanism, and opt-in behavior. Any
future integration must:

- Keep headlines and original links available without AI.
- Run only on an explicit per-run option and remain off by default.
- Label generated text and retain publisher attribution.
- Treat model output as untrusted terminal input.
- Enforce request timeouts, rate limits and a hard cost ceiling.
- On failure, warn and continue with the original headline/link.
- Never send article text, private content, or credentials unless separately
  approved.

A metadata-only prototype would be a small-to-medium change after those choices
are made; adding article extraction and hosted summarization would be a
materially larger product/security change. These are qualitative planning
estimates, not measured implementation or provider-cost results.
