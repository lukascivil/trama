---
description: Tell what a change would break across the repositories of a trama workspace and in which order to ship it. Read-only. Use when the user asks what a change affects, what would break, or before planning a change that touches an API, event or package another repository uses.
---

# Impact of a change

Arguments: the change, in the user's words ($ARGUMENTS). Work from the workspace root. Change nothing. Answer in the user's language.

1. Find what the change touches: the `id` (`GET /hello`, a topic, a package), or a part of one (a field of its response, a parameter), and the repository that provides it. Read the cards in `catalog/` (`response` lists the fields an endpoint returns, `request` the ones it accepts) and the provider's code when the request names code rather than an `id`.
2. Run `trama status`. If the card of the provider or of a consumer is missing or stale, say so first and suggest `/trama:map <repo>`: an answer from an outdated card can be wrong.
3. Run `trama uses "<id>"` for each `id`. It lists the provider, every consumer with its evidence, and pending consumers (`pending in`, from `from: unknown`).
4. Find every place the change touches, with `path:line`:
   - in the provider: the code that produces it, its tests and its docs;
   - in every consumer: open each evidence line, then search the repository for what the change touches. Keep only real uses of the provider's `id` or field; leave out names that only look the same (a local variable called `message`, `err.message`). Count `fields_used` as a hint, not as the full answer.
5. Decide for each consumer whether the change breaks it:
   - additive (a new optional field, a new route): compatible;
   - removing or renaming something it reads, changing a type, or adding or tightening a required input (the provider's `request` is what records those): breaking.
   Say also which tests would not catch the break (mocks of the provider, tests that decode with the provider's own types).
6. Answer with:
   - the affected repositories and every `path:line`;
   - breaking or compatible, and why;
   - the safe order. For a breaking change, expand → migrate → contract: the provider adds the new form and keeps the old one; each consumer moves to the new form; the old form is removed only after every consumer shipped;
   - pending relations that might also use the `id`: they are not proven, so name them as a risk;
   - that the cards will need `/trama:map` for the repositories whose `provides`, `request`, `response` or `fields_used` change.
