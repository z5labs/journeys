---
title: "OAuth State and PKCE Storage Alternatives"
description: >
    Analysis of different approaches for storing OAuth state parameters and PKCE values during authorization flows, including encrypted cookies, database storage, and stateless tokens.
type: docs
weight: 1
---

## Overview

This document analyzes alternatives to Redis for storing OAuth state parameters and PKCE (Proof Key for Code Exchange) values during OAuth 2.0 authorization flows. The analysis evaluates trade-offs between different storage mechanisms to inform implementation decisions for user authentication flows.

## Why OAuth State and PKCE Need Temporary Storage

### Data Requirements

During OAuth authorization flows, the following data must be temporarily stored:

**OAuth State Parameter:**
- Random value for CSRF protection (must match between initiation and callback)
- Original destination URL for post-login redirect
- Session context (registration vs. login flow)
- Generated in `/v1/auth/{provider}`, validated in `/v1/auth/{provider}/callback`

**PKCE Values:**
- `code_verifier`: Random string (43-128 characters) generated at initiation
- `code_challenge`: SHA256 hash of code_verifier sent to OAuth provider
- The code_verifier must be available during callback to exchange authorization code for tokens

**Additional Context:**
- Provider identifier (google, facebook, apple)
- Redirect URI used in the flow
- Timestamp for expiration checking

### Time-to-Live (TTL)

**Standard TTL: 10 minutes**
- OAuth flows typically complete in seconds
- 10 minutes provides generous buffer for user authentication
- Authorization codes expire quickly (1-10 minutes depending on provider)
- State should expire to prevent replay attacks

### Read/Write Pattern

**Write Phase (Initiation - `GET /v1/auth/{provider}`):**
1. Generate random state value
2. Generate PKCE code_verifier
3. Store: state → {code_verifier, redirect_uri, timestamp, provider, original_destination}
4. Redirect user to OAuth provider

**Read Phase (Callback - `GET /v1/auth/{provider}/callback`):**
1. Receive state parameter from provider
2. Look up stored data by state key
3. Validate state matches and hasn't expired
4. Retrieve code_verifier for token exchange
5. Delete state data (single use)

### Security Requirements

1. **CSRF Protection**: State must be unpredictable and bound to user session
2. **Single Use**: State should be deleted after successful use
3. **Confidentiality**: PKCE code_verifier must not leak (prevents code interception attacks)
4. **Integrity**: Data must not be tampered with
5. **Expiration**: Must expire to prevent replay attacks

---

## Storage Alternatives

### 1. Encrypted Cookies (⭐ Recommended)

**Description:**
Store OAuth state and PKCE data in an encrypted, httpOnly cookie on the user's browser. No server-side storage required.

**How It Works:**
1. Encrypt state and PKCE data server-side using AES-256-GCM
2. Store encrypted payload in httpOnly, secure, SameSite cookie
3. On callback, decrypt and validate cookie contents
4. Delete cookie after successful use

**Implementation Pattern:**
```
Cookie: oauth_state=encrypted_and_signed_payload
Payload: {state, code_verifier, redirect_uri, timestamp, provider}
```

**Security Features:**
- Strong encryption (AES-256-GCM) with server-side key
- HMAC signature to prevent tampering
- httpOnly flag (prevents JavaScript access)
- Secure flag (HTTPS only)
- SameSite=Lax or Strict (CSRF protection)
- Max-Age=600 (10-minute expiration)

**Advantages:**
- ✅ Fully stateless - scales horizontally without session affinity
- ✅ No infrastructure dependencies (no Redis, no database)
- ✅ Simple deployment - works immediately
- ✅ Fast - no network round-trip to storage layer
- ✅ No single point of failure
- ✅ Aligns with stateless JWT architecture (ADR-0004)

**Disadvantages:**
- ❌ Cookie size limit (~4KB) - sufficient for OAuth data (~500 bytes)
- ❌ Cookie sent on every request to domain (minor bandwidth overhead)
- ❌ Cannot revoke before expiration (10-minute exposure window)
- ❌ Browser must support cookies (edge case: disabled cookies)
- ❌ Harder to debug (encrypted payload)
- ❌ Encryption key management required

**Best For:**
- Getting started quickly
- Single-server or multi-server deployments
- Stateless architecture
- When infrastructure simplicity is priority

**Implementation Sketch (Go):**
```go
// Initiate OAuth flow
func initiateOAuth(w http.ResponseWriter, r *http.Request) {
    state := generateRandomState() // 32 bytes
    codeVerifier := generatePKCEVerifier() // 43-128 chars

    payload := StatePayload{
        State:        state,
        CodeVerifier: codeVerifier,
        RedirectURI:  r.FormValue("redirect_uri"),
        Provider:     r.PathValue("provider"),
        Timestamp:    time.Now().Unix(),
        Nonce:        generateNonce(),
    }

    // Encrypt and sign
    encryptedCookie := encryptAndSign(payload, secretKey)

    http.SetCookie(w, &http.Cookie{
        Name:     "oauth_state",
        Value:    encryptedCookie,
        MaxAge:   600, // 10 minutes
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        Path:     "/v1/auth",
    })

    redirectToProvider(w, state, codeVerifier)
}

// Handle OAuth callback
func handleCallback(w http.ResponseWriter, r *http.Request) {
    stateParam := r.URL.Query().Get("state")

    cookie, err := r.Cookie("oauth_state")
    if err != nil {
        return fmt.Errorf("missing state cookie")
    }

    payload, err := decryptAndVerify(cookie.Value, secretKey)
    if err != nil {
        return fmt.Errorf("invalid state cookie")
    }

    // Validate
    if payload.State != stateParam {
        return fmt.Errorf("state mismatch - CSRF detected")
    }
    if time.Now().Unix() - payload.Timestamp > 600 {
        return fmt.Errorf("state expired")
    }

    // Use code_verifier for token exchange
    tokens, err := exchangeCode(r.URL.Query().Get("code"), payload.CodeVerifier)

    // Delete cookie (single use)
    http.SetCookie(w, &http.Cookie{
        Name:   "oauth_state",
        MaxAge: -1,
        Path:   "/v1/auth",
    })
}
```

---

### 2. Database with TTL (PostgreSQL)

**Description:**
Store OAuth state in the existing application database with expiration timestamps. Periodic cleanup removes expired entries.

**How It Works:**
1. Create `oauth_state` table with columns: state_key, data, expires_at
2. Insert state data with 10-minute expiration
3. On callback, SELECT and DELETE in transaction
4. Periodic cleanup job deletes expired rows

**Schema Example:**
```sql
CREATE TABLE oauth_state (
    state_key VARCHAR(64) PRIMARY KEY,
    code_verifier VARCHAR(128) NOT NULL,
    redirect_uri TEXT NOT NULL,
    provider VARCHAR(20) NOT NULL,
    original_destination TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_expires_at ON oauth_state(expires_at);
```

**Cleanup Strategies:**
1. **Trigger-based**: Automatic deletion on SELECT if expired
2. **pg_cron**: Scheduled cleanup within PostgreSQL
3. **Application job**: Periodic DELETE WHERE expires_at < NOW()
4. **Partitioning**: Drop old partitions daily

**Advantages:**
- ✅ No additional infrastructure (reuse existing database)
- ✅ Data persistence and transaction support
- ✅ Familiar technology for most teams
- ✅ Can query/debug state data easily
- ✅ ACID guarantees for consistency
- ✅ Can store unlimited data (no size limits)
- ✅ Can revoke state immediately (DELETE)

**Disadvantages:**
- ❌ Adds load to primary database
- ❌ Requires database connection per OAuth flow
- ❌ Slower than in-memory cache (5-20ms vs <1ms)
- ❌ Cleanup job required (PostgreSQL lacks native TTL)
- ❌ Connection pool contention with application queries
- ❌ Not ideal for high-frequency temporary data

**Best For:**
- Small to medium traffic (< 1000 OAuth flows/hour)
- When you want to avoid Redis infrastructure
- When you need to audit/debug OAuth flows
- Development and testing environments

---

### 3. Stateless Signed Tokens (JWT-like State)

**Description:**
Encode state data into a signed token and use the token itself as the OAuth state parameter. No storage required.

**How It Works:**
1. Encode state data in a JWT-like signed token
2. Use the signed token as the state parameter itself
3. On callback, verify signature and decode data
4. No storage required - data is in the URL

**Token Structure:**
```
state = base64(header).base64(payload).hmac_signature
payload = {code_verifier, redirect_uri, timestamp, provider, nonce}
```

**Security Requirements:**
- HMAC-SHA256 signature with server-side secret
- Include timestamp and nonce for replay protection
- Validate signature, timestamp, and nonce on callback
- Optional: encrypt payload with JWE for confidentiality

**Advantages:**
- ✅ Fully stateless - perfect horizontal scaling
- ✅ No storage infrastructure required
- ✅ No cleanup jobs needed
- ✅ Simple key management (single signing key)
- ✅ Works across multiple servers immediately
- ✅ Survives application restarts

**Disadvantages:**
- ❌ State parameter becomes large (200-500 bytes)
- ❌ URL length limits (~2000 characters in browsers)
- ❌ All data visible in URL (unless encrypted with JWE)
- ❌ Harder to debug (encoded token)
- ❌ Cannot revoke before expiration
- ❌ Must handle clock skew for timestamp validation

**Best For:**
- Highly scalable deployments
- When you want zero infrastructure dependencies
- Microservices architectures
- When state data is minimal

---

### 4. In-Memory Storage (Development Only)

**Description:**
Store OAuth state in application memory using Go's built-in map structures. Works only for single-instance deployments.

**How It Works:**
- Use Go's `sync.Map` or `map[string]StateData` with mutex
- Store state data in application memory
- Background goroutine cleans up expired entries
- Lost on application restart

**Implementation Considerations:**
- Thread-safe map with mutex or sync.Map
- TTL cleanup goroutine (check every minute)
- Lost on application restart (acceptable for OAuth)

**Advantages:**
- ✅ Simplest implementation (no external dependencies)
- ✅ Fastest performance (<0.1ms lookup)
- ✅ Zero infrastructure cost
- ✅ Perfect for development/testing
- ✅ Great for proof-of-concept

**Disadvantages:**
- ❌ Single server only (no horizontal scaling)
- ❌ Data lost on restart/crash (OAuth flows fail)
- ❌ Doesn't work with load balancers
- ❌ Memory usage grows if not cleaned properly
- ❌ Not suitable for production multi-server deployments

**Best For:**
- Development environments
- Proof-of-concept implementations
- Single-server self-hosted deployments (rare)
- Learning/testing OAuth flows

---

### 5. Redis / Memcached

**Description:**
Distributed in-memory cache for storing OAuth state with automatic TTL expiration.

**How It Works:**
- Store state data in cache with 10-minute expiration
- Cache handles TTL automatically
- On callback, GET and DELETE state

**Comparison:**

| Feature | Redis | Memcached |
|---------|-------|-----------|
| Data Types | Hash, List, Set, String | String only |
| Persistence | Optional (RDB/AOF) | None |
| Threading | Single-threaded | Multi-threaded |
| Replication | Built-in | External |
| Complexity | Higher | Lower |

**Advantages:**
- ✅ Excellent performance (sub-millisecond latency)
- ✅ Automatic expiration (native TTL)
- ✅ Scales horizontally
- ✅ Works with load balancers
- ✅ Can revoke state immediately
- ✅ Familiar technology

**Disadvantages:**
- ❌ Additional infrastructure to deploy and maintain
- ❌ Infrastructure cost ($15-50/month cloud, or self-hosted resources)
- ❌ Operational overhead (monitoring, updates, backups)
- ❌ Network latency (sub-millisecond but non-zero)
- ❌ Single point of failure (unless replicated)

**Best For:**
- Already using Redis/Memcached for other features
- High-traffic deployments (>10,000 OAuth flows/hour)
- When you need immediate revocation capability
- Enterprise deployments with dedicated ops team

---

## Comparison Matrix

### Scalability

| Approach | Single Server | Multi-Server | Load Balancer | Notes |
|----------|---------------|--------------|---------------|-------|
| Encrypted Cookies | ✅ Excellent | ✅ Excellent | ✅ No affinity needed | Fully stateless |
| Database (PostgreSQL) | ✅ Good | ✅ Good | ✅ No affinity needed | Connection pool limits |
| In-Memory | ✅ Good | ❌ No | ❌ Requires sticky sessions | Single server only |
| Stateless Tokens | ✅ Excellent | ✅ Excellent | ✅ No affinity needed | Fully stateless |
| Redis/Memcached | ✅ Excellent | ✅ Excellent | ✅ No affinity needed | Distributed cache |

### Security

| Approach | CSRF Protection | Data Confidentiality | Tampering Prevention | Revocation |
|----------|-----------------|---------------------|---------------------|------------|
| Encrypted Cookies | ✅ Yes | ✅ Encrypted | ✅ HMAC signature | ❌ No (until expiry) |
| Database | ✅ Yes | ✅ Server-side only | ✅ Server-controlled | ✅ Yes (DELETE) |
| In-Memory | ✅ Yes | ✅ Server-side only | ✅ Server-controlled | ✅ Yes (delete) |
| Stateless Tokens | ✅ Yes | ⚠️ Visible (unless JWE) | ✅ HMAC signature | ❌ No (until expiry) |
| Redis/Memcached | ✅ Yes | ✅ Server-side only | ✅ Server-controlled | ✅ Yes (DEL) |

### Performance

| Approach | Latency | Throughput | Resource Usage |
|----------|---------|------------|----------------|
| Encrypted Cookies | <0.1ms (crypto) | Very High | CPU (encryption) |
| Database | 5-20ms | Medium | Database connections |
| In-Memory | <0.1ms | Very High | Application memory |
| Stateless Tokens | <0.1ms (crypto) | Very High | CPU (signing) |
| Redis/Memcached | <1ms | Very High | Cache memory + network |

### Infrastructure Requirements

| Approach | Dependencies | Deployment Complexity | Cloud Cost | Operational Burden |
|----------|--------------|---------------------|------------|-------------------|
| Encrypted Cookies | None | Simple | $0 | Minimal |
| Database | Existing DB | Simple | $0 (reuse) | Medium (cleanup) |
| In-Memory | None | Simple | $0 | Minimal |
| Stateless Tokens | None | Simple | $0 | Minimal |
| Redis/Memcached | Cache server | Complex | $15-50/mo | Medium |

---

## Industry Best Practices

### OAuth 2.0 Specifications

**RFC 6749 (OAuth 2.0):**
- State parameter is RECOMMENDED for CSRF protection
- Must be "unguessable" - cryptographically random
- No specific storage mechanism prescribed

**RFC 7636 (PKCE):**
- code_verifier must be stored client-side (SPA) or server-side (backend)
- Must be retrieved during token exchange
- 43-128 character random string
- No specific storage mechanism prescribed

**OAuth 2.1 (Draft):**
- PKCE is REQUIRED for all authorization code flows
- State parameter still RECOMMENDED
- Emphasizes short-lived authorization codes (10 minutes max)

### Common Patterns

1. **Encrypted Cookie Storage** (Modern Approach)
   - OAuth2 Proxy uses this pattern
   - Auth0 and other providers recommend it
   - Backend-for-Frontend (BFF) pattern

2. **JWT-like Signed State** (Cloud-Native)
   - Used in microservices architectures
   - Kubernetes/cloud-native deployments
   - Token Handler Pattern

3. **Hybrid Approach**
   - Cookie stores state + code_verifier
   - Redis stores additional context if needed
   - Balances stateless with flexibility

---

## Recommendations

### For This Project

Based on ADR-0004 (Stateless JWT approach) and ADR-0007 (User Registration), the recommended approach is **Encrypted Cookies**.

**Why Encrypted Cookies:**
1. **Aligns with Architecture**: Extends stateless JWT pattern to OAuth state
2. **Zero Infrastructure**: No Redis to deploy, monitor, or maintain
3. **Works for All Options**: Compatible with both SaaS and self-hosted vendor UIs
4. **Horizontal Scaling**: Works seamlessly across load balancers
5. **Simple Migration**: Can switch to Redis later with minimal code changes
6. **Fast**: Sub-millisecond performance without network calls

### Implementation Priority

**Phase 1: Start with Encrypted Cookies** (0-100K users)
- Implement encrypted cookie storage
- No infrastructure dependencies
- Handles millions of OAuth flows/day
- Simple, fast, stateless

**Phase 2: Evaluate Redis** (if needed at 100K+ users)
- Add Redis only if you need:
  - Immediate revocation capability
  - Detailed audit logging
  - Complex state data (>4KB)
  - Already using Redis for other features

**Phase 3: Hybrid Approach** (1M+ users, if needed)
- Encrypted cookies for normal flows (99% of traffic)
- Redis for special cases (security events, admin flows)

### When to Use Alternatives

**Use Database with TTL when:**
- You want to avoid encryption code complexity
- You need detailed audit trails
- Traffic is low (<1000 OAuth flows/hour)
- Development/staging environments

**Use Stateless Tokens when:**
- You're building microservices
- State data is minimal (<200 bytes)
- You need maximum scalability
- You're comfortable with visible state in URLs

**Use In-Memory when:**
- Development/testing only
- Proof-of-concept implementations
- Single-server deployments (rare)

**Use Redis/Memcached when:**
- Already deployed for other features
- Need immediate revocation
- High-traffic production (>10K flows/hour)
- Enterprise requirements mandate it

---

## Related Documentation

- [ADR-0002: SSO Authentication Strategy](../../adrs/0002-sso-authentication-strategy.md)
- [ADR-0004: Session Management](../../adrs/0004-session-management.md)
- [ADR-0007: User Registration](../../adrs/0007-user-registration.md)
- [API: GET /v1/auth/{provider}](../../apis/v1-auth-provider-initiate.md)
- [API: GET /v1/auth/{provider}/callback](../../apis/v1-auth-provider-callback.md)

---

## References

- [RFC 6749 - The OAuth 2.0 Authorization Framework](https://datatracker.ietf.org/doc/html/rfc6749)
- [RFC 7636 - Proof Key for Code Exchange (PKCE)](https://datatracker.ietf.org/doc/html/rfc7636)
- [OAuth 2.1 Authorization Framework (Draft)](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-08)
- [OAuth 2.0 Security Best Current Practice](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-security-topics)
