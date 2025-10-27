---
title: "[0004] Session Management"
description: >
    Selection of session management strategy for maintaining user authentication state in the journey tracking API
type: docs
weight: 4
status: "accepted"
date: 2025-01-26
deciders: []
consulted: []
informed: []
---

## Context and Problem Statement

Following the decision to implement SSO authentication using OAuth 2.0 with OpenID Connect providers (Google, Facebook, Apple) as defined in [ADR-0003](0003-oauth2-oidc-provider-selection.md), we need to determine how to manage user sessions after successful authentication. After a user authenticates via an OAuth2/OIDC provider and receives a JWT access token, we must decide how the API will maintain authentication state for subsequent requests. Should we rely solely on JWT tokens, implement server-side sessions, or use a hybrid approach? How do we handle token refresh, expiration, and security considerations while maintaining good user experience?

## Decision Drivers

* Security and token validation overhead per request
* User experience and session persistence expectations
* Token expiration and refresh token management
* Scalability and stateless API design goals
* Integration complexity with OAuth2/OIDC providers
* Support for token revocation and logout
* Mobile app considerations (token storage and refresh)
* API performance and latency requirements
* Compliance requirements for session security
* Development and operational complexity

## Considered Options

* [option 1] Stateless JWT-only approach (no server-side sessions)
* [option 2] Server-side sessions with session store (Redis/database)
* [option 3] Hybrid approach (JWT + refresh token with server-side tracking)
* [option 4] OAuth2 proxy pattern (delegated session management)

## Decision Outcome

Chosen option: "[option 1] Stateless JWT-only approach (no server-side sessions)", because it provides simpler and faster development without requiring additional infrastructure for session storage. The API can validate JWT tokens on each request without database lookups, enabling a fully stateless architecture that scales horizontally without session affinity concerns. This approach eliminates the operational overhead of managing a session store while maintaining security through proper JWT validation and reasonable token expiration policies.

### Consequences

* Good, because fully stateless API enables horizontal scaling without session affinity
* Good, because no session store infrastructure required (no Redis/database for sessions)
* Good, because lower latency - no session lookup on each request
* Good, because simpler deployment and operations - no session store to maintain
* Good, because JWT contains user identity and can include claims for authorization
* Good, because aligns with microservices best practices (stateless services)
* Good, because faster initial development - no session management code required
* Bad, because difficult to revoke tokens before expiration (logout, security breach scenarios)
* Bad, because no server-side control over active sessions
* Bad, because token refresh requires client-side logic (handling refresh tokens)
* Bad, because need to balance token lifetime (short = frequent refresh, long = security risk)
* Neutral, because JWT validation requires cryptographic signature check on every request
* Neutral, because can add server-side session tracking later if needed

### Confirmation

Implementation will be confirmed through:
- Load testing verifying session lookup/validation performance meets latency requirements
- Security audit confirming token validation and session security practices
- Integration tests covering authentication, token refresh, and session expiration scenarios
- Logout functionality testing including token revocation where applicable
- Monitoring dashboards tracking session-related metrics (active sessions, refresh rates, failures)

## Pros and Cons of the Options

### [option 1] Stateless JWT-only approach (no server-side sessions)

Use only JWT tokens from OAuth2/OIDC providers for authentication on every request. No server-side session state.

* Good, because fully stateless API enables horizontal scaling without session affinity
* Good, because no session store infrastructure required (no Redis/database for sessions)
* Good, because lower latency (no session lookup on each request)
* Good, because simpler deployment and operations (no session store to maintain)
* Good, because JWT contains user identity and can include claims for authorization
* Good, because aligns with microservices best practices (stateless services)
* Neutral, because requires JWT validation on every request (cryptographic signature check)
* Neutral, because JWT size may increase request payload (typically 1-2KB with claims)
* Bad, because difficult to revoke tokens before expiration (logout, security breach)
* Bad, because no server-side control over active sessions
* Bad, because token refresh requires client-side logic (handling refresh tokens)
* Bad, because short-lived tokens require frequent refresh (user experience impact)
* Bad, because long-lived tokens increase security risk if compromised

### [option 2] Server-side sessions with session store (Redis/database)

Traditional server-side sessions stored in Redis or database. Session ID in cookie or header.

* Good, because immediate token revocation capability (logout, security events)
* Good, because full control over active sessions and session lifecycle
* Good, because can track session metadata (last access time, IP address, device info)
* Good, because small session identifier in requests (minimal overhead)
* Good, because easier to implement session timeout and idle timeout policies
* Good, because supports gradual user permission changes (no waiting for token expiration)
* Neutral, because session store becomes single source of truth for authentication state
* Bad, because requires session store infrastructure (Redis cluster, PostgreSQL)
* Bad, because session lookup on every request (additional latency, database load)
* Bad, because session store becomes potential single point of failure
* Bad, because requires session affinity or distributed session store for horizontal scaling
* Bad, because increased operational complexity (session store monitoring, backups)
* Bad, because cross-origin/mobile app considerations (CORS, cookie handling)

### [option 3] Hybrid approach (JWT + refresh token with server-side tracking)

Use short-lived JWT access tokens with refresh tokens. Track refresh tokens server-side for revocation.

* Good, because balances stateless benefits with revocation capability
* Good, because short-lived access tokens limit exposure if compromised (15-60 min typical)
* Good, because refresh token tracking enables logout and security controls
* Good, because access token validation is fast (no database lookup)
* Good, because can revoke refresh tokens immediately (security events)
* Good, because aligns with OAuth2 best practices
* Good, because scales well (most requests use access token, refresh is rare)
* Neutral, because requires both JWT validation and refresh token management logic
* Neutral, because still needs database/cache for refresh token storage
* Bad, because increased implementation complexity (token refresh flow)
* Bad, because client must implement token refresh logic
* Bad, because potential for race conditions during token refresh
* Bad, because mobile apps must securely store refresh tokens

### [option 4] OAuth2 proxy pattern (delegated session management)

Use OAuth2 proxy (like oauth2-proxy) to handle authentication and session management upstream.

* Good, because delegates session complexity to specialized component
* Good, because consistent session handling across multiple services
* Good, because proxy handles token refresh automatically
* Good, because can add authentication to any backend service without code changes
* Good, because well-tested open-source solutions available
* Neutral, because adds another component to infrastructure
* Bad, because introduces additional network hop (latency)
* Bad, because proxy becomes critical dependency (single point of failure)
* Bad, because less control over authentication flow and customization
* Bad, because may complicate debugging and error handling
* Bad, because requires learning and configuring proxy-specific behavior
* Bad, because may not support all OAuth2/OIDC provider features

## More Information

### Related Decisions and Documentation

- [ADR-0002: SSO Authentication Strategy](0002-sso-authentication-strategy.md) - OAuth2/OIDC decision
- [ADR-0003: OAuth2/OIDC Provider Selection](0003-oauth2-oidc-provider-selection.md) - Google, Facebook, Apple
- [OpenFGA Research](../../analysis/open-source/openfga.md) - Fine-grained authorization (separate from sessions)

### Security Considerations

**Token Storage:**
- Browser: httpOnly, secure, SameSite cookies vs. localStorage
- Mobile: Secure keychain/keystore for refresh tokens
- Never store tokens in localStorage if XSS is a concern

**Token Expiration:**
- Access tokens: Short-lived (15-60 minutes)
- Refresh tokens: Long-lived (days to months) with rotation
- Provider-specific: Apple refresh tokens valid for 6 months

**Revocation Requirements:**
- Immediate logout: Requires server-side tracking or token revocation endpoint
- Security incidents: Need ability to invalidate all sessions for a user
- Permission changes: May need to force token refresh

### Performance Impact

**JWT Validation Cost:**
- Cryptographic signature verification: ~0.1-1ms per request
- JWKS caching: Fetch once per key rotation (24-48 hours typical)
- Can optimize with local caching of public keys

**Session Store Lookup:**
- Redis: ~1-2ms per lookup (sub-millisecond on same host)
- PostgreSQL: ~5-20ms depending on load and indexing
- Can optimize with connection pooling and caching

### OAuth2/OIDC Provider Token Lifetimes

**Google:**
- Access token: 1 hour
- Refresh token: No expiration (can be revoked)

**Facebook:**
- Short-lived token: 1-2 hours
- Long-lived token: 60 days
- No refresh tokens (exchange short for long-lived)

**Apple:**
- Access token: 10 minutes
- Refresh token: 6 months (must re-authenticate after)
- ID token: Separate from access token

### Implementation Considerations

**Stateless JWT Approach:**
```go
// Middleware validates JWT on every request
func ValidateJWT(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractToken(r)
        claims, err := validateJWTSignature(token)
        if err != nil {
            http.Error(w, "Unauthorized", 401)
            return
        }

        ctx := context.WithValue(r.Context(), "userID", claims.Subject)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Server-Side Session Approach:**
```go
// Middleware looks up session
func ValidateSession(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        sessionID := extractSessionID(r)
        session, err := sessionStore.Get(ctx, sessionID)
        if err != nil || session.Expired() {
            http.Error(w, "Unauthorized", 401)
            return
        }

        ctx := context.WithValue(r.Context(), "session", session)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Hybrid Approach:**
```go
// Use access token for requests, refresh token for renewal
func ValidateAccessToken(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        accessToken := extractAccessToken(r)
        claims, err := validateJWT(accessToken)
        if err != nil {
            // Check if refresh token is available
            if refreshToken := extractRefreshToken(r); refreshToken != "" {
                // Validate refresh token and issue new access token
                newAccessToken, err := refreshAccessToken(refreshToken)
                if err == nil {
                    // Continue with new access token
                    claims, _ = validateJWT(newAccessToken)
                }
            }
        }

        ctx := context.WithValue(r.Context(), "userID", claims.Subject)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Mobile App Considerations

- Mobile apps should use authorization code flow with PKCE
- Store refresh tokens in secure keychain/keystore
- Handle token refresh gracefully (401 → refresh → retry)
- Consider token renewal before expiration (proactive refresh)
- Handle offline scenarios (cached tokens, sync on reconnect)

### Cross-Origin Considerations

- SPA (Single Page App): Must handle CORS, consider token storage
- httpOnly cookies require SameSite configuration
- Mobile apps: Use custom URL schemes for OAuth callback
- Consider API subdomain strategy (api.example.com) for cookie sharing
