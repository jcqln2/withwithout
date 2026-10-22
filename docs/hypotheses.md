# Pre-registered hypotheses

**Status: pre-registered on 2026-10-22, before any treatment data was collected.**

Do not edit the claim text, the test, or the primary metric in the table below after this file is committed. A correction to a source quotation, or a note that a claim was withdrawn by Optimum, goes in a dated addendum under the table. It does not replace the row.

Source of the claim wording: Optimum's public site and docs, as stated in `docs/PLAN.md` section 1. These rows are the claims this project will test. They are not this project's findings.

Decision rule: for each hypothesis, report the effect size, a 95% confidence interval, and one verdict: **supported**, **not supported**, or **inconclusive**. Inconclusive is a valid result.

| ID | Claim (source: Optimum site / docs) | How we test it | Primary metric |
|----|-------------------------------------|----------------|----------------|
| H1 | mump2p propagates blocks 2-4x faster than Gossipsub on average | Track A (controlled A/B) | Block arrival delay vs. slot start; paired delta per block |
| H2 | Optimum network sees a block first ~90% of the time | Track A + C | % of blocks where the treatment node receives it before control nodes |
| H3 | Fewer missed attestations / better head votes | Track B (cohort DiD) | Head-vote correctness rate, missed attestation rate |
| H4 | Fewer missed block proposals | Track B | Missed proposal rate |
| H5 | Higher validator rewards / better MEV capture | Track B | Per-validator reward per epoch (normalized), proposer block value where available |
| H6 | Lower bandwidth use | Track A | Bytes in/out per node per hour |

No verdicts yet. The scorecard stays empty until the pre-registered analyses run.

## Addenda

None.
