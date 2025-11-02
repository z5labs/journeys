---
title: Research & Design
type: docs
---

## R&D Process

The Research & Design process follows a structured workflow to ensure comprehensive analysis and documentation of user experiences, technical solutions, and implementation details.

### Process Steps

1. **Document the User Journey**
   - Create a user journey document for the specific user experience
   - Include flow diagrams using Mermaid to visualize user interactions
   - Define prioritized technical requirements (P0/P1/P2)
   - Use the `/new-user-journey` command to create standardized documentation

2. **Design the Solution**
   - Create an ADR that designs a solution to implement the user journey
   - Identify and document:
     - Additional ADRs needed for specific components
     - APIs that need to be defined
     - User interface flows (mobile, web, etc.)
     - Data flow from user to end systems (database, notification system, etc.)
   - Capture the complete system architecture and integration points

3. **Document Component ADRs**
   - Create ADRs for specific technical components identified in the solution design
   - Examples: authentication strategy, session management, account linking, data storage
   - Use the `/new-adr` command to create standardized MADR 4.0.0 format documents
   - Document technical decisions with context, considered options, and consequences

4. **Document Required APIs**
   - For each API endpoint identified in the solution, create comprehensive API documentation
   - Use the `/new-api-doc` command to create standardized documentation
   - Include:
     - Request/response schemas
     - Authentication requirements
     - Business logic flows (Mermaid diagrams)
     - Error responses and status codes
     - Example curl requests

5. **Document API Implementation**
   - For each documented API, create an ADR describing the implementation approach
   - Document technical decisions including:
     - Programming language selection
     - Framework and libraries
     - Architecture patterns
     - Testing strategy
   - Example: ADR-0006 documents the tech stack for API development (z5labs/humus framework)

6. **Design User Interface**
   - Create UI/UX designs for the user journey
   - Ensure designs align with the documented user flows and API contracts
   - Consider platform-specific requirements (mobile, web, desktop)

### Documentation Structure

The R&D documentation is organized into the following sections:

- **[User Journeys](user-journeys/)** - User experience flows with technical requirements
- **[ADRs](adrs/)** - Architectural Decision Records documenting technical decisions
- **[APIs](apis/)** - REST API endpoint documentation with schemas and examples
- **[Analysis](analysis/)** - Research and analysis of technologies and solutions