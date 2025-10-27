---
title: "[0001] Use MADR for architectural decision records"
description: >
    Adopt the Markdown Architectural Decision Records (MADR) format for documenting architectural decisions in this project.
type: docs
weight: 1
status: "accepted"
date: 2025-10-25
deciders: []
consulted: []
informed: []
---

## Context and Problem Statement

As the journeys project grows, we need a consistent way to document architectural decisions, their context, and rationale. This documentation helps both current and future team members understand why certain technical choices were made, and provides a historical record of decision-making processes.

How should we document architectural decisions in a way that is accessible, maintainable, and integrated with our development workflow?

## Decision Drivers

* Need for lightweight, version-controlled documentation that lives with the code
* Desire for a standardized format that is easy to read and write
* Requirement to capture context, alternatives considered, and consequences
* Integration with existing tools (Git, Markdown, text editors)
* Low barrier to entry for creating and maintaining ADRs

## Considered Options

* MADR (Markdown Architectural Decision Records)
* Michael Nygard's ADR format
* RFC-style documents
* Wiki-based documentation
* No formal ADR process

## Decision Outcome

Chosen option: "MADR (Markdown Architectural Decision Records)", because it provides a well-structured template that captures all necessary information while remaining simple to use. MADR is an evolution of the original ADR format with better organization and clearer sections for pros/cons analysis.

### Consequences

* Good, because MADR provides a clear, standardized template that guides decision documentation
* Good, because Markdown files can be version controlled alongside code in Git
* Good, because MADR is widely adopted with good tooling and community support
* Good, because the format integrates well with Hugo for potential documentation site generation
* Good, because the template includes optional sections, allowing flexibility in detail level
* Neutral, because team members need to learn the MADR format and conventions
* Bad, because requires discipline to create ADRs consistently for significant decisions

### Confirmation

Compliance with this ADR will be confirmed through:
* Code review process ensuring significant architectural decisions have corresponding ADRs
* ADR files stored in `docs/content/architecture/decisions/` following the MADR template
* Presence of this ADR (0001) serving as the foundational example

## Pros and Cons of the Options

### MADR (Markdown Architectural Decision Records)

Based on Michael Nygard's ADR format with additional structure and clarity.

* Good, because provides comprehensive template with clear sections
* Good, because includes front matter compatible with static site generators like Hugo
* Good, because supports tracking of stakeholders (deciders, consulted, informed)
* Good, because explicitly captures pros/cons for each option considered
* Good, because widely adopted standard with version 4.0.0 specification
* Neutral, because more structured than simpler ADR formats (which can be good or bad depending on preference)

### Michael Nygard's ADR format

The original lightweight ADR format popularized by Michael Nygard.

* Good, because extremely simple and lightweight
* Good, because well-known and widely adopted
* Neutral, because less structured than MADR (more flexibility but less guidance)
* Bad, because lacks some useful sections like stakeholder tracking and detailed pros/cons

### RFC-style documents

Formal Request for Comments style documentation.

* Good, because very thorough and comprehensive
* Neutral, because well-suited for complex, cross-cutting decisions
* Bad, because heavyweight process not appropriate for all architectural decisions
* Bad, because higher barrier to entry, may discourage creating ADRs

### Wiki-based documentation

Using a wiki platform (Confluence, GitHub Wiki, etc.) for ADRs.

* Good, because easy to create and edit with web interface
* Good, because supports rich formatting and linking
* Bad, because separates documentation from code repository
* Bad, because version control and history tracking is less transparent than Git
* Bad, because requires separate platform/tool outside development workflow

### No formal ADR process

Ad-hoc decision documentation or no documentation.

* Good, because no overhead or process to follow
* Bad, because no consistent place to find decision rationale
* Bad, because knowledge is lost when team members leave
* Bad, because architectural decisions may not be documented at all

## More Information

* MADR 4.0.0 specification: https://adr.github.io/madr/
* ADRs are stored in `docs/content/architecture/decisions/` with naming format `NNNN-title-with-dashes.md`
* Use the `/new-adr` Claude Code command to create new ADRs following this template
* Status values: `proposed` | `accepted` | `rejected` | `deprecated` | `superseded by ADR-XXXX`
