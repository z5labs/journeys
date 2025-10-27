# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go REST API project built with the z5labs/humus framework. The application is a journey tracking system that allows users to create and manage journeys. The codebase follows a clean architecture pattern with the main API code located in the `api/` directory.

## Architecture

### Project Structure
- `api/` - Main Go module containing the REST API service
  - `main.go` - Entry point that loads configuration and initializes the REST API using the humus framework
  - `config.yaml` - Embedded configuration file (currently empty but loaded at runtime)
  - `app/` - Application initialization and configuration
    - `app.go` - Contains the `Init` function that sets up the REST API and registers endpoints
  - `endpoint/` - HTTP endpoint handlers
    - `create_journey.go` - Handler for creating new journeys (POST /v1/journey)

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
Endpoints follow a consistent pattern (see `endpoint/create_journey.go:20-36`):
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

## Development Commands

All commands should be run from the repository root unless otherwise specified.

### Working Directory
The Go module is located in `api/`, so most Go commands need to be run from that directory or with the appropriate path.

### Building
```bash
cd api
go build -o ../dist/api .
```

### Running Tests
```bash
cd api
go test ./...                    # Run all tests
go test ./endpoint              # Run tests for a specific package
go test -v ./...                # Run with verbose output
go test -run TestName ./...     # Run a specific test
```

### Running the Application
```bash
cd api
go run .
```

### Linting and Code Quality
Standard Go tools:
```bash
cd api
go fmt ./...                    # Format code
go vet ./...                    # Run static analysis
```

### Dependency Management
```bash
cd api
go mod tidy                     # Clean up dependencies
go mod download                 # Download dependencies
go list -m all                  # List all dependencies
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
    - `analysis/open-source/` - Research documents on open-source technologies
      - `keycloak.md` - Keycloak OIDC provider research
      - `open-policy-agent.md` - OPA (policy-based authorization) research
      - `openfga.md` - OpenFGA (relationship-based authorization) research

## Custom Slash Commands

- `/new-adr` - Create a new Markdown Architectural Decision Record (MADR) in `docs/content/r&d/adrs/` following the MADR 4.0.0 standard
- `/new-user-journey` - Create a new User Journey document in `docs/content/r&d/user-journeys/` with flow diagrams and prioritized technical requirements

## Project Conventions

### Issue Management
- Story issues use the template at `.github/ISSUE_TEMPLATE/story.yaml`
- Story titles follow the format: `story(subject): short description`
- Stories require a description and acceptance criteria

### Architectural Decisions
- ADRs are stored in `docs/content/r&d/adrs/`
- Use the `/new-adr` command to create new ADRs
- ADRs follow MADR 4.0.0 format with Hugo front matter
- Naming convention: `NNNN-title-with-dashes.md` (zero-padded sequential numbering)
- Status values: `proposed` | `accepted` | `rejected` | `deprecated` | `superseded by ADR-XXXX`

**Key Decisions:**
- **ADR-0002** (accepted): SSO Authentication Strategy - OAuth2/OIDC with external providers
- **ADR-0003** (accepted): OAuth2/OIDC Provider Selection - Google, Facebook, and Apple
- **ADR-0004** (accepted): Session Management - Stateless JWT-only approach (no server-side sessions)

### User Journeys
- User journeys are stored in `docs/content/r&d/user-journeys/`
- Use the `/new-user-journey` command to create new journeys
- Each journey includes:
  - Mermaid flow diagram representing user interactions
  - Technical requirements with priority levels (P0/P1/P2)
  - Requirements categorized by type: Access Control, Rate Limits, Analytics, Data Storage, etc.
- Naming convention: `NNNN-title-with-dashes.md` (zero-padded sequential numbering)
- Priority levels help determine what must be in initial design (P0) vs. what can be phased (P1/P2)

### Code Organization
- Each endpoint should be in its own file in `api/endpoint/`
- Endpoint registration happens in `api/app/app.go:20` via `endpoint.RegisterEndpoint(api)`
- Use `humus.Logger("component_name")` for structured logging with slog
- Request/Response structs use JSON tags for serialization

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
