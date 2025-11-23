# Copilot Instructions for Journeys Project

This journey tracking system combines a Hugo-based documentation site with a planned Go REST API built using the z5labs/humus framework.

## Project Architecture

### Current State
- **Documentation site**: Hugo with Docsy theme in `docs/`
- **API implementation**: Planned for `api/` directory (not yet created)
- **Custom tooling**: Claude Code commands in `.claude/commands/` for creating ADRs, user journeys, and API docs

### Key Design Decisions (ADRs)
- **ADR-0002**: OAuth2/OIDC authentication with external providers (no password management)
- **ADR-0003**: Google, Facebook, and Apple as OAuth providers
- **ADR-0004**: Stateless JWT-only sessions (no server-side storage)
- **ADR-0006**: z5labs/humus framework for REST API development

## REST API Development (z5labs/humus Framework)

### Framework-Specific Patterns

All API code will use the z5labs/humus framework. Key patterns:

**Application Bootstrap** (`api/main.go`):
```go
func main() {
    rest.Run(configFS, "config.yaml", app.Init)
}
```

**Endpoint Registration** (`api/app/app.go`):
```go
func Init() rest.InitFunc {
    return func(cfg Config) (*rest.Api, error) {
        api := rest.NewApi()
        endpoint.RegisterFooEndpoint(api)
        return api, nil
    }
}
```

**Endpoint Handler** (`api/endpoint/foo.go`):
```go
func RegisterFooEndpoint(api *rest.Api) {
    h := &fooHandler{log: humus.Logger("foo")}
    api.Route(http.MethodPost, "/v1/foo", rpc.NewOperation(
        rpc.ConsumeJson(rpc.ReturnJson(h)),
    ))
}

func (h *fooHandler) Handle(ctx context.Context, req *FooRequest) (*FooResponse, error) {
    // Business logic here
}
```

### Critical Rules
- One endpoint per file in `api/endpoint/`
- Use `humus.Logger("component")` for all logging (slog-based)
- All requests/responses use JSON with struct tags
- Wrap operations: `rpc.NewOperation(rpc.ConsumeJson(rpc.ReturnJson(handler)))`
- OpenTelemetry instrumentation is built-in via humus

## Documentation Workflow

### Hugo Site Management

**Build and serve**:
```bash
cd docs
hugo server              # Development with live reload
hugo --minify           # Production build to docs/public/
```

**CRITICAL - Hugo Module Dependencies**:
- **NEVER** run `go mod tidy`, `go get`, or any Go CLI commands on `docs/go.mod` or `docs/go.sum`
- These files manage Hugo module dependencies (Docsy theme), not Go code dependencies
- Use `hugo mod tidy` in the `docs/` directory to update Hugo modules
- Running Go CLI commands will break Hugo builds

**Content structure**:
- `docs/content/r&d/adrs/` - Architectural Decision Records (MADR 4.0.0 format)
- `docs/content/r&d/user-journeys/` - User flows with Mermaid diagrams and P0/P1/P2 requirements
- `docs/content/r&d/apis/` - REST API documentation with schemas and examples

### Creating Documentation

**Use Claude Code custom commands** (`.claude/commands/`):
- `/new-adr` - Create MADR-format ADR (categories: strategic, user-journey, api-design)
- `/new-user-journey` - Create user journey with Mermaid flow and prioritized requirements
- `/new-api-doc` - Create API endpoint documentation with request/response schemas

**Naming conventions**:
- ADRs: `NNNN-title-with-dashes.md` (e.g., `0007-user-registration.md`)
- User journeys: Same pattern as ADRs
- API docs: `v1-resource-action.md` (e.g., `v1-journey-create.md` for `POST /v1/journey`)

## Git Workflow

### Branch Naming
- Pattern: `story/issue-{number}/{short-description}`
- Example: `story/issue-54/ai-add-copilot-instructions`

### Commit Messages
- Format: `type(scope): description`
- Types: `story`, `docs`, `feat`, `fix`, `refactor`, `chore`
- Example: `story(issue-54): add copilot instructions`

### Issue Templates
- Use `.github/ISSUE_TEMPLATE/story.yaml` for story issues
- Title format: `story(subject): short description`
- Required: description and acceptance criteria

## Authentication & Authorization Integration

### OAuth2 Flow
1. User authenticates via external provider (Google/Facebook/Apple)
2. API receives JWT token from provider
3. API validates JWT and extracts user identity
4. For authorization decisions, API will call OPA or OpenFGA (not yet decided)

### User Identity Mapping
- Provider-specific IDs: `user:google:123456`, `user:facebook:789`
- Account linking allows multiple providers per user

## Go Code Standards

**Critical requirement**: Reference `.github/instructions/go.instructions.md` for comprehensive Go guidelines. Key points:

- **Package declarations**: Each `.go` file has EXACTLY ONE `package` line. Check existing files in directory before adding.
- **Error handling**: Always check errors; wrap with `fmt.Errorf("%w", err)` for context
- **Concurrency**: Use `sync.WaitGroup.Go()` method if `go >= 1.25`, otherwise use classic `Add/Done` pattern
- **HTTP clients**: Never store `*http.Request` in client structs; create fresh requests per call
- **Readers**: Most `io.Reader` streams are single-use; buffer once with `io.ReadAll`, then recreate via `bytes.NewReader`

## Development Tools

### MCP Server
- `.mcp.json` configures gopls MCP server for Go language support
- Provides workspace analysis and Go-specific tooling (when `api/` exists)

### CI/CD
- `docs.yaml`: Builds and deploys Hugo site to GitHub Pages on main branch changes
- `docs-preview.yaml`: Creates PR preview deployments

## Common Development Tasks

**Add new ADR**:
1. Use `/new-adr` command or manually create in `docs/content/r&d/adrs/`
2. Follow MADR 4.0.0 template with Hugo front matter
3. Set status: `proposed` → `accepted` | `rejected` | `deprecated`

**Document API endpoint**:
1. Use `/new-api-doc` command
2. Include Mermaid business logic flow
3. Document auth requirements (OAuth2 scopes)
4. Provide curl examples with sample responses

**Implement API endpoint** (future):
1. Create handler file in `api/endpoint/`
2. Define Request/Response structs with JSON tags
3. Implement `Handle(ctx, req) (resp, error)` method
4. Register via `endpoint.RegisterEndpoint(api)` in `api/app/app.go`
5. Use `humus.Logger()` for logging
