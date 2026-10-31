---
description: Tell what a change would break across the repositories of a trama workspace and in which order to ship it. Read-only. Use when the user asks what a change affects, what would break, or before planning a change that touches an API, event or package another repository uses.
---

# Impact of a change

Arguments: the change, in the user's words ($ARGUMENTS). Work from the workspace root. Change nothing.

1. Run `trama status`. If a card involved is missing or stale, say so first and suggest `/trama:map <repo>`: an answer from an outdated card can be wrong.
2. Find what the change touches: the `id` (`GET /hello`, a topic, a package) and the repository that provides it. Read the cards in `catalog/` and the provider's code when the request names code rather than an `id`.
3. Run `trama uses "<id>"` for each `id`. It lists the provider and every consumer with their evidence.
4. Confirm in the code of every consumer: open each evidence line, then search the repository for what the change touches (a field, a parameter, a status code). List every place with `path:line`. Count `fields_used` as a hint, not as the full answer.
5. Decide for each consumer whether the change breaks it:
   - additive (a new optional field, a new route): compatible;
   - removing or renaming something it reads, changing a type or a required input: breaking.
6. Answer with:
   - the affected repositories and every `path:line`;
   - breaking or compatible, and why;
   - the safe order. For a breaking change, expand → migrate → contract: the provider adds the new form and keeps the old one; each consumer moves to the new form; the old form is removed only after every consumer shipped;
   - pending relations (`from: unknown`) that might also use the `id`: they are not proven, so name them as a risk.
