---
title: "Open Policy Agent (OPA) Research"
type: docs
weight: 20
---

This document provides comprehensive research on Open Policy Agent (OPA), focusing on its integration with OAuth2/OIDC systems and deployment strategies for authorization in microservices architectures.

## Overview

Open Policy Agent (OPA) is an open-source, general-purpose policy engine that decouples authorization logic from application code. OPA operates as a **Policy Decision Point (PDP)** that evaluates policies against input data to make authorization decisions in real-time.

### Key Characteristics

- **Language-agnostic:** Works with any programming language or framework
- **Domain-agnostic:** Applicable to authorization, admission control, data filtering, etc.
- **Policy as Code:** Uses declarative Rego language for policy definition
- **Fast:** In-memory evaluation with sub-millisecond decision times
- **Flexible Deployment:** Can run as sidecar, service, library, or daemon

---

## 1. Integration with OAuth2 and OpenID Connect

OPA complements OAuth2/OIDC by adding fine-grained authorization on top of authentication. While OAuth2/OIDC answers "Who are you?" and "Are you authenticated?", OPA answers "What are you allowed to do?"

### 1.1 Architecture Pattern

```
┌─────────┐                 ┌──────────────┐
│  User   │────────────────>│ OAuth2/OIDC  │
└─────────┘  Authenticate   │   Provider   │
                             └──────┬───────┘
                                    │ JWT Token
                                    v
┌─────────┐   Request + JWT  ┌─────────────┐
│ Client  │─────────────────>│     API     │
└─────────┘                  │  Gateway    │ (PEP)
                             └──────┬──────┘
                                    │ Validate JWT +
                                    │ Check Authorization
                                    v
                             ┌─────────────┐
                             │     OPA     │ (PDP)
                             │  - Verify   │
                             │  - Decode   │
                             │  - Evaluate │
                             └─────────────┘
```

**Flow:**
1. User authenticates with OAuth2/OIDC provider → receives JWT access token
2. Client includes JWT in Authorization header
3. API Gateway/Application (PEP) extracts JWT and context
4. PEP calls OPA with JWT and request context
5. OPA validates JWT signature using JWKS
6. OPA verifies claims (issuer, audience, expiration)
7. OPA evaluates policy against token claims
8. OPA returns allow/deny decision
9. PEP enforces the decision

### 1.2 JWT Token Verification

OPA provides built-in functions for JWT handling:

#### Key Functions

**`io.jwt.decode_verify(token, constraints)`**
- Verifies signature and decodes JWT in one operation
- Returns `[valid, header, payload]`
- Recommended for most use cases

**`io.jwt.verify_rs256(token, certificate)`**
- Verifies RS256 signatures
- Requires explicit certificate/JWKS

**`io.jwt.decode(token)`**
- Decodes without verification
- **Warning:** Never use alone - must verify signature separately

#### Example Policy: JWT Verification with JWKS

```rego
package authz

import future.keywords.if
import future.keywords.in

# JWKS endpoint from OIDC provider
jwks_endpoint := "https://auth.example.com/.well-known/jwks.json"

# Fetch JWKS (cached by OPA)
jwks := http.send({
    "method": "GET",
    "url": jwks_endpoint,
    "cache": true,
    "force_cache_duration_seconds": 86400  # 24 hours
}).body

# Define JWT verification constraints
constraints := {
    "cert": jwks,
    "alg": "RS256",
    "iss": "https://auth.example.com",
    "aud": "journeys-api",
    "time": time.now_ns()
}

# Verify and decode the token
token := input.token
[valid, header, payload] := io.jwt.decode_verify(token, constraints)

# Authorization rule
default allow := false

allow if {
    valid
    payload.scope[_] == "journeys:write"
    payload.sub == input.user_id
}

# Extract user info for logging/context
user_id := payload.sub if valid
user_email := payload.email if valid
user_roles := payload.realm_access.roles if valid
```

### 1.3 OIDC Discovery Support

OPA can dynamically query OIDC discovery endpoints to avoid hardcoding provider metadata.

#### Example: Dynamic Discovery

```rego
package authz

import future.keywords.if

# OIDC provider base URL
oidc_provider := "https://auth.example.com"

# Fetch OIDC configuration
oidc_config := http.send({
    "method": "GET",
    "url": sprintf("%s/.well-known/openid-configuration", [oidc_provider]),
    "cache": true,
    "force_cache_duration_seconds": 3600  # 1 hour
}).body

# Extract JWKS URI from discovery
jwks_uri := oidc_config.jwks_uri

# Fetch JWKS
jwks := http.send({
    "method": "GET",
    "url": jwks_uri,
    "cache": true,
    "force_cache_duration_seconds": 86400  # 24 hours
}).body

# Now use jwks for verification...
```

### 1.4 Key Rotation Support

OPA handles OIDC key rotation through the `kid` (Key ID) header:

```rego
# Decode token to get key ID
[_, header, _] := io.jwt.decode(input.token)
key_id := header.kid

# OPA's HTTP caching uses kid as part of cache key
# When kid changes, new JWKS is fetched automatically
```

### 1.5 OAuth2 Client Credentials Flow

OPA can act as an OAuth2 client to obtain access tokens:

```rego
# Request access token using client credentials
token_response := http.send({
    "method": "POST",
    "url": "https://auth.example.com/token",
    "headers": {
        "Authorization": sprintf("Basic %s", [base64.encode(sprintf("%s:%s", [client_id, client_secret]))]),
        "Content-Type": "application/x-www-form-urlencoded"
    },
    "body": "grant_type=client_credentials&scope=api:read"
}).body

access_token := token_response.access_token
```

### 1.6 Use Cases with OAuth2/OIDC

- **Fine-grained Authorization:** Token provides identity; OPA adds attribute-based access control
- **Multi-tenancy:** Verify user belongs to correct tenant/organization
- **Role-Based Access Control (RBAC):** Check roles in token claims
- **Scope Validation:** Ensure token has required OAuth2 scopes
- **API Gateway Integration:** Validate tokens before routing to backend
- **Microservices Authorization:** Consistent policy across services

---

## 2. Deployment Strategies

OPA offers multiple deployment patterns, each with distinct tradeoffs.

### 2.1 Deployment Pattern Comparison

| Pattern | Latency | Resources | Fault Tolerance | Complexity | Best For |
|---------|---------|-----------|-----------------|------------|----------|
| **Sidecar** | Lowest (localhost) | High (per service) | Excellent | Medium | Latency-sensitive, microservices |
| **Centralized PDP** | Higher (network call) | Low (shared) | Depends on HA setup | Low | Large datasets, tolerant latency |
| **Distributed PDP** | Mixed | Medium | Very Good | High | Complex multi-tier apps |
| **DaemonSet** | Low (localhost) | Medium | Good | Medium | Not recommended |
| **Embedded Library** | Lowest (in-process) | Low | Excellent | Low | Go applications only |

### 2.2 Sidecar Deployment (Recommended)

OPA runs as a sidecar container alongside each application container in the same pod.

#### Characteristics

**Pros:**
- **Ultra-low latency:** Authorization calls are localhost (no network hops)
- **Network fault tolerant:** Each service has its own OPA instance
- **Auto-scaling:** OPA scales with the application
- **Per-application configuration:** Different policies per service
- **High availability:** Service continues even if others are partitioned

**Cons:**
- **Resource intensive:** OPA instance per service replica
- **Not ideal for large datasets:** Data replicated across instances
- **More complex updates:** Must coordinate policy updates across instances

#### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: journeys-api
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: journeys-api
    spec:
      containers:
      # Application container
      - name: api
        image: journeys-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: OPA_URL
          value: "http://localhost:8181/v1/data/authz/allow"

      # OPA sidecar container
      - name: opa
        image: openpolicyagent/opa:latest
        ports:
        - containerPort: 8181
        args:
        - "run"
        - "--server"
        - "--addr=0.0.0.0:8181"
        - "--bundle=/policies/bundle.tar.gz"
        volumeMounts:
        - name: policy-bundle
          mountPath: /policies
        livenessProbe:
          httpGet:
            path: /health
            port: 8181
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health?bundle=true
            port: 8181
          initialDelaySeconds: 5
          periodSeconds: 10

      volumes:
      - name: policy-bundle
        configMap:
          name: opa-policy
```

#### Application Integration (Go Example)

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type OPAInput struct {
    Token  string `json:"token"`
    Method string `json:"method"`
    Path   string `json:"path"`
    UserID string `json:"user_id"`
}

type OPARequest struct {
    Input OPAInput `json:"input"`
}

type OPAResponse struct {
    Result bool `json:"result"`
}

func checkAuthorization(token, method, path, userID string) (bool, error) {
    opaURL := "http://localhost:8181/v1/data/authz/allow"

    input := OPARequest{
        Input: OPAInput{
            Token:  token,
            Method: method,
            Path:   path,
            UserID: userID,
        },
    }

    payload, _ := json.Marshal(input)
    resp, err := http.Post(opaURL, "application/json", bytes.NewBuffer(payload))
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()

    var opaResp OPAResponse
    json.NewDecoder(resp.Body).Decode(&opaResp)

    return opaResp.Result, nil
}
```

### 2.3 Centralized PDP (Cluster Service)

Single OPA service shared by multiple applications.

#### Characteristics

**Pros:**
- **Lower resource consumption:** Single shared instance (with HA replicas)
- **Centralized policy management:** One place to update policies
- **Handles large datasets:** Efficient memory usage for large data
- **Simple deployment:** Standard Kubernetes service

**Cons:**
- **Higher latency:** Network calls required for each decision
- **Potential bottleneck:** Can become performance bottleneck
- **Network dependency:** Authorization fails if network partitioned
- **Single point of failure:** Requires proper HA setup

#### Kubernetes Example

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opa-service
spec:
  replicas: 3  # High availability
  template:
    metadata:
      labels:
        app: opa
    spec:
      containers:
      - name: opa
        image: openpolicyagent/opa:latest
        ports:
        - containerPort: 8181
        args:
        - "run"
        - "--server"
        - "--addr=0.0.0.0:8181"
        - "--bundle=/policies/bundle.tar.gz"
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: opa
spec:
  selector:
    app: opa
  ports:
  - port: 8181
    targetPort: 8181
  type: ClusterIP
```

### 2.4 Distributed PDP (Hybrid)

Combines sidecar and centralized patterns - PDPs embedded where compute exists, centralized for others.

#### Characteristics

**Pros:**
- **Balanced latency:** Low latency for embedded PDPs
- **Flexible:** Adapts to different service requirements
- **Reduces API calls:** Only where beneficial

**Cons:**
- **Complex deployment:** Multiple deployment patterns
- **Must ensure consistency:** Same policy logic everywhere
- **More operational overhead:** Managing multiple PDP types

**Use Case:** Multi-tier SaaS with containerized services (sidecar) and serverless functions (centralized).

### 2.5 DaemonSet Pattern

OPA runs as daemon on each Kubernetes node, shared by pods on that node.

**Note:** Generally **not recommended** according to official OPA documentation. Resource savings minimal compared to sidecar, but loses per-service isolation.

### 2.6 Embedded Library Pattern

OPA compiled into application as Go library.

#### Characteristics

**Pros:**
- **Minimal latency:** In-process (no IPC or network)
- **No separate deployment:** Single binary
- **Simplest operations:** One thing to deploy

**Cons:**
- **Tight coupling:** Policy updates require app redeployment
- **Go only:** Native support limited to Go applications
- **Harder to manage:** Policy management not separated

#### Go Example

```go
import (
    "context"
    "github.com/open-policy-agent/opa/rego"
)

func evaluatePolicy(token, method, path string) (bool, error) {
    ctx := context.Background()

    query := rego.New(
        rego.Query("data.authz.allow"),
        rego.Load([]string{"policies/"}, nil),
    )

    rs, err := query.Eval(ctx, rego.EvalInput(map[string]interface{}{
        "token":  token,
        "method": method,
        "path":   path,
    }))

    if err != nil {
        return false, err
    }

    return rs.Allowed(), nil
}
```

### 2.7 API Gateway Integration

OPA integrates with popular API gateways as external authorizer.

#### Kong Gateway

```yaml
plugins:
- name: opa
  config:
    opa_url: "http://opa:8181/v1/data/authz/allow"
    include_body_in_opa_input: false
```

#### Envoy Proxy (External Authorization)

```yaml
http_filters:
- name: envoy.ext_authz
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz
    grpc_service:
      envoy_grpc:
        cluster_name: opa
      timeout: 0.5s
```

#### AWS API Gateway with Lambda Authorizer

```python
import json
import requests

def lambda_handler(event, context):
    token = event['authorizationToken']

    # Call OPA
    opa_response = requests.post(
        'http://opa:8181/v1/data/authz/allow',
        json={'input': {
            'token': token,
            'methodArn': event['methodArn']
        }}
    )

    result = opa_response.json()['result']

    if result:
        return generate_policy('user', 'Allow', event['methodArn'])
    else:
        return generate_policy('user', 'Deny', event['methodArn'])
```

---

## 3. Policy Management and Distribution

### 3.1 Bundle Management

**Bundles** package policies and data together for distribution to OPA instances.

#### Bundle Structure

```
bundle.tar.gz
├── .manifest
├── policies/
│   ├── authz.rego
│   ├── rbac.rego
│   └── abac.rego
└── data/
    ├── users.json
    └── roles.json
```

#### OPA Configuration for Bundles

```yaml
services:
  - name: bundle-service
    url: https://bundle-server.example.com
    credentials:
      bearer:
        token: "secret-token"

bundles:
  authz:
    service: bundle-service
    resource: bundles/authz.tar.gz
    polling:
      min_delay_seconds: 60
      max_delay_seconds: 120
```

### 3.2 Policy Update Strategies

#### Short Polling (Default)

- OPA sends periodic requests to bundle server
- **ETag-based caching:** Only downloads if bundle changed
- **Configurable intervals:** Balance freshness vs. load

```yaml
bundles:
  authz:
    polling:
      min_delay_seconds: 60
      max_delay_seconds: 120
```

#### Long Polling

- Server holds request until update available or timeout
- Reduces server load and network traffic
- Faster update propagation

```yaml
bundles:
  authz:
    polling:
      long_polling_timeout_seconds: 300
```

#### Push-Based with OPAL

**OPAL** (Open Policy Administration Layer) provides active push:

- Detects policy changes in Git/storage
- Pushes updates to all OPA instances
- Real-time policy updates
- Pub/sub architecture

**Architecture:**
```
Git Repo → OPAL Server → OPAL Client (sidecar to OPA) → OPA
```

### 3.3 Discovery and Centralized Management

#### Discovery Feature

OPA can fetch its configuration dynamically:

```yaml
services:
  - name: discovery
    url: https://config-server.example.com

discovery:
  name: discovery
  resource: /configurations/opa-config.json
```

**Discovery bundle** generates OPA runtime configuration:
- Which bundles to load
- Where to fetch them
- Update intervals
- Decision logging endpoints

#### Git-Based Management

Tools like **OPA Control Plane (OCP)** or **Styra DAS**:

- Store policies in Git repositories
- Build bundles on commit
- Environment promotion (dev → staging → prod)
- Policy testing and validation
- Audit trails and compliance

### 3.4 Bundle Signing and Security

Digital signatures ensure bundle integrity:

```bash
# Generate key pair
openssl genrsa -out private_key.pem 2048
openssl rsa -in private_key.pem -pubout -out public_key.pem

# Sign bundle
opa sign bundle.tar.gz --signing-key private_key.pem

# Configure OPA to verify
services:
  - name: bundle-service
    url: https://bundles.example.com

bundles:
  authz:
    service: bundle-service
    resource: bundles/authz.tar.gz
    signing:
      keyid: my-key
      scope: read
```

---

## 4. Docker Deployment

### 4.1 Basic Docker Deployment

```bash
# Run OPA in server mode
docker run -d \
  --name opa \
  -p 8181:8181 \
  openpolicyagent/opa:latest \
  run --server --addr=0.0.0.0:8181

# Load policy
curl -X PUT http://localhost:8181/v1/policies/authz \
  --data-binary @policy.rego

# Query decision
curl -X POST http://localhost:8181/v1/data/authz/allow \
  -H 'Content-Type: application/json' \
  -d '{"input": {"token": "...", "path": "/v1/journey"}}'
```

### 4.2 Docker Compose with Application

```yaml
version: '3.8'

services:
  journeys-api:
    build: ./api
    ports:
      - "8080:8080"
    environment:
      - OPA_URL=http://opa:8181/v1/data/authz/allow
    depends_on:
      - opa

  opa:
    image: openpolicyagent/opa:latest
    ports:
      - "8181:8181"
    command:
      - "run"
      - "--server"
      - "--addr=0.0.0.0:8181"
      - "--bundle=/policies/bundle.tar.gz"
    volumes:
      - ./policies:/policies
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8181/health"]
      interval: 10s
      timeout: 3s
      retries: 3
```

### 4.3 OPA Configuration File

```yaml
# config.yaml
services:
  - name: bundle-service
    url: https://bundle-server.example.com

bundles:
  authz:
    service: bundle-service
    resource: /bundles/authz.tar.gz

decision_logs:
  service: bundle-service
  resource: /logs

status:
  service: bundle-service

default_decision: /authz/allow
```

```bash
docker run -d \
  --name opa \
  -p 8181:8181 \
  -v $(pwd)/config.yaml:/config/config.yaml \
  openpolicyagent/opa:latest \
  run --server --config-file=/config/config.yaml
```

---

## 5. Performance Considerations

### 5.1 Latency Benchmarks

Typical OPA decision latency (in-memory evaluation):

- **Sidecar/Localhost:** 0.1-1 ms
- **Same cluster:** 2-10 ms
- **Cross-cluster:** 20-100 ms
- **Embedded library:** 0.05-0.5 ms

### 5.2 Optimization Strategies

1. **Use Sidecar for Latency-Critical Paths**
2. **Cache JWT Verification Results** (with proper TTL)
3. **Minimize HTTP Calls in Policies** (use bundles for data)
4. **Partial Evaluation** for pre-computing policy parts
5. **Batch Decisions** when authorizing multiple resources
6. **Resource Limits** to prevent memory issues

### 5.3 Caching Strategies

```rego
# Cache JWKS for 24 hours
jwks := http.send({
    "method": "GET",
    "url": jwks_endpoint,
    "cache": true,
    "force_cache_duration_seconds": 86400
}).body

# Cache OIDC config for 1 hour
oidc_config := http.send({
    "method": "GET",
    "url": discovery_endpoint,
    "cache": true,
    "force_cache_duration_seconds": 3600
}).body
```

---

## 6. Considerations for Journey Tracking System

### 6.1 Recommended Architecture

For the journeys REST API project:

1. **Deployment Pattern:** Sidecar in Kubernetes
   - Low latency for authorization decisions
   - Scales with API replicas
   - Fault-tolerant

2. **OAuth2/OIDC Integration:**
   - Keycloak (or chosen provider) handles authentication
   - Issues JWT access tokens
   - OPA validates JWT and enforces fine-grained authorization

3. **Policy Structure:**
   ```rego
   package journeys.authz

   import future.keywords.if

   # JWT verification
   token := input.token
   [valid, _, payload] := io.jwt.decode_verify(token, constraints)

   # Create journey - requires authenticated user with journeys:write scope
   allow if {
       input.method == "POST"
       input.path == "/v1/journey"
       valid
       "journeys:write" in payload.scope
   }

   # Read journey - user can only read their own journeys
   allow if {
       input.method == "GET"
       startswith(input.path, "/v1/journey/")
       valid
       journey_id := split(input.path, "/")[3]
       data.journeys[journey_id].user_id == payload.sub
   }
   ```

### 6.2 Integration Points

1. **API Middleware:** Check authorization before handler
2. **Endpoint Registration:** Inject OPA client dependency
3. **Error Handling:** Return 403 Forbidden for denied requests
4. **Logging:** Log authorization decisions for audit

### 6.3 Policy Management Workflow

1. **Development:**
   - Write policies in Rego
   - Test with `opa test`
   - Version control in Git

2. **CI/CD:**
   - Lint policies with `opa fmt`
   - Run policy tests
   - Build bundle with `opa build`
   - Push to bundle server or embed in ConfigMap

3. **Deployment:**
   - OPA sidecar loads bundle on startup
   - Polls for updates periodically
   - Hot-reload policies without restart

### 6.4 Example: Go API Integration

```go
// endpoint/create_journey.go
package endpoint

import (
    "context"
    "net/http"
)

type CreateJourneyHandler struct {
    log      *slog.Logger
    opaURL   string
}

func (h *CreateJourneyHandler) Handle(ctx context.Context, req *CreateJourneyRequest) (*CreateJourneyResponse, error) {
    // Extract JWT from Authorization header
    token := extractToken(ctx)

    // Check authorization with OPA
    allowed, err := h.checkOPA(ctx, token, "POST", "/v1/journey", "")
    if err != nil {
        h.log.Error("OPA check failed", "error", err)
        return nil, fmt.Errorf("authorization check failed: %w", err)
    }

    if !allowed {
        return nil, &rest.Error{
            Code:    http.StatusForbidden,
            Message: "insufficient permissions",
        }
    }

    // Proceed with business logic
    // ...
}

func (h *CreateJourneyHandler) checkOPA(ctx context.Context, token, method, path, userID string) (bool, error) {
    input := map[string]interface{}{
        "token":  token,
        "method": method,
        "path":   path,
        "user_id": userID,
    }

    payload, _ := json.Marshal(map[string]interface{}{"input": input})

    req, _ := http.NewRequestWithContext(ctx, "POST", h.opaURL, bytes.NewBuffer(payload))
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()

    var result struct {
        Result bool `json:"result"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return result.Result, nil
}
```

---

## 7. Security Best Practices

1. **Always Verify JWT Signatures:** Never trust unverified tokens
2. **Use HTTPS for Bundle Distribution:** Prevent MITM attacks
3. **Sign Policy Bundles:** Ensure integrity of policies
4. **Limit OPA Network Access:** Only allow necessary connections
5. **Regular Key Rotation:** Support OIDC key rotation
6. **Audit Decision Logs:** Track who accessed what
7. **Principle of Least Privilege:** Default deny policies
8. **Validate All Input:** Never trust input data
9. **Resource Limits:** Prevent DoS via complex policies
10. **Keep OPA Updated:** Apply security patches

---

## 8. Resources

### Official Documentation
- [OPA Documentation](https://www.openpolicyagent.org/docs/latest/)
- [OAuth2 and OIDC Integration](https://www.openpolicyagent.org/docs/latest/oauth-oidc/)
- [Deployment Guide](https://www.openpolicyagent.org/docs/latest/deployments/)
- [Rego Language Reference](https://www.openpolicyagent.org/docs/latest/policy-reference/)

### Tools and Integrations
- [OPA Playground](https://play.openpolicyagent.org/) - Test policies online
- [Styra DAS](https://www.styra.com/) - OPA management platform
- [OPAL](https://github.com/permitio/opal) - Push-based policy updates
- [Conftest](https://www.conftest.dev/) - Policy testing for configs

### Community Resources
- [OPA GitHub](https://github.com/open-policy-agent/opa)
- [Rego Style Guide](https://github.com/StyraInc/rego-style-guide)
- [AWS OPA Guidance](https://docs.aws.amazon.com/prescriptive-guidance/latest/saas-multitenant-api-access-authorization/opa.html)

---

## Conclusion

Open Policy Agent provides a powerful, flexible authorization layer that complements OAuth2/OIDC authentication. For the journeys project:

- **Separation of Concerns:** OAuth2/OIDC handles authentication; OPA handles authorization
- **Fine-Grained Control:** Beyond simple role checks to attribute-based access control
- **Scalable Architecture:** Sidecar deployment provides low latency and fault tolerance
- **Policy as Code:** Version-controlled, testable authorization logic
- **Consistent Enforcement:** Same policy logic across all services

The recommended approach is **sidecar deployment** with **JWT verification** using JWKS from the chosen OIDC provider (e.g., Keycloak), managed through **Git-based bundles** with CI/CD integration.
