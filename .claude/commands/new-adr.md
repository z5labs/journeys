---
description: Create a new Markdown Architectural Decision Record (MADR)
---

Create a new Markdown Architectural Decision Record (MADR) following the MADR 4.0.0 standard.

**Requirements:**
1. All ADRs MUST be created in `docs/content/architecture/decisions/`
2. Use naming format: `NNNN-title-with-dashes.md` where NNNN is zero-padded sequential (e.g., `0001-use-madr.md`)
3. Ask the user for the ADR title/topic if not provided
4. Determine the next sequential number by checking existing ADRs in `docs/content/architecture/decisions/`
5. Fill in today's date in YYYY-MM-DD format
6. Set initial status to "proposed"

**MADR Template to use:**

```markdown
---
title: "[short title of solved problem and solution]"
description: >
    [short summary of the context and problem statement]
type: docs
weight: [NNNN]
status: "proposed"
date: YYYY-MM-DD
deciders: []
consulted: []
informed: []
---

## Context and Problem Statement

[Describe the context and problem statement, e.g., in free form using two to three sentences or in the form of an illustrative story.
 You may want to articulate the problem in form of a question and add links to collaboration boards or issue management systems.]

<!-- This is an optional element. Feel free to remove. -->
## Decision Drivers

* [driver 1, e.g., a force, facing concern, ...]
* [driver 2, e.g., a force, facing concern, ...]
* ...

## Considered Options

* [option 1]
* [option 2]
* [option 3]
* ...

## Decision Outcome

Chosen option: "[option 1]", because [justification. e.g., only option, which meets k.o. criterion decision driver | which resolves force force | ... | comes out best (see below)].

<!-- This is an optional element. Feel free to remove. -->
### Consequences

* Good, because [positive consequence, e.g., improvement of one or more desired qualities, ...]
* Bad, because [negative consequence, e.g., compromising one or more desired qualities, ...]
* ...

<!-- This is an optional element. Feel free to remove. -->
### Confirmation

[Describe how the implementation of/compliance with the ADR is confirmed. E.g., by a review or an ArchUnit test.
 Although we classify this element as optional, it is included in most ADRs.]

<!-- This is an optional element. Feel free to remove. -->
## Pros and Cons of the Options

### [option 1]

[example | description | pointer to more information | ...]

* Good, because [argument a]
* Good, because [argument b]
<!-- use "neutral" if the given argument weights neither for good nor bad -->
* Neutral, because [argument c]
* Bad, because [argument d]
* ...

### [option 2]

[example | description | pointer to more information | ...]

* Good, because [argument a]
* Good, because [argument b]
* Neutral, because [argument c]
* Bad, because [argument d]
* ...

### [option 3]

[example | description | pointer to more information | ...]

* Good, because [argument a]
* Good, because [argument b]
* Neutral, because [argument c]
* Bad, because [argument d]
* ...

<!-- This is an optional element. Feel free to remove. -->
## More Information

[You can use this section to provide additional evidence/confidence for the decision outcome, such as:
 - Links to other decisions and resources
 - Related requirements
 - Related principles
 - ...]
```

**Status values:** `proposed` | `accepted` | `rejected` | `deprecated` | `superseded by ADR-XXXX`

**Reference:** MADR 4.0.0 - https://adr.github.io/madr/
