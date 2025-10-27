---
title: "OpenFGA Research"
type: docs
weight: 30
---

This document provides comprehensive research on OpenFGA, focusing on its integration with OAuth2/OIDC systems and deployment strategies for fine-grained authorization in microservices architectures.

## Overview

OpenFGA is an open-source, high-performance authorization/permission engine inspired by Google's Zanzibar paper. It implements **Relationship-Based Access Control (ReBAC)** to enable developers to build granular access control using an easy-to-read modeling language and friendly APIs.

### Key Characteristics

- **Zanzibar-inspired:** Based on Google's authorization system that manages trillions of permissions
- **Fine-Grained Authorization (FGA):** Ability to grant specific users permission to perform certain actions on specific resources
- **High Performance:** Sub-millisecond authorization checks at scale
- **Flexible Model:** Supports ReBAC, RBAC, and ABAC patterns
- **Cloud Native:** CNCF Sandbox project with OpenTelemetry support
- **Multi-language SDKs:** Go, JavaScript/Node.js, Python, Java, .NET

### What Makes OpenFGA Different

Unlike traditional policy engines, OpenFGA stores **relationships** between users and objects, allowing complex permission hierarchies and inheritance patterns. Instead of evaluating policies, OpenFGA traverses relationship graphs to determine access.

---

## 1. Integration with OAuth2 and OpenID Connect

OpenFGA complements OAuth2/OIDC by providing fine-grained authorization after authentication. The typical architecture separates concerns:

- **OAuth2/OIDC:** Handles authentication ("Who are you?") and issues JWT tokens
- **OpenFGA:** Handles authorization ("What can you do?") based on relationships

### 1.1 Architecture Pattern

```
┌─────────┐                    ┌──────────────┐
│  User   │───────────────────>│ OAuth2/OIDC  │
└─────────┘   Authenticate     │   Provider   │
                                │  (Keycloak,  │
                                │   Auth0)     │
                                └──────┬───────┘
                                       │ JWT Token
                                       v
┌─────────┐  Request + JWT     ┌──────────────┐
│ Client  │───────────────────>│  API Server  │
└─────────┘                    └──────┬───────┘
                                      │
                          ┌───────────┴──────────┐
                          │                      │
                          v                      v
                   ┌──────────────┐      ┌──────────────┐
                   │   Validate   │      │   OpenFGA    │
                   │   JWT Token  │      │   Check      │
                   │              │      │ Relationship │
                   └──────────────┘      └──────────────┘
                                                │
                                                v
                                         [Allow/Deny]
```

**Flow:**
1. User authenticates with OAuth2/OIDC provider → receives JWT
2. Client sends request with JWT in Authorization header
3. API validates JWT signature and extracts user ID from claims
4. API calls OpenFGA to check if user has required relationship to resource
5. OpenFGA returns allow/deny based on stored relationships
6. API enforces the decision

### 1.2 Integration Examples

#### Express/Node.js with Auth0

```javascript
const { auth } = require('express-oauth2-jwt-bearer');
const { OpenFgaClient } = require('@openfga/sdk');

// Validate JWT
const checkJwt = auth({
  audience: 'https://api.example.com',
  issuerBaseURL: 'https://auth.example.com',
});

// Initialize OpenFGA client
const fgaClient = new OpenFgaClient({
  apiUrl: process.env.FGA_API_URL,
  storeId: process.env.FGA_STORE_ID,
  authorizationModelId: process.env.FGA_MODEL_ID,
});

app.get('/documents/:id', checkJwt, async (req, res) => {
  // JWT is valid, extract user ID
  const userId = req.auth.payload.sub;
  const documentId = req.params.id;

  // Check authorization with OpenFGA
  const { allowed } = await fgaClient.check({
    user: `user:${userId}`,
    relation: 'viewer',
    object: `document:${documentId}`,
  });

  if (!allowed) {
    return res.status(403).json({ error: 'Forbidden' });
  }

  // User is authorized, return document
  const document = await getDocument(documentId);
  res.json(document);
});
```

#### Go with JWT Validation

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"

    "github.com/golang-jwt/jwt/v5"
    openfga "github.com/openfga/go-sdk"
    "github.com/openfga/go-sdk/client"
)

func checkAuthorization(w http.ResponseWriter, r *http.Request) {
    // Extract and validate JWT
    tokenString := extractToken(r)
    claims, err := validateJWT(tokenString)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    userId := claims["sub"].(string)
    documentId := r.URL.Query().Get("document_id")

    // Initialize OpenFGA client
    fgaClient, _ := client.NewSdkClient(&client.ClientConfiguration{
        ApiUrl:  os.Getenv("FGA_API_URL"),
        StoreId: os.Getenv("FGA_STORE_ID"),
    })

    // Check authorization
    body := client.ClientCheckRequest{
        User:     fmt.Sprintf("user:%s", userId),
        Relation: "viewer",
        Object:   fmt.Sprintf("document:%s", documentId),
    }

    data, err := fgaClient.Check(context.Background()).Body(body).Execute()
    if err != nil {
        http.Error(w, "Authorization check failed", http.StatusInternalServerError)
        return
    }

    if !data.GetAllowed() {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // User is authorized
    // ... proceed with request
}
```

#### Spring Boot with Keycloak

```java
@RestController
public class DocumentController {

    @Autowired
    private OpenFgaClient fgaClient;

    @GetMapping("/documents/{id}")
    public ResponseEntity<Document> getDocument(
            @PathVariable String id,
            @AuthenticationPrincipal Jwt jwt) {

        String userId = jwt.getSubject();

        // Check authorization with OpenFGA
        CheckRequest request = new CheckRequest()
            .user("user:" + userId)
            .relation("viewer")
            .object("document:" + id);

        CheckResponse response = fgaClient.check(request).execute();

        if (!response.getAllowed()) {
            return ResponseEntity.status(HttpStatus.FORBIDDEN).build();
        }

        Document doc = documentService.getDocument(id);
        return ResponseEntity.ok(doc);
    }
}
```

### 1.3 Keycloak Integration Pattern

For Keycloak (or other OIDC providers):

1. **Authentication Stage:**
   - Keycloak acts as Identity Provider (IdP)
   - User authenticates and receives access token (JWT)
   - Token contains user claims (sub, email, roles, etc.)

2. **Authorization Stage:**
   - API validates JWT signature using Keycloak's JWKS
   - Extract user identifier from `sub` claim
   - Pass user identifier to OpenFGA for relationship checks
   - OpenFGA evaluates based on stored relationships

3. **Relationship Management:**
   - When users join organizations/teams, write tuples to OpenFGA
   - When documents are created, establish ownership relationships
   - When sharing occurs, create viewer/editor relationships

### 1.4 OpenFGA Server OIDC Authentication

OpenFGA server itself can be configured to require OIDC authentication:

```yaml
# Docker run with OIDC auth
docker run -p 8080:8080 \
  -e OPENFGA_AUTHN_METHOD=oidc \
  -e OPENFGA_AUTHN_OIDC_ISSUER=https://auth.example.com \
  -e OPENFGA_AUTHN_OIDC_AUDIENCE=openfga-api \
  -e OPENFGA_HTTP_TLS_ENABLED=true \
  -e OPENFGA_HTTP_TLS_CERT=/path/to/cert.pem \
  -e OPENFGA_HTTP_TLS_KEY=/path/to/key.pem \
  openfga/openfga run
```

This secures the OpenFGA API itself, requiring valid JWTs to interact with the authorization system.

### 1.5 Key Integration Points

**User Identity Mapping:**
- JWT `sub` claim → OpenFGA user identifier
- Format: `user:{sub}` or `user:{email}`
- Consistent mapping across all services

**Token Scopes vs. Relationships:**
- OAuth2 scopes: Limit what application can do
- OpenFGA relationships: Determine what user can access
- Both work together for complete authorization

**Session Management:**
- JWT provides stateless session
- OpenFGA relationships can be updated independently
- Revoke access by removing tuples, no need to invalidate tokens

---

## 2. Core Concepts and Architecture

### 2.1 Fundamental Building Blocks

#### Stores

A **store** is an isolated container for authorization data. Each store contains:
- Authorization model versions
- Relationship tuples
- Metadata

Stores enable:
- Environment separation (dev, staging, production)
- Multi-tenancy (one store per customer)
- Authorization domain isolation

#### Types

A **type** categorizes objects in your system. Examples:
- `user`
- `organization`
- `workspace`
- `document`
- `folder`
- `journey` (for your journey tracking system)

Types are defined in the authorization model with their possible relations.

#### Relations

**Relations** define how users can relate to objects. Examples:
- `owner`
- `editor`
- `viewer`
- `member`
- `admin`
- `parent`

Relations can be:
- **Direct:** Explicitly assigned via tuple
- **Computed:** Derived from other relations via model logic

#### Objects

An **object** is a specific instance of a type, identified by:
```
{type}:{id}
```

Examples:
- `document:roadmap-2025`
- `folder:engineering`
- `organization:acme-corp`
- `journey:550e8400-e29b-41d4-a716-446655440000`

#### Users

A **user** can be:
- **Individual user:** `user:anne`, `user:bob`
- **Userset:** `organization:acme#member` (all members of organization)
- **Object reference:** `document:spec#owner`
- **Wildcard:** `user:*` (everyone)

### 2.2 Relationship Tuples

A **relationship tuple** is the core data structure:

```json
{
  "user": "user:anne",
  "relation": "owner",
  "object": "document:roadmap"
}
```

This states: "anne is an owner of document:roadmap"

**Tuple Components:**
- `user`: Who has the relationship
- `relation`: What kind of relationship
- `object`: What object the relationship is with

**Conditional Tuples:**

Tuples can include conditions using CEL (Common Expression Language):

```json
{
  "user": "user:anne",
  "relation": "viewer",
  "object": "document:roadmap",
  "condition": {
    "name": "ip_range_condition",
    "context": {
      "ip_address": "192.168.1.100"
    }
  }
}
```

### 2.3 Authorization Model

The **authorization model** defines type definitions and relation logic using a DSL.

#### Basic Model Structure

```
model
  schema 1.1

type user

type document
  relations
    define owner: [user]
    define editor: [user] or owner
    define viewer: [user] or editor
```

This model states:
- Documents have owners, editors, and viewers
- Owners can also edit (via `or owner`)
- Editors can also view (via `or editor`)

#### Relationship Inheritance

```
model
  schema 1.1

type user

type folder
  relations
    define parent: [folder]
    define owner: [user]
    define viewer: [user] or owner or viewer from parent

type document
  relations
    define parent: [folder]
    define owner: [user]
    define viewer: [user] or owner or viewer from parent
```

This enables:
- If you can view a folder, you can view documents in it
- Folder permissions propagate to children

#### Complex Example: Organization Context

```
model
  schema 1.1

type user

type organization
  relations
    define member: [user]
    define admin: [user]

type workspace
  relations
    define organization: [organization]
    define member: [user] or member from organization
    define admin: [user] or admin from organization

type journey
  relations
    define workspace: [workspace]
    define owner: [user]
    define editor: [user] or owner
    define viewer: [user] or editor or member from workspace
```

This model:
- Journeys belong to workspaces
- Workspaces belong to organizations
- Organization members automatically get viewer access to journeys
- Explicit owners and editors can be assigned

### 2.4 Query Operations

#### Check

Verify if a user has a specific relation to an object:

```javascript
const { allowed } = await fgaClient.check({
  user: 'user:anne',
  relation: 'viewer',
  object: 'document:roadmap'
});
// Returns: { allowed: true/false }
```

#### ListObjects

Get all objects of a type that a user has a relation with:

```javascript
const { objects } = await fgaClient.listObjects({
  user: 'user:anne',
  relation: 'viewer',
  type: 'document'
});
// Returns: { objects: ['document:roadmap', 'document:spec', ...] }
```

#### ListUsers

Get all users who have a relation to an object:

```javascript
const { users } = await fgaClient.listUsers({
  object: 'document:roadmap',
  relation: 'viewer',
  user_filters: [{ type: 'user' }]
});
// Returns: { users: ['user:anne', 'user:bob', ...] }
```

#### Contextual Tuples

Tuples that exist only for the duration of a query:

```javascript
const { allowed } = await fgaClient.check({
  user: 'user:anne',
  relation: 'viewer',
  object: 'document:roadmap',
  contextualTuples: [
    {
      user: 'user:anne',
      relation: 'member',
      object: 'organization:acme'
    }
  ]
});
```

Useful for testing "what-if" scenarios without persisting tuples.

---

## 3. Authorization Modeling Language (DSL)

OpenFGA uses a domain-specific language for defining authorization models.

### 3.1 Basic Syntax

```
model
  schema 1.1

type {object_type}
  relations
    define {relation_name}: {definition}
```

### 3.2 Relation Definition Patterns

#### Direct Assignment

```
define owner: [user]
```

Allows explicit assignment of users to the relation.

#### Union (OR)

```
define viewer: [user] or editor or owner
```

A user is a viewer if they are:
- Directly assigned as viewer, OR
- An editor, OR
- An owner

#### Intersection (AND)

```
define can_delete: owner and admin
```

A user can delete only if they are BOTH owner AND admin.

#### Exclusion (BUT NOT)

```
define active_member: member but not banned
```

A user is an active member if they are a member BUT NOT banned.

#### Parent Relationship (FROM)

```
define viewer: [user] or viewer from parent
```

A user is a viewer if they are:
- Directly assigned as viewer, OR
- A viewer of the parent object

### 3.3 Type Restrictions

Specify which types can fill a relation:

```
define parent: [folder, drive]
```

The `parent` can be either a `folder` or a `drive`.

### 3.4 Real-World Example: Google Drive-like System

```
model
  schema 1.1

type user

type organization
  relations
    define member: [user]
    define admin: [user]

type drive
  relations
    define organization: [organization]
    define owner: [user, organization#member]
    define admin: [user, organization#admin] or owner
    define writer: [user, organization#member] or admin
    define commenter: [user, organization#member] or writer
    define viewer: [user, organization#member] or commenter

type folder
  relations
    define parent: [drive, folder]
    define owner: [user, organization#member]
    define admin: [user, organization#member] or owner or admin from parent
    define writer: [user, organization#member] or admin or writer from parent
    define commenter: [user, organization#member] or writer or commenter from parent
    define viewer: [user, organization#member] or commenter or viewer from parent

type document
  relations
    define parent: [folder, drive]
    define owner: [user, organization#member]
    define editor: [user, organization#member] or owner or editor from parent
    define viewer: [user, organization#member] or editor or viewer from parent
```

### 3.5 Naming Conventions

**Recommended:**
- Use underscores to separate words: `can_create_document`
- Remove prepositions: "can create a document" → `can_create_document`
- Use lowercase
- Alphanumeric, underscores, and hyphens only

**Examples:**
- Good: `can_view`, `can_edit`, `can_delete`, `is_member`
- Avoid: `canView`, `CanEdit`, `can-delete-item`

---

## 4. Deployment Strategies

OpenFGA supports multiple deployment patterns for different use cases.

### 4.1 Docker Deployment

#### Basic Docker Run

```bash
# Pull latest image
docker pull openfga/openfga:latest

# Run OpenFGA server
docker run -d \
  --name openfga \
  -p 8080:8080 \
  -p 8081:8081 \
  -p 3000:3000 \
  openfga/openfga run
```

**Ports:**
- `8080`: HTTP API
- `8081`: gRPC API
- `3000`: Playground UI (disable in production)

#### Docker with PostgreSQL

```bash
# Create network
docker network create openfga-network

# Run PostgreSQL
docker run -d \
  --name postgres \
  --network openfga-network \
  -e POSTGRES_USER=openfga \
  -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DB=openfga \
  postgres:14

# Wait for Postgres to be ready
sleep 5

# Run migrations
docker run --rm \
  --network openfga-network \
  openfga/openfga migrate \
  --datastore-engine postgres \
  --datastore-uri "postgres://openfga:secret@postgres:5432/openfga?sslmode=disable"

# Run OpenFGA
docker run -d \
  --name openfga \
  --network openfga-network \
  -p 8080:8080 \
  -p 8081:8081 \
  openfga/openfga run \
  --datastore-engine postgres \
  --datastore-uri "postgres://openfga:secret@postgres:5432/openfga?sslmode=disable"
```

#### Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14
    container_name: openfga-postgres
    environment:
      POSTGRES_USER: openfga
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: openfga
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U openfga"]
      interval: 5s
      timeout: 5s
      retries: 5

  migrate:
    image: openfga/openfga:latest
    container_name: openfga-migrate
    depends_on:
      postgres:
        condition: service_healthy
    command: |
      migrate --datastore-engine postgres --datastore-uri "postgres://openfga:secret@postgres:5432/openfga?sslmode=disable"

  openfga:
    image: openfga/openfga:latest
    container_name: openfga
    depends_on:
      migrate:
        condition: service_completed_successfully
    ports:
      - "8080:8080"
      - "8081:8081"
      - "3000:3000"
    environment:
      - OPENFGA_DATASTORE_ENGINE=postgres
      - OPENFGA_DATASTORE_URI=postgres://openfga:secret@postgres:5432/openfga?sslmode=disable
      - OPENFGA_LOG_FORMAT=json
      - OPENFGA_LOG_LEVEL=info
    command: run

  journeys-api:
    build: ./api
    container_name: journeys-api
    depends_on:
      - openfga
    ports:
      - "8000:8000"
    environment:
      - FGA_API_URL=http://openfga:8080
      - FGA_STORE_ID=${FGA_STORE_ID}
      - FGA_MODEL_ID=${FGA_MODEL_ID}

volumes:
  postgres_data:
```

### 4.2 Kubernetes Deployment

#### Using Helm Chart

```bash
# Add OpenFGA Helm repository
helm repo add openfga https://openfga.github.io/helm-charts
helm repo update

# Install OpenFGA
helm install openfga openfga/openfga \
  --set datastore.engine=postgres \
  --set datastore.uri="postgres://user:pass@postgres:5432/openfga" \
  --set replicaCount=3 \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=3 \
  --set autoscaling.maxReplicas=10
```

#### Kubernetes Manifest Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: openfga-config
data:
  config.yaml: |
    datastore:
      engine: postgres
      uri: postgres://openfga:secret@postgres:5432/openfga
    log:
      format: json
      level: info
    metrics:
      enabled: true
      addr: 0.0.0.0:2112
    playground:
      enabled: false
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: openfga
  labels:
    app: openfga
spec:
  replicas: 3
  selector:
    matchLabels:
      app: openfga
  template:
    metadata:
      labels:
        app: openfga
    spec:
      containers:
      - name: openfga
        image: openfga/openfga:latest
        args:
          - run
        ports:
        - name: http
          containerPort: 8080
        - name: grpc
          containerPort: 8081
        - name: metrics
          containerPort: 2112
        env:
        - name: OPENFGA_DATASTORE_ENGINE
          value: postgres
        - name: OPENFGA_DATASTORE_URI
          valueFrom:
            secretKeyRef:
              name: openfga-db
              key: uri
        - name: OPENFGA_LOG_FORMAT
          value: json
        - name: OPENFGA_METRICS_ENABLED
          value: "true"
        - name: OPENFGA_PLAYGROUND_ENABLED
          value: "false"
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: openfga
spec:
  selector:
    app: openfga
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: grpc
    port: 8081
    targetPort: 8081
  - name: metrics
    port: 2112
    targetPort: 2112
  type: ClusterIP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: openfga-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: openfga
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 4.3 Deployment Patterns

#### Pattern 1: Single Region, High Availability

```
┌─────────────────────────────────────────┐
│         Load Balancer                   │
└────────┬────────┬────────┬──────────────┘
         │        │        │
    ┌────▼───┐ ┌─▼────┐ ┌─▼────┐
    │ OpenFGA│ │OpenFGA│ │OpenFGA│
    │Instance│ │Instance│ │Instance│
    └────┬───┘ └──┬───┘ └──┬────┘
         │        │        │
         └────────┼────────┘
                  │
         ┌────────▼─────────┐
         │   PostgreSQL     │
         │  Primary + Replica│
         └──────────────────┘
```

**Characteristics:**
- 3+ OpenFGA instances for HA
- Load balancer distributes requests
- PostgreSQL with replication
- Same datacenter/region

#### Pattern 2: Multi-Region Active-Active

```
Region 1                       Region 2
┌──────────────┐              ┌──────────────┐
│  OpenFGA     │              │  OpenFGA     │
│  Instances   │              │  Instances   │
└──────┬───────┘              └──────┬───────┘
       │                             │
┌──────▼───────┐              ┌──────▼───────┐
│  DynamoDB    │◄────────────►│  DynamoDB    │
│ Global Table │  Replication │ Global Table │
└──────────────┘              └──────────────┘
```

**Characteristics:**
- DynamoDB Global Tables for active-active
- Cross-region replication
- Read/write from any region
- Eventually consistent

---

## 5. Production Best Practices

### 5.1 Security Configuration

#### Enable TLS

```bash
docker run -d \
  -e OPENFGA_HTTP_TLS_ENABLED=true \
  -e OPENFGA_HTTP_TLS_CERT=/certs/server.crt \
  -e OPENFGA_HTTP_TLS_KEY=/certs/server.key \
  -e OPENFGA_GRPC_TLS_ENABLED=true \
  -e OPENFGA_GRPC_TLS_CERT=/certs/server.crt \
  -e OPENFGA_GRPC_TLS_KEY=/certs/server.key \
  -v /path/to/certs:/certs \
  openfga/openfga run
```

#### Authentication Methods

**Pre-shared Keys:**

```bash
docker run -d \
  -e OPENFGA_AUTHN_METHOD=preshared \
  -e OPENFGA_AUTHN_PRESHARED_KEYS="key1,key2,key3" \
  openfga/openfga run
```

**OIDC:**

```bash
docker run -d \
  -e OPENFGA_AUTHN_METHOD=oidc \
  -e OPENFGA_AUTHN_OIDC_ISSUER=https://auth.example.com \
  -e OPENFGA_AUTHN_OIDC_AUDIENCE=openfga-api \
  openfga/openfga run
```

#### Disable Playground in Production

```bash
docker run -d \
  -e OPENFGA_PLAYGROUND_ENABLED=false \
  openfga/openfga run
```

### 5.2 Performance Configuration

#### Database Connection Pooling

```bash
# For 100 max DB connections with 3 OpenFGA instances
# Each instance should use: 100 / 3 ≈ 33 connections

docker run -d \
  -e OPENFGA_DATASTORE_MAX_OPEN_CONNS=33 \
  -e OPENFGA_DATASTORE_MAX_IDLE_CONNS=20 \
  openfga/openfga run
```

**Guidelines:**
- Divide max DB connections equally across instances
- Set idle connections high to avoid recreation overhead
- Co-locate database in same datacenter for low latency

#### Concurrency Limits

```bash
docker run -d \
  -e OPENFGA_MAX_CONCURRENT_READS_FOR_CHECK=1000 \
  -e OPENFGA_MAX_CONCURRENT_READS_FOR_LIST_OBJECTS=1000 \
  -e OPENFGA_MAX_CONCURRENT_READS_FOR_LIST_USERS=1000 \
  -e OPENFGA_RESOLVE_NODE_LIMIT=25 \
  -e OPENFGA_RESOLVE_NODE_BREADTH_LIMIT=100 \
  openfga/openfga run
```

**Tuning:**
- Lower values if queries impact other endpoints
- `RESOLVE_NODE_LIMIT`: Max query depth
- `RESOLVE_NODE_BREADTH_LIMIT`: Max concurrent evaluations

#### Caching

```bash
docker run -d \
  -e OPENFGA_CHECK_QUERY_CACHE_ENABLED=true \
  -e OPENFGA_CHECK_QUERY_CACHE_TTL=10s \
  openfga/openfga run
```

**Trade-offs:**
- Reduces latency and database load
- Increases response staleness
- Good for read-heavy workloads

#### Result Limits

```bash
docker run -d \
  -e OPENFGA_LIST_OBJECTS_MAX_RESULTS=500 \
  -e OPENFGA_LIST_USERS_MAX_RESULTS=500 \
  openfga/openfga run
```

Default is 1,000 - lower for better performance.

### 5.3 Server Pool Strategy

**Recommendation:** Prefer **small pool of high-capacity servers** over large pool of small servers.

**Why:**
- Better cache hit ratios (in-memory caching)
- More efficient resource usage
- Simpler operations

**Example:**
- Good: 3 servers with 4 CPU, 8GB RAM each
- Avoid: 12 servers with 1 CPU, 2GB RAM each

### 5.4 Logging

```bash
docker run -d \
  -e OPENFGA_LOG_FORMAT=json \
  -e OPENFGA_LOG_LEVEL=info \
  openfga/openfga run
```

**Levels:**
- `none`: Not recommended for production
- `error`: Only errors
- `warn`: Warnings and errors
- `info`: Standard production level
- `debug`: Verbose, use sparingly

**Important:** Always enable logging in production for security incident detection.

---

## 6. High Availability and Scaling

### 6.1 Horizontal Scaling

OpenFGA is primarily **CPU-bound**, making horizontal scaling effective.

**Deployment:**
- Run multiple OpenFGA instances (3+ for HA)
- Use load balancer to distribute traffic
- All instances connect to same database
- Stateless servers enable easy scaling

**Auto-scaling (Kubernetes):**

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: openfga-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: openfga
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### 6.2 Database Replication

#### PostgreSQL Read Replicas

OpenFGA supports primary-replica configuration:

```bash
docker run -d \
  -e OPENFGA_DATASTORE_ENGINE=postgres \
  -e OPENFGA_DATASTORE_URI="postgres://user:pass@primary:5432/openfga" \
  -e OPENFGA_DATASTORE_SECONDARY_URI="postgres://user:pass@replica:5432/openfga" \
  openfga/openfga run
```

**Behavior:**
- Primary: Writes + high-consistency reads
- Replica: Regular read operations
- Reduces load on primary

**Consistency Modes:**

```javascript
// Default read (may use replica)
const { allowed } = await fgaClient.check({
  user: 'user:anne',
  relation: 'viewer',
  object: 'document:1'
});

// High consistency read (uses primary)
const { allowed } = await fgaClient.check({
  user: 'user:anne',
  relation: 'viewer',
  object: 'document:1',
  consistency: 'HIGHER_CONSISTENCY'
});
```

**Use higher consistency when:**
- Just wrote a tuple and immediately checking it
- Replication lag would cause issues
- Strong consistency is required

#### DynamoDB for Multi-Region

For global deployments:

```bash
docker run -d \
  -e OPENFGA_DATASTORE_ENGINE=dynamodb \
  -e OPENFGA_DATASTORE_URI="region=us-east-1,table_name=openfga" \
  openfga/openfga run
```

**DynamoDB Global Tables:**
- Active-active across regions
- Automatic cross-region replication
- Writes in one region visible globally
- Built-in high availability

### 6.3 Load Balancing

**HTTP/REST API:**
- Standard Layer 7 load balancing
- Round-robin or least-connections
- Health check: `GET /healthz`

**gRPC API:**
- Requires gRPC-aware load balancer
- Options: Envoy, NGINX, cloud LB with gRPC support
- Client-side load balancing via gRPC client

**Example NGINX config:**

```nginx
upstream openfga_http {
    least_conn;
    server openfga-1:8080;
    server openfga-2:8080;
    server openfga-3:8080;
}

upstream openfga_grpc {
    server openfga-1:8081;
    server openfga-2:8081;
    server openfga-3:8081;
}

server {
    listen 80;
    location / {
        proxy_pass http://openfga_http;
        proxy_set_header Host $host;
    }
}

server {
    listen 8081 http2;
    location / {
        grpc_pass grpc://openfga_grpc;
    }
}
```

---

## 7. Monitoring and Observability

### 7.1 Metrics

OpenFGA exposes Prometheus-compatible metrics on port `2112`.

**Enable metrics:**

```bash
docker run -d \
  -p 2112:2112 \
  -e OPENFGA_METRICS_ENABLED=true \
  -e OPENFGA_DATASTORE_METRICS_ENABLED=true \
  -e OPENFGA_METRICS_ENABLE_RPC_HISTOGRAMS=true \
  openfga/openfga run
```

**Key Metrics:**
- `openfga_request_duration_seconds`: Request latency per endpoint
- `openfga_datastore_query_count`: Database query count
- `openfga_datastore_query_duration_seconds`: Database query latency
- `openfga_check_cache_hit_total`: Cache hit count
- `openfga_check_cache_miss_total`: Cache miss count

**Prometheus scrape config:**

```yaml
scrape_configs:
  - job_name: 'openfga'
    static_configs:
      - targets: ['openfga:2112']
```

### 7.2 Tracing

OpenFGA supports OpenTelemetry tracing.

**Enable tracing:**

```bash
docker run -d \
  -e OPENFGA_TRACE_ENABLED=true \
  -e OPENFGA_TRACE_OTLP_ENDPOINT=otel-collector:4317 \
  -e OPENFGA_TRACE_SAMPLE_RATIO=0.1 \
  openfga/openfga run
```

**Sampling:**
- `0.1` = 10% of requests traced
- Lower sampling in high-throughput environments
- Balance between visibility and overhead

**Trace data includes:**
- Request path through system
- Database queries
- Model evaluation steps
- Latency breakdown

### 7.3 Logging

**Structured JSON logging:**

```bash
docker run -d \
  -e OPENFGA_LOG_FORMAT=json \
  -e OPENFGA_LOG_LEVEL=info \
  openfga/openfga run
```

**Log fields:**
- `timestamp`: ISO 8601 timestamp
- `level`: Log level
- `msg`: Message
- `user`, `relation`, `object`: Request context
- `latency`: Request duration
- `status`: HTTP status code

**Example log entry:**

```json
{
  "timestamp": "2025-01-15T10:30:45Z",
  "level": "info",
  "msg": "check request completed",
  "user": "user:anne",
  "relation": "viewer",
  "object": "document:1",
  "allowed": true,
  "latency_ms": 12.5,
  "store_id": "01HQXYZ..."
}
```

### 7.4 Health Checks

**Endpoints:**
- `/healthz`: Liveness probe (is server running?)
- `/ready`: Readiness probe (can server handle traffic?)

**Kubernetes probes:**

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 7.5 Observability Stack Example

**Docker Compose with full observability:**

```yaml
services:
  openfga:
    image: openfga/openfga:latest
    environment:
      - OPENFGA_METRICS_ENABLED=true
      - OPENFGA_TRACE_ENABLED=true
      - OPENFGA_TRACE_OTLP_ENDPOINT=otel-collector:4317
      - OPENFGA_LOG_FORMAT=json

  otel-collector:
    image: otel/opentelemetry-collector:latest
    command: ["--config=/etc/otel-config.yaml"]
    volumes:
      - ./otel-config.yaml:/etc/otel-config.yaml

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
```

---

## 8. SDKs and APIs

### 8.1 Available SDKs

OpenFGA provides official SDKs for:

- **Go:** `github.com/openfga/go-sdk`
- **JavaScript/Node.js:** `@openfga/sdk`
- **Python:** `openfga-sdk`
- **Java:** `dev.openfga:openfga-sdk`
- **.NET:** `OpenFga.Sdk`

### 8.2 API Protocols

#### HTTP/REST API

- RESTful interface on port `8080`
- JSON request/response
- Easier integration
- Suitable for most languages

**Example cURL:**

```bash
curl -X POST http://localhost:8080/stores/01HQXYZ.../check \
  -H "Content-Type: application/json" \
  -d '{
    "tuple_key": {
      "user": "user:anne",
      "relation": "viewer",
      "object": "document:roadmap"
    }
  }'
```

#### gRPC API

- High-performance protocol on port `8081`
- Protocol Buffers (protobuf)
- Lower latency
- Better for high-throughput services

**Advantages:**
- Strongly typed
- Bi-directional streaming
- More efficient serialization

### 8.3 SDK Usage Examples

#### Go SDK

```go
import (
    "context"
    "fmt"

    "github.com/openfga/go-sdk/client"
)

func main() {
    fgaClient, err := client.NewSdkClient(&client.ClientConfiguration{
        ApiUrl:               "http://localhost:8080",
        StoreId:              "01HQXYZ...",
        AuthorizationModelId: "01HQABC...",
    })

    // Check authorization
    body := client.ClientCheckRequest{
        User:     "user:anne",
        Relation: "viewer",
        Object:   "document:roadmap",
    }

    data, err := fgaClient.Check(context.Background()).
        Body(body).
        Execute()

    if err != nil {
        panic(err)
    }

    fmt.Printf("Allowed: %v\n", data.GetAllowed())
}
```

#### JavaScript/Node.js SDK

```javascript
const { OpenFgaClient } = require('@openfga/sdk');

const fgaClient = new OpenFgaClient({
  apiUrl: 'http://localhost:8080',
  storeId: '01HQXYZ...',
  authorizationModelId: '01HQABC...',
});

// Check authorization
const { allowed } = await fgaClient.check({
  user: 'user:anne',
  relation: 'viewer',
  object: 'document:roadmap',
});

console.log('Allowed:', allowed);

// Write tuple
await fgaClient.write({
  writes: [
    {
      user: 'user:bob',
      relation: 'editor',
      object: 'document:roadmap',
    },
  ],
});

// List objects
const { objects } = await fgaClient.listObjects({
  user: 'user:anne',
  relation: 'viewer',
  type: 'document',
});

console.log('Anne can view:', objects);
```

#### Python SDK

```python
from openfga_sdk.client import OpenFgaClient

fga_client = OpenFgaClient(
    api_url="http://localhost:8080",
    store_id="01HQXYZ...",
    authorization_model_id="01HQABC..."
)

# Check authorization
response = fga_client.check(
    user="user:anne",
    relation="viewer",
    object="document:roadmap"
)

print(f"Allowed: {response.allowed}")
```

---

## 9. Considerations for Journey Tracking System

### 9.1 Recommended Architecture

For the journeys REST API built with Go and the humus framework:

**1. Authorization Model**

```
model
  schema 1.1

type user

type organization
  relations
    define member: [user]
    define admin: [user]

type workspace
  relations
    define organization: [organization]
    define owner: [user]
    define member: [user] or member from organization
    define admin: [user] or admin from organization

type journey
  relations
    define workspace: [workspace]
    define owner: [user]
    define editor: [user] or owner
    define viewer: [user] or editor or member from workspace
    define can_delete: owner or admin from workspace
```

**2. Deployment Pattern**

- **Docker Compose** for local development
- **Kubernetes with Helm** for production
- **PostgreSQL** as datastore (already using for journeys API)
- **3 replicas** minimum for high availability
- **Same cluster** as journeys API for low latency

**3. Integration Points**

```go
// api/app/app.go
package app

import (
    "github.com/openfga/go-sdk/client"
)

type App struct {
    fgaClient *client.OpenFgaClient
    // ... other dependencies
}

func Init() (*App, error) {
    fgaClient, err := client.NewSdkClient(&client.ClientConfiguration{
        ApiUrl:               os.Getenv("FGA_API_URL"),
        StoreId:              os.Getenv("FGA_STORE_ID"),
        AuthorizationModelId: os.Getenv("FGA_MODEL_ID"),
    })

    return &App{
        fgaClient: fgaClient,
    }, nil
}
```

**4. Endpoint Authorization**

```go
// api/endpoint/create_journey.go
package endpoint

import (
    "context"
    "net/http"

    "github.com/openfga/go-sdk/client"
    "github.com/z5labs/humus/rest"
)

type CreateJourneyHandler struct {
    log       *slog.Logger
    fgaClient *client.OpenFgaClient
}

func (h *CreateJourneyHandler) Handle(ctx context.Context, req *CreateJourneyRequest) (*CreateJourneyResponse, error) {
    // Extract user ID from JWT (validated by OAuth2/OIDC middleware)
    userID := extractUserID(ctx)

    // Check if user can create journeys in workspace
    body := client.ClientCheckRequest{
        User:     fmt.Sprintf("user:%s", userID),
        Relation: "member",
        Object:   fmt.Sprintf("workspace:%s", req.WorkspaceID),
    }

    data, err := h.fgaClient.Check(ctx).Body(body).Execute()
    if err != nil {
        return nil, fmt.Errorf("authorization check failed: %w", err)
    }

    if !data.GetAllowed() {
        return nil, &rest.Error{
            Code:    http.StatusForbidden,
            Message: "insufficient permissions",
        }
    }

    // Create journey
    journey := createJourney(req)

    // Write ownership tuple to OpenFGA
    writeBody := client.ClientWriteRequest{
        Writes: []client.ClientTupleKey{
            {
                User:     fmt.Sprintf("user:%s", userID),
                Relation: "owner",
                Object:   fmt.Sprintf("journey:%s", journey.ID),
            },
            {
                User:     fmt.Sprintf("workspace:%s", req.WorkspaceID),
                Relation: "workspace",
                Object:   fmt.Sprintf("journey:%s", journey.ID),
            },
        },
    }

    _, err = h.fgaClient.Write(ctx).Body(writeBody).Execute()
    if err != nil {
        return nil, fmt.Errorf("failed to write authorization tuples: %w", err)
    }

    return &CreateJourneyResponse{Journey: journey}, nil
}
```

**5. List User's Journeys**

```go
// api/endpoint/list_journeys.go
func (h *ListJourneysHandler) Handle(ctx context.Context, req *ListJourneysRequest) (*ListJourneysResponse, error) {
    userID := extractUserID(ctx)

    // Use ListObjects to get journeys user can view
    body := client.ClientListObjectsRequest{
        User:     fmt.Sprintf("user:%s", userID),
        Relation: "viewer",
        Type:     "journey",
    }

    data, err := h.fgaClient.ListObjects(ctx).Body(body).Execute()
    if err != nil {
        return nil, fmt.Errorf("failed to list authorized journeys: %w", err)
    }

    // Extract journey IDs from objects
    journeyIDs := make([]string, 0, len(data.GetObjects()))
    for _, obj := range data.GetObjects() {
        // obj format: "journey:550e8400-..."
        journeyID := strings.TrimPrefix(obj, "journey:")
        journeyIDs = append(journeyIDs, journeyID)
    }

    // Fetch journeys from database
    journeys := fetchJourneys(journeyIDs)

    return &ListJourneysResponse{Journeys: journeys}, nil
}
```

### 9.2 Workflow Examples

#### User Creates Journey

1. User authenticates with OAuth2/OIDC → receives JWT
2. Frontend calls `POST /v1/journey` with JWT
3. API validates JWT, extracts user ID
4. API checks OpenFGA: "Is user a member of workspace?"
5. If yes, create journey in database
6. Write tuples to OpenFGA:
   - `user:anne` is `owner` of `journey:123`
   - `journey:123` has `workspace` `workspace:456`
7. Return journey to frontend

#### User Shares Journey

```go
// api/endpoint/share_journey.go
func (h *ShareJourneyHandler) Handle(ctx context.Context, req *ShareJourneyRequest) (*ShareJourneyResponse, error) {
    userID := extractUserID(ctx)

    // Check if user can share (must be owner)
    canShare, _ := h.fgaClient.Check(ctx).Body(client.ClientCheckRequest{
        User:     fmt.Sprintf("user:%s", userID),
        Relation: "owner",
        Object:   fmt.Sprintf("journey:%s", req.JourneyID),
    }).Execute()

    if !canShare.GetAllowed() {
        return nil, &rest.Error{Code: http.StatusForbidden}
    }

    // Grant viewer access to other user
    _, err := h.fgaClient.Write(ctx).Body(client.ClientWriteRequest{
        Writes: []client.ClientTupleKey{
            {
                User:     fmt.Sprintf("user:%s", req.ShareWithUserID),
                Relation: req.Permission, // "viewer" or "editor"
                Object:   fmt.Sprintf("journey:%s", req.JourneyID),
            },
        },
    }).Execute()

    return &ShareJourneyResponse{Success: true}, nil
}
```

#### Organization-Wide Access

When user creates organization:

```go
// Write organization membership
fgaClient.Write(ctx).Body(client.ClientWriteRequest{
    Writes: []client.ClientTupleKey{
        {
            User:     "user:anne",
            Relation: "admin",
            Object:   "organization:acme",
        },
        {
            User:     "user:bob",
            Relation: "member",
            Object:   "organization:acme",
        },
    },
}).Execute()
```

When workspace is created in organization:

```go
// Link workspace to organization
fgaClient.Write(ctx).Body(client.ClientWriteRequest{
    Writes: []client.ClientTupleKey{
        {
            User:     "organization:acme",
            Relation: "organization",
            Object:   "workspace:engineering",
        },
    },
}).Execute()
```

**Result:** All organization members automatically get viewer access to journeys in the workspace (via model definition).

### 9.3 Migration Strategy

1. **Phase 1 - Setup:**
   - Deploy OpenFGA alongside existing API
   - Create authorization model
   - No enforcement yet

2. **Phase 2 - Write Tuples:**
   - On journey creation, write ownership tuples
   - Backfill existing journeys
   - Monitor tuple growth

3. **Phase 3 - Dual Mode:**
   - Check both old and new authorization
   - Log discrepancies
   - Verify correctness

4. **Phase 4 - Enforcement:**
   - Switch to OpenFGA-only authorization
   - Remove old authorization code
   - Monitor performance

---

## 10. OpenFGA vs OPA Comparison

For context with the OPA research document:

| Aspect | OpenFGA | OPA |
|--------|---------|-----|
| **Model** | Relationship-Based (ReBAC) | Policy-Based (PBAC) |
| **Core Concept** | Store relationships, traverse graph | Evaluate policies against input |
| **Best For** | User-resource relationships, hierarchies | Complex rules, attribute-based |
| **Language** | DSL for relationships | Rego for policies |
| **Data Storage** | Tuples in database (persistent) | Bundles loaded in-memory |
| **Decision Method** | Graph traversal | Policy evaluation |
| **OAuth2/OIDC Role** | User identity from JWT | JWT validation + policy evaluation |
| **Latency** | Sub-millisecond to milliseconds | Sub-millisecond (in-memory) |
| **Scaling** | Horizontal (stateless) + database | Horizontal (stateless) + bundles |
| **Use Case** | "Can user X access resource Y?" | "Is action allowed given context?" |
| **Complexity** | Simpler for relationships | More flexible, steeper learning curve |

**When to Use OpenFGA:**
- User-to-resource permissions
- Sharing and collaboration features
- Hierarchical organizations/workspaces
- Google Drive-like access control

**When to Use OPA:**
- Complex policy rules (time-based, location-based)
- API gateway authorization
- Kubernetes admission control
- General-purpose policy decisions

**Can You Use Both?**
Yes - use OAuth2/OIDC for authentication, OpenFGA for resource-level authorization, and OPA for policy enforcement at API gateway level.

---

## 11. Resources

### Official Documentation
- [OpenFGA Documentation](https://openfga.dev/docs)
- [Authorization Concepts](https://openfga.dev/docs/authorization-concepts)
- [Modeling Language](https://openfga.dev/docs/modeling/getting-started)
- [API Reference](https://openfga.dev/api/service)

### SDKs and Tools
- [Go SDK](https://github.com/openfga/go-sdk)
- [JavaScript SDK](https://github.com/openfga/js-sdk)
- [Python SDK](https://pypi.org/project/openfga-sdk/)
- [VS Code Extension](https://marketplace.visualstudio.com/items?itemName=openfga.openfga-vscode)
- [OpenFGA Playground](https://play.fga.dev/)

### Deployment
- [Kubernetes Setup](https://openfga.dev/docs/getting-started/setup-openfga/kubernetes)
- [Docker Setup](https://openfga.dev/docs/getting-started/setup-openfga/docker)
- [Helm Charts](https://artifacthub.io/packages/helm/openfga/openfga)
- [Production Best Practices](https://openfga.dev/docs/getting-started/running-in-production)

### Integration Examples
- [Auth0 + OpenFGA Tutorial](https://auth0.com/blog/express-typescript-fga/)
- [Spring Boot Example](https://github.com/jimmyjames/fga-spring-examples)
- [Framework Integration Guide](https://openfga.dev/docs/getting-started/framework)

### Community
- [GitHub Repository](https://github.com/openfga/openfga)
- [Community Discord](https://discord.gg/8naAwJfWN6)
- [CNCF OpenFGA](https://www.cncf.io/projects/openfga/)

---

## Conclusion

OpenFGA provides a powerful, scalable authorization solution based on Google's Zanzibar paper. For the journeys project:

**Key Benefits:**
- **Separation of Concerns:** OAuth2/OIDC handles authentication; OpenFGA handles fine-grained authorization
- **Relationship-Based:** Natural modeling of user-workspace-journey relationships
- **High Performance:** Sub-millisecond checks at scale
- **Flexible Sharing:** Easy to implement collaboration features
- **Cloud Native:** Built for Kubernetes, OpenTelemetry, modern infrastructure

**Recommended Implementation:**
1. Deploy OpenFGA as sidecar or service in Kubernetes
2. Use PostgreSQL datastore (already in use for journeys API)
3. Integrate Go SDK in humus-based endpoints
4. Model organizations → workspaces → journeys hierarchy
5. Write tuples on resource creation/sharing
6. Check authorization before operations

**Integration with OAuth2/OIDC:**
- OAuth2/OIDC provider (Keycloak) handles authentication
- JWT provides user identity
- OpenFGA checks relationship-based permissions
- Complete fine-grained access control system

OpenFGA complements OPA by focusing on relationship-based authorization rather than policy evaluation, making it ideal for user-resource permission systems like the journey tracking application.
