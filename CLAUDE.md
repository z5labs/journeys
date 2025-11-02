# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a journey tracking system project that will include a Go REST API built with the z5labs/humus framework. The application will allow users to create and manage journeys. Currently, the repository contains comprehensive documentation and architectural decisions. The API implementation will follow a clean architecture pattern with the main API code located in the `api/` directory (to be created).

## Architecture

### Project Structure
- `docs/` - Hugo-based documentation site using Docsy theme
  - `content/r&d/adrs/` - Architectural Decision Records
  - `content/r&d/user-journeys/` - User journey documentation with flow diagrams
  - `content/r&d/analysis/open-source/` - Technology research documents
- `.claude/` - Claude Code custom commands
  - `commands/` - Slash command definitions for `/new-adr`, `/new-user-journey`, etc.
- `.github/` - GitHub configuration
  - `ISSUE_TEMPLATE/` - Issue templates for stories and other types

**Planned API Structure** (not yet implemented):
- `api/` - Main Go module containing the REST API service
  - `main.go` - Entry point that loads configuration and initializes the REST API using the humus framework
  - `config.yaml` - Embedded configuration file
  - `app/` - Application initialization and configuration
    - `app.go` - Contains the `Init` function that sets up the REST API and registers endpoints
  - `endpoint/` - HTTP endpoint handlers (one file per endpoint)

### Key Frameworks and Libraries
- **github.com/z5labs/humus** - Primary REST framework providing:
  - REST API scaffolding via `rest.Run()` and `rest.Api`
  - RPC-style operation handling with JSON serialization (`rpc.NewOperation`, `rpc.ConsumeJson`, `rpc.ReturnJson`)
  - Embedded configuration loading from YAML
  - Logging utilities via `humus.Logger()`
- **OpenTelemetry** - Instrumentation and observability (metrics, traces, logs)
- **github.com/google/uuid** - UUID generation for resource IDs

### Application Flow
1. `main.go` embeds `config.yaml` and calls `rest.Run()` with the config and `app.Init`
2. `app.Init()` creates a new REST API instance and registers all endpoints
3. Endpoint handlers are structured with:
   - A handler struct containing dependencies (e.g., logger)
   - A registration function that adds the route to the API
   - Request/Response structs for JSON serialization
   - A `Handle` method implementing the business logic

### Endpoint Pattern
When implementing endpoints, follow this consistent pattern:
```go
func RegisterEndpoint(api *rest.Api) {
    h := &HandlerStruct{
        log: humus.Logger("endpoint"),
    }
    err := api.Route(
        http.Method,
        "/v1/path",
        rpc.NewOperation(
            rpc.ConsumeJson(
                rpc.ReturnJson(h),
            ),
        ),
    )
    if err != nil {
        panic(err)
    }
}
```

Handlers implement a `Handle(ctx context.Context, req *Request) (*Response, error)` method.

## Development Tools

### MCP Server Configuration
The repository includes a `.mcp.json` file configuring the gopls MCP server for enhanced Go language support in Claude Code. This provides Go-aware tools for workspace analysis, symbol search, file context, and diagnostics (when the `api/` directory is created and contains Go code).

## Development Commands

All commands should be run from the repository root unless otherwise specified.

### API Development (when implemented)
The Go module will be located in `api/`, so most Go commands need to be run from that directory:

```bash
cd api
go build -o ../dist/api .       # Build the API
go test ./...                    # Run all tests
go test ./endpoint              # Run tests for a specific package
go test -v ./...                # Run with verbose output
go test -run TestName ./...     # Run a specific test
go run .                        # Run the application
go fmt ./...                    # Format code
go vet ./...                    # Run static analysis
go mod tidy                     # Clean up dependencies
```

## Documentation

The project uses Hugo with the Docsy theme for documentation. All documentation source files are in the `docs/` directory.

### Hugo Commands

Run from the repository root:

```bash
cd docs
hugo server                     # Start development server with live reload
hugo                           # Build static site to docs/public/
hugo --minify                  # Build minified production site
npm run serve                  # Alternative: run dev server via npm
npm run build                  # Alternative: build via npm
```

### Documentation Structure

- `docs/content/` - Documentation content
  - `_index.md` - Home page
  - `r&d/` - Research & Development documentation
    - `adrs/` - Architectural Decision Records (ADRs)
    - `user-journeys/` - User Journey documentation with flow diagrams and technical requirements
    - `apis/` - REST API endpoint documentation with detailed request/response schemas
    - `analysis/open-source/` - Research documents on open-source technologies
      - `keycloak.md` - Keycloak OIDC provider research
      - `open-policy-agent.md` - OPA (policy-based authorization) research
      - `openfga.md` - OpenFGA (relationship-based authorization) research

The documentation site uses Hugo with the Docsy theme. Hugo configuration is in `docs/hugo.yaml`.

## Custom Slash Commands

- `/new-adr` - Create a new Markdown Architectural Decision Record (MADR) in `docs/content/r&d/adrs/` following the MADR 4.0.0 standard
- `/new-user-journey` - Create a new User Journey document in `docs/content/r&d/user-journeys/` with flow diagrams and prioritized technical requirements
- `/new-api-doc` - Create a new REST API endpoint documentation page in `docs/content/r&d/apis/` with comprehensive request/response schemas, business logic flows, and examples

## Project Conventions

### Issue Management and Git Workflow
- Story issues use the template at `.github/ISSUE_TEMPLATE/story.yaml`
- Story titles follow the format: `story(subject): short description`
- Stories require a description and acceptance criteria
- Branch naming convention: `story/issue-{number}/{short-description-with-dashes}`
  - Example: `story/issue-12/docs-document-adding-content-to-a-journey`
- Commit message format follows conventional commits: `type(scope): description`
  - Common types: `story`, `docs`, `feat`, `fix`, `refactor`, `chore`
  - Example: `story(issue-7): document creating a journey`
- Main branch: `main` (used as the base for pull requests)

### Architectural Decisions
- ADRs are stored in `docs/content/r&d/adrs/`
- Use the `/new-adr` command to create new ADRs
- ADRs follow MADR 4.0.0 format with Hugo front matter
- Naming convention: `NNNN-title-with-dashes.md` (zero-padded sequential numbering)
- Status values: `proposed` | `accepted` | `rejected` | `deprecated` | `superseded by ADR-XXXX`

**Key Decisions:**
- **ADR-0001** (accepted): Use MADR 4.0.0 for architectural decision records
- **ADR-0002** (accepted): SSO Authentication Strategy - OAuth2/OIDC with external providers
- **ADR-0003** (accepted): OAuth2/OIDC Provider Selection - Google, Facebook, and Apple
- **ADR-0004** (accepted): Session Management - Stateless JWT-only approach (no server-side sessions)
- **ADR-0005** (accepted): Account Linking - Strategy for linking multiple OAuth providers to single user account
- **ADR-0006** (proposed): API Development Tech Stack Selection - z5labs/humus framework for REST API

### User Journeys
- User journeys are stored in `docs/content/r&d/user-journeys/`
- Use the `/new-user-journey` command to create new journeys
- Each journey includes:
  - Mermaid flow diagram representing user interactions
  - Technical requirements with priority levels (P0/P1/P2)
  - Requirements categorized by type: Access Control, Rate Limits, Analytics, Data Storage, etc.
- Naming convention: `NNNN-title-with-dashes.md` (zero-padded sequential numbering)
- Priority levels help determine what must be in initial design (P0) vs. what can be phased (P1/P2)

**Existing User Journeys:**
- **0001**: User Registration - Initial account creation via OAuth2 provider
- **0002**: User Login via SSO - Authentication flow through external providers
- **0003**: Account Linking - Linking multiple OAuth2 providers to one account
- **0004**: Creating a Journey - How authenticated users create new journeys

### Code Organization (for future API implementation)
- Each endpoint should be in its own file in `api/endpoint/`
- Endpoint registration happens in `api/app/app.go` via `endpoint.RegisterEndpoint(api)`
- Use `humus.Logger("component_name")` for structured logging with slog
- Request/Response structs use JSON tags for serialization

### Documentation Organization
- ADRs document architectural decisions following MADR 4.0.0 format
- User journeys include Mermaid diagrams and prioritized technical requirements (P0/P1/P2)
- API documentation follows a comprehensive template with request/response schemas, authentication requirements, business logic flows (Mermaid), error responses, and curl examples
- Technology research documents provide analysis of potential solutions
- All documentation includes Hugo front matter for proper site generation

### API Documentation
- API docs are stored in `docs/content/r&d/apis/`
- Use the `/new-api-doc` command to create new API documentation
- Naming convention based on endpoint: remove leading slash, replace `/` with `-`, add action suffix
  - Examples: `POST /v1/journey` → `v1-journey-create.md`, `GET /v1/journey/{id}` → `v1-journey-get.md`
- Action suffixes: POST → `-create`, GET (with params) → `-get`, GET (without params) → `-list`, PUT → `-update`, PATCH → `-patch`, DELETE → `-delete`
- Status values: `draft` | `reviewed` | `published` | `deprecated`

**Existing API Documentation:**
- **GET /v1/auth/{provider}** - Initiate OAuth2 authentication flow with provider
- **GET /v1/auth/{provider}/callback** - Handle OAuth2 callback from provider
- **GET /v1/account/providers** - List OAuth2 providers linked to user account

## Authentication and Authorization Architecture

The project has decided on a separation-of-concerns approach for authentication and authorization:

### Authentication (ADR-0002, ADR-0003)
- **Strategy:** OAuth2/OIDC with external providers (Google, Facebook, Apple)
- **No custom password management:** Users authenticate via established identity providers
- **JWT tokens:** OAuth2/OIDC providers issue JWT access tokens containing user identity
- **Provider mapping:** User identifiers are mapped from provider-specific IDs (e.g., `user:google:123456`)

### Authorization (Research Phase)
Two authorization systems have been researched for fine-grained access control:

1. **Open Policy Agent (OPA)** - Policy-based authorization
   - Evaluates policies written in Rego language
   - Best for complex rules, attribute-based decisions, API gateway integration
   - See `docs/content/r&d/analysis/open-source/open-policy-agent.md`

2. **OpenFGA** - Relationship-based authorization (ReBAC)
   - Stores user-resource relationships, traverses graphs for decisions
   - Best for user-resource permissions, sharing, hierarchical organizations
   - Inspired by Google Zanzibar
   - See `docs/content/r&d/analysis/open-source/openfga.md`

### Integration Pattern
OAuth2/OIDC handles **authentication** ("Who are you?"), while OPA or OpenFGA handles **authorization** ("What can you access?"):

```
1. User authenticates with OAuth2 provider → JWT token
2. API validates JWT, extracts user identity
3. API calls authorization system to check permissions
4. Authorization system returns allow/deny decision
```

**Example:** User authenticates with Google, API validates JWT, then checks OpenFGA to see if `user:google:123` can view `journey:550e8400-...`
