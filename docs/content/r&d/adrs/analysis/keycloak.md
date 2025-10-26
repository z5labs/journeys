---
title: "Keycloak Research - OAuth2/OIDC Identity Provider"
description: >
    Comprehensive research on Keycloak as an identity and access management solution for SSO authentication
type: docs
weight: 1
date: 2025-10-26
---

# Keycloak Research: OAuth2/OIDC Identity Provider

This document provides comprehensive research on Keycloak as a potential identity and access management (IAM) solution for the journey tracking REST API. The research focuses on Keycloak's OAuth2/OIDC support, deployment options, and operational requirements.

**Research Date:** 2025-10-26
**Keycloak Version:** 26.4.2
**Source:** [Official Keycloak Documentation](https://www.keycloak.org/)

---

## Overview

### What is Keycloak?

Keycloak is an **open-source identity and access management (IAM) solution** that allows you to add authentication to applications and secure services with minimum effort. It is a Cloud Native Computing Foundation (CNCF) incubation project, reflecting its importance in contemporary cloud infrastructure.

### Key Features

- **Single Sign-On (SSO):** Users authenticate once with Keycloak rather than logging into each application individually, with single sign-out functionality across all connected applications
- **Identity Management:**
  - Admin console for centralized management of applications, users, and authorization policies
  - Account management console where users can update profiles, change passwords, and configure two-factor authentication
  - User federation supporting LDAP and Active Directory connections
  - Social login and identity brokering capabilities
- **Authorization:** Fine-grained authorization services enabling administrators to define specific permission policies across services
- **Technical Strengths:** High performance, clustering for scalability, customizable themes, and extensibility through code
- **Standards Compliance:** Full support for OpenID Connect, OAuth 2.0, and SAML 2.0

---

## 1. OAuth2 and OpenID Connect Support

### Protocol Support

Keycloak provides **full, standards-compliant support** for:
- **OpenID Connect (OIDC)** - Complete implementation
- **OAuth 2.0** - Full support including latest extensions
- **SAML 2.0** - Also supported for enterprise use cases

### Standard OIDC Endpoints

All endpoints follow the pattern: `https://{host}/realms/{realm}/.well-known/openid-configuration`

**Key endpoints include:**

- **Authorization Endpoint** - Initiates authentication flow (redirects user agent to Keycloak)
- **Token Endpoint** - Issues access tokens, refresh tokens, ID tokens (exchanges authorization code or credentials)
- **UserInfo Endpoint** - Returns user claims (bearer token protected)
- **Introspection Endpoint** - Validates token state
- **Token Revocation Endpoint** - Revokes access/refresh tokens
- **Device Authorization Endpoint** - Device authorization flow
- **JWKS Endpoint** - Public keys for token validation
- **Discovery Document** - `.well-known/openid-configuration` for auto-discovery

### Supported OAuth2 Grant Types

- **Authorization Code** (recommended for web apps)
- **Implicit** (deprecated but supported for legacy apps)
- **Refresh Token** (maintains long-lived sessions)
- **Password** (Resource Owner Password Credentials)
- **Client Credentials** (service-to-service authentication)

### Recent Updates (2025)

- **OAuth 2.0 Broker Support**: Keycloak can now federate with any OAuth 2.0 provider (previously limited to OIDC and SAML)
- **Email Verification Support**: OpenID Connect broker now supports the standard `email_verified` claim from ID tokens
- **Enhanced Standards Compliance**: Continued improvements to OIDC Core Specification alignment

---

## 2. Social Identity Provider Brokering

### Identity Brokering Architecture

Keycloak can **act as an identity broker**, delegating authentication to external identity providers (IDPs). This allows applications to integrate with Keycloak once, then easily add/remove social providers through Keycloak's admin console **without changing application code**.

When a user authenticates through a federated provider, Keycloak uses **identity provider mappers** to map incoming tokens and assertions to user and session attributes, facilitating the transfer of identity information from external providers to requesting applications.

### Supported Social Identity Providers

Keycloak includes **built-in support** for the following social identity providers:

- **Google** ✅
- **Facebook** ✅
- **Microsoft** ✅
- **GitHub** ✅
- **GitLab**
- **Bitbucket**
- **LinkedIn**
- **Instagram**
- **Twitter**
- **PayPal**
- **Stack Overflow**
- **OpenShift 4**

### Generic Provider Support

In addition to preconfigured social providers, Keycloak supports:
- **Generic OAuth 2.0 providers** (any OAuth2-compliant provider)
- **Generic OIDC providers** (any OpenID Connect provider)
- **Generic SAML providers** (enterprise SSO)
- **Custom providers** (can be developed and plugged in)

**Note:** Apple (Sign in with Apple) can be configured using the generic OIDC provider support, as Apple provides full OpenID Connect compliance.

---

## 3. Fine-Grained Authorization Services

### What is Fine-Grained Authorization?

Keycloak's **Authorization Services** enable fine-grained authorization policies that go far beyond simple role-based access control (RBAC). Instead of just checking "Does this user have the 'admin' role?", you can enforce complex rules like:

- "Can this user edit this specific document?"
- "Can this user view resources owned by their department during business hours?"
- "Can this user approve purchases under $1000 but require manager approval above that?"

This separates **authentication** (who you are) from **authorization** (what you can do), allowing security policies to be centrally managed and dynamically evaluated without changing application code.

### Core Concepts

#### Resources

**Resources** represent the protected assets of your application that require access control. They can be:
- Web pages or API endpoints (e.g., `/api/v1/journeys/{id}`)
- Files in a file system (e.g., "Project Budget 2025.xlsx")
- Domain objects (e.g., "Journey #12345", "User Profile")
- Abstract concepts (e.g., "Bank Accounts", "Employee Records")

Resources can represent:
- **Groups of similar items** (like a Java class): "All Journeys"
- **Specific instances** (like an object): "Journey #12345 owned by Alice"

Each resource is identified by a **unique URI** and can have **attributes** for additional metadata (e.g., owner, department, classification level).

**Example:**
```
Resource: "journey-api"
  URI: /api/v1/journeys/*
  Type: http://example.com/resource/journey
  Attributes:
    - owner: alice
    - department: engineering
```

#### Scopes

**Scopes** define the boundaries of what actions can be performed on a resource. They represent the "verbs" or operations:
- `view` - Read access
- `edit` - Modify access
- `delete` - Remove access
- `share` - Grant access to others
- `approve` - Approve/authorize actions

Scopes can also represent specific attributes within resources (e.g., `view:salary`, `edit:personal-info`).

A single resource typically has multiple associated scopes. For example, a "Journey" resource might have: `view`, `edit`, `delete`, `share`, `export`.

**Example:**
```
Scopes for "Journey" resource:
  - journey:view
  - journey:edit
  - journey:delete
  - journey:share
  - journey:export
```

#### Policies

**Policies** define the conditions that must be satisfied for access to be granted. They are **generic and reusable** - not tied to specific resources. A policy answers: "Under what conditions is access allowed?"

**Supported policy types:**

| Policy Type | Description | Example |
|------------|-------------|---------|
| **Role-Based** | User must have specific role(s) | User has role "journey-owner" |
| **User-Based** | Access granted to specific user(s) | Only user "alice@example.com" |
| **Group-Based** | User must belong to group(s) | User in group "engineering-team" |
| **Time-Based** | Access only during specific times | Monday-Friday, 9 AM - 5 PM |
| **JavaScript** | Custom logic using JavaScript | `$evaluation.context.attributes.ip.startsWith('10.0.')` |
| **Attribute-Based (ABAC)** | Evaluate user/resource attributes | User.department == Resource.department |
| **Context-Based (CBAC)** | Runtime environment factors | Request IP, user agent, geo-location |
| **Aggregated** | Combine multiple policies | Policy A AND Policy B |
| **Client-Based** | Based on OAuth client ID | Only requests from mobile app |

Policies can be **combined** using boolean logic (AND, OR, NOT) to create sophisticated authorization rules.

**Example - Role Policy:**
```json
{
  "name": "Journey Owner Policy",
  "type": "role",
  "logic": "POSITIVE",
  "roles": [
    { "name": "journey-owner", "required": true }
  ]
}
```

**Example - JavaScript Policy:**
```javascript
var context = $evaluation.getContext();
var resource = context.getResource();
var user = context.getIdentity();

// Only allow access if user owns the resource
if (resource.getAttribute('owner') == user.getId()) {
  $evaluation.grant();
} else {
  $evaluation.deny();
}
```

**Example - Time-Based Policy:**
```json
{
  "name": "Business Hours Only",
  "type": "time",
  "dayMonth": "1-31",
  "month": "1-12",
  "hour": "9-17",
  "logic": "POSITIVE"
}
```

#### Permissions

**Permissions** are the bridge between resources and policies. They define the final authorization rule: **"Who can perform what actions on specific resources based on policy evaluation."**

A permission answers: "What resources/scopes are protected by which policies?"

**Two types of permissions:**

1. **Resource-Based Permissions** - Protect specific resources
   - Example: "Journey #12345 can be edited if 'Journey Owner Policy' is satisfied"

2. **Scope-Based Permissions** - Protect specific actions across resources
   - Example: "The 'delete' action on any Journey requires 'Admin Policy' to be satisfied"

**Example - Resource-Based Permission:**
```json
{
  "name": "Edit Journey Permission",
  "resource": "journey-12345",
  "scopes": ["journey:edit"],
  "policies": ["Journey Owner Policy", "Business Hours Only"],
  "decisionStrategy": "UNANIMOUS"
}
```

This permission requires **both** policies to pass (UNANIMOUS) before allowing edit access.

**Example - Scope-Based Permission:**
```json
{
  "name": "Delete Any Journey Permission",
  "scopes": ["journey:delete"],
  "policies": ["Admin Policy"],
  "decisionStrategy": "AFFIRMATIVE"
}
```

### How They Work Together

The authorization flow combines these concepts:

```
Request: "Can Alice edit Journey #12345?"
    ↓
1. Identify Resource: "Journey #12345"
    ↓
2. Identify Scope: "journey:edit"
    ↓
3. Find Permissions protecting this resource + scope
    ↓
4. Evaluate Policies referenced by permissions:
   - Is Alice in role "journey-owner"? ✓
   - Is current time during business hours? ✓
    ↓
5. Decision: GRANT (all policies satisfied)
```

### Authorization Architecture

Keycloak implements a standards-based authorization architecture with four key components:

#### 1. Policy Administration Point (PAP)

The **management interface** where administrators define:
- Resources to protect
- Scopes (actions)
- Policies (rules)
- Permissions (combining resources/scopes with policies)

**Access via:**
- Admin Console UI (`https://{host}/admin`)
- Protection API (RESTful API)

#### 2. Policy Decision Point (PDP)

The **evaluation engine** that:
- Receives authorization requests
- Evaluates applicable policies
- Returns authorization decisions (GRANT or DENY)
- Issues Requesting Party Tokens (RPT) with permissions

**Keycloak's PDP evaluates:**
- User attributes (roles, groups, claims)
- Resource attributes (owner, type, metadata)
- Context attributes (time, IP, location)
- Custom JavaScript logic

#### 3. Policy Enforcement Point (PEP)

The **enforcement mechanism** in your application that:
- Intercepts requests to protected resources
- Checks if requester has authorization
- Consults the PDP (Keycloak server) for decisions
- Allows or denies access based on decision

**Implementation options:**
- **Keycloak Policy Enforcer** (built-in adapter for Java/Spring/Quarkus)
- **Application code** (manual enforcement using Keycloak APIs)
- **API Gateway** (enforce at edge, e.g., Kong, NGINX)

#### 4. Policy Information Point (PIP)

The **attribute source** that provides:
- User attributes from identity tokens
- Resource metadata from Protection API
- Context information from request headers
- External data sources (databases, APIs)

### Policy Enforcement Mechanisms

#### Enforcement Modes

Applications can configure the policy enforcer in three modes:

| Mode | Behavior | Use Case |
|------|----------|----------|
| **ENFORCING** (default) | Deny all requests unless explicitly permitted | Production security |
| **PERMISSIVE** | Allow all requests unless explicitly denied | Gradual rollout, testing |
| **DISABLED** | No policy evaluation, all access granted | Development, debugging |

Even when **DISABLED**, applications can access permission details through the `AuthorizationContext` object to make dynamic UI decisions (e.g., hiding menu items).

#### Policy Enforcer Integration

The **Keycloak Policy Enforcer** acts as a filter/interceptor in your application:

**Java/Spring Example:**
```java
@Configuration
@EnableWebSecurity
public class SecurityConfig extends KeycloakWebSecurityConfigurerAdapter {

    @Override
    protected void configure(HttpSecurity http) throws Exception {
        super.configure(http);
        http.authorizeRequests()
            .anyRequest()
            .authenticated();
    }
}
```

**Configuration (`keycloak.json`):**
```json
{
  "policy-enforcer": {
    "enforcement-mode": "ENFORCING",
    "paths": [
      {
        "path": "/api/v1/journeys/*",
        "methods": [
          {
            "method": "GET",
            "scopes": ["journey:view"]
          },
          {
            "method": "PUT",
            "scopes": ["journey:edit"]
          },
          {
            "method": "DELETE",
            "scopes": ["journey:delete"]
          }
        ]
      }
    ]
  }
}
```

The enforcer automatically:
1. Intercepts requests to `/api/v1/journeys/*`
2. Extracts access token from request
3. Calls Keycloak PDP to evaluate permissions
4. Caches authorization decisions for performance
5. Returns 403 Forbidden if denied, or allows request if granted

#### UMA Protocol Support

Keycloak implements **User-Managed Access (UMA) 2.0**, an OAuth2 extension for fine-grained authorization:

**UMA Flow:**
```
1. Client requests resource without permission ticket
   → Resource Server returns 401 + permission ticket

2. Client exchanges permission ticket + access token for RPT
   → Authorization Server evaluates policies
   → Returns RPT (Requesting Party Token) with granted permissions

3. Client retries request with RPT
   → Resource Server validates RPT
   → Grants access if RPT contains required permissions
```

**UMA Token Request:**
```bash
POST /realms/journeys/protocol/openid-connect/token
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ietf:params:oauth:grant-type:uma-ticket
  &ticket=016f84e8-f9b9-11e0-bd6f-0021cc6004de
  &claim_token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
  &claim_token_format=urn:ietf:params:oauth:token-type:jwt
```

**Response (RPT):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 300,
  "permissions": [
    {
      "rsid": "d2fe9843-6462-4bfc-baba-b5787bb6e0e7",
      "rsname": "Journey #12345",
      "scopes": ["journey:view", "journey:edit"]
    }
  ]
}
```

### Supported Authorization Strategies

Keycloak supports multiple access control mechanisms that can be **mixed and matched**:

1. **Role-Based Access Control (RBAC)**
   - Traditional role evaluation (admin, user, manager)
   - Composite roles (roles containing other roles)
   - Realm roles vs. client roles

2. **Attribute-Based Access Control (ABAC)**
   - User attributes (department, clearance-level, employee-type)
   - Resource attributes (classification, owner, project)
   - Environmental attributes (time, IP, location)

3. **User-Based Access Control (UBAC)**
   - Restrict access to specific named users
   - Useful for personal resources or temporary grants

4. **Context-Based Access Control (CBAC)**
   - Request context (IP address, user agent)
   - Geo-location restrictions
   - Device fingerprinting

5. **Rule-Based Access Control**
   - JavaScript-based custom logic
   - Complex business rules
   - Integration with external systems

6. **Time-Based Access Control**
   - Temporal restrictions (business hours, weekdays)
   - Expiration dates
   - Scheduled access windows

7. **Group-Based Access Control**
   - Group membership evaluation
   - Hierarchical groups
   - Dynamic group membership

### Use Case Examples

#### Example 1: Resource Ownership

**Scenario:** Users can only edit their own journeys.

**Implementation:**
```javascript
// JavaScript Policy: "Own Resource Policy"
var context = $evaluation.getContext();
var identity = context.getIdentity();
var resource = context.getResource();

if (resource.getAttribute('owner') == identity.getId()) {
  $evaluation.grant();
}
```

**Permission:**
- Resource: "Journey"
- Scopes: `journey:edit`
- Policies: `["Own Resource Policy"]`

#### Example 2: Department-Based Access

**Scenario:** Users can view journeys from their department, but only managers can edit them.

**Policies:**
1. **Same Department Policy** (ABAC):
   ```javascript
   var userDept = $evaluation.getContext().getIdentity().getAttribute('department');
   var resourceDept = $evaluation.getContext().getResource().getAttribute('department');
   if (userDept == resourceDept) {
     $evaluation.grant();
   }
   ```

2. **Manager Role Policy** (RBAC):
   - Requires role: `department-manager`

**Permissions:**
1. **View Journey Permission:**
   - Scopes: `journey:view`
   - Policies: `["Same Department Policy"]`

2. **Edit Journey Permission:**
   - Scopes: `journey:edit`
   - Policies: `["Same Department Policy" AND "Manager Role Policy"]`

#### Example 3: Time and Location Restrictions

**Scenario:** Sensitive financial journeys can only be accessed during business hours from office IP ranges.

**Policies:**
1. **Business Hours Policy** (Time-based):
   ```json
   {
     "type": "time",
     "hour": "9-17",
     "dayOfWeek": "1-5"
   }
   ```

2. **Office Network Policy** (JavaScript + CBAC):
   ```javascript
   var ipAddress = $evaluation.getContext().getAttribute('ip');
   if (ipAddress.startsWith('10.0.') || ipAddress.startsWith('192.168.1.')) {
     $evaluation.grant();
   }
   ```

**Permission:**
- Resource: "Financial Journeys"
- Scopes: `journey:view`
- Policies: `["Business Hours Policy" AND "Office Network Policy"]`
- Decision Strategy: `UNANIMOUS` (all must pass)

### Benefits of Fine-Grained Authorization

1. **Separation of Concerns**
   - Authorization logic centralized in Keycloak
   - Application code focuses on business logic
   - Security policies managed independently

2. **Dynamic Policy Updates**
   - Change authorization rules without code deployment
   - Runtime policy evaluation
   - Immediate effect across all applications

3. **Reduced Code Dependencies**
   - No hard-coded permission checks
   - Consistent authorization across microservices
   - Easier testing and maintenance

4. **Audit and Compliance**
   - Centralized audit logs of authorization decisions
   - Policy history and versioning
   - Compliance reporting

5. **Flexible Security Models**
   - Combine multiple access control strategies
   - Progressive complexity (start simple, add as needed)
   - Support for complex business rules

### Limitations and Considerations

1. **Performance Impact**
   - Policy evaluation adds latency (mitigated by caching)
   - Network calls to Keycloak PDP (use Policy Enforcer caching)
   - Complex JavaScript policies can be slow

2. **Complexity**
   - Learning curve for policy design
   - Debugging authorization issues can be challenging
   - Over-engineering risk for simple use cases

3. **Tight Coupling to Keycloak**
   - Applications depend on Keycloak availability
   - Migration to different IAM solution is complex
   - Offline scenarios require local caching

4. **Not Suitable for All Cases**
   - Simple RBAC might be sufficient for many apps
   - Data-level row filtering better in database layer
   - High-throughput APIs may need simpler authorization

### When to Use Fine-Grained Authorization

**Good fit when you need:**
- ✅ Resource-level permissions (not just role checks)
- ✅ Dynamic authorization rules (changeable without deployment)
- ✅ Attribute-based access control (user/resource attributes)
- ✅ Centralized authorization across multiple services
- ✅ Complex business rules for access control
- ✅ Audit trail of authorization decisions

**Not necessary when:**
- ❌ Simple role-based access is sufficient
- ❌ Authorization logic is purely data-driven (use database views)
- ❌ Real-time performance is critical (< 10ms latency required)
- ❌ Authorization rules never change
- ❌ Single application with no external integrations

---

## 4. Deployment Options

Keycloak offers **six deployment methods**:

1. **OpenJDK** - Bare metal or virtual machine deployment
2. **Docker** - Container deployment for single-node or development
3. **Podman** - Alternative container runtime
4. **Kubernetes** - Cloud-native orchestrated deployment
5. **OpenShift** - Red Hat's Kubernetes distribution
6. **Kubernetes Operator** - Automated lifecycle management on Kubernetes

### Recommended Deployment Path

- **Development/Testing:** Docker with `start-dev` mode
- **Production:** Kubernetes with Operator for automated management
- **Simple Production:** Docker with PostgreSQL and proper TLS configuration

---

## 4. Dependencies

### Core Dependencies

#### 1. Java Runtime (Required)
- **OpenJDK 17+** (bundled in official container images)
- JVM for running Keycloak server

#### 2. Database (Production Requirement)

**Supported databases:**

| Database | Supported Versions | Driver Included | Notes |
|----------|-------------------|-----------------|-------|
| **PostgreSQL** | 13.x - 17.x | ✅ Yes | **Recommended** |
| MySQL | 8.0 LTS, 8.4 LTS | ✅ Yes | |
| MariaDB | 10.6 LTS - 11.8 LTS | ✅ Yes | |
| Oracle Database | 19.3+, 23.x | ❌ No | Requires manual driver installation (ojdbc17 + orai18n) |
| Microsoft SQL Server | 2019, 2022 | ✅ Yes | |
| Amazon Aurora PostgreSQL | 15.x - 17.x | ❌ No | Requires AWS JDBC Driver |
| Azure SQL Database | Current | ✅ Yes | |

**Database Configuration:**
- Minimal setup requires: `db`, `db-username`, `db-password`, `db-url-host`
- Can be set via `conf/keycloak.conf`, CLI arguments, or environment variables
- Recommended: Use `build` command with database type, then `start --optimized`

#### 3. TLS/SSL Certificates (Production Requirement)
- **HTTPS is mandatory** in production mode
- HTTP is disabled by default for security
- Requires valid TLS certificates (can use Let's Encrypt, cert-manager on Kubernetes)

#### 4. Infinispan (Clustering Dependency)
- **Built-in distributed caching system**
- Required for multi-node deployments
- Uses database for node discovery (jdbc-ping mechanism)
- Network ports required:
  - **Port 7800** (configurable) - Unicast communication
  - **Port 57800** - Failure detection
- JGroups transport stack handles inter-node communication
- TLS encryption enabled by default for TCP stacks

### Optional Dependencies

- **LDAP/Active Directory** - For user federation
- **External Infinispan clusters** - For multi-site deployments
- **Reverse proxy/Load balancer** - For production deployments (recommended)

---

## 5. Deployment Examples

### Docker Deployment (Development)

```bash
# Development mode (NOT for production)
docker run -p 8080:8080 \
  -e KC_BOOTSTRAP_ADMIN_USERNAME=admin \
  -e KC_BOOTSTRAP_ADMIN_PASSWORD=admin \
  quay.io/keycloak/keycloak:26.4.2 \
  start-dev
```

**Access:** `http://localhost:8080/admin`

### Docker Deployment (Production-Ready)

```bash
# Build optimized image
docker run --name keycloak-build \
  -e KC_DB=postgres \
  quay.io/keycloak/keycloak:26.4.2 \
  build

# Commit the optimized image
docker commit keycloak-build keycloak:optimized

# Run in production mode
docker run -d \
  -p 8443:8443 \
  -e KC_BOOTSTRAP_ADMIN_USERNAME=admin \
  -e KC_BOOTSTRAP_ADMIN_PASSWORD=changeme \
  -e KC_DB=postgres \
  -e KC_DB_URL_HOST=postgres.example.com \
  -e KC_DB_USERNAME=keycloak \
  -e KC_DB_PASSWORD=secret \
  -e KC_HOSTNAME=auth.example.com \
  -v /path/to/certs:/opt/keycloak/conf/certs \
  keycloak:optimized \
  start --optimized
```

**Production requirements:**
- PostgreSQL container/service
- TLS certificates mounted
- Proper hostname configured
- Secure admin password
- Reverse proxy (optional but recommended)

### Kubernetes Deployment

```bash
# Deploy Keycloak statefulset
kubectl create -f https://raw.githubusercontent.com/keycloak/keycloak-quickstarts/refs/heads/main/kubernetes/keycloak.yaml

# Configure ingress
wget -q -O - https://raw.githubusercontent.com/keycloak/keycloak-quickstarts/refs/heads/main/kubernetes/keycloak-ingress.yaml | \
  sed "s/KEYCLOAK_HOST/keycloak.$(minikube ip).nip.io/" | \
  kubectl create -f -
```

**Production requirements:**
- PostgreSQL StatefulSet or managed database service (AWS RDS, Azure Database, etc.)
- TLS certificates (use cert-manager for automatic management)
- Resource limits configured (CPU/memory based on sizing guide)
- Ingress controller configured (Nginx, Traefik, etc.)
- Session affinity (sticky sessions) enabled on load balancer
- Horizontal Pod Autoscaling (optional)

---

## 6. Production Configuration

### Essential Configuration Requirements

1. **HTTPS/TLS** - Mandatory, HTTP disabled in production mode
2. **Hostname** - Must be configured before startup (`--hostname=auth.example.com`)
3. **Database** - Production-grade database required (PostgreSQL recommended)
4. **Admin Credentials** - Set via environment variables:
   - `KC_BOOTSTRAP_ADMIN_USERNAME`
   - `KC_BOOTSTRAP_ADMIN_PASSWORD`

### Configuration Methods (Priority Order)

Configuration values are applied hierarchically:

1. **Command-line arguments** (highest priority)
2. **Environment variables**
3. **`conf/keycloak.conf` file**
4. **Java KeyStore** (lowest priority)

### Two-Phase Optimization Strategy

Keycloak recommends a build-then-run approach for optimal performance:

```bash
# Phase 1: Build optimized image
bin/kc.sh build --db=postgres --features=preview

# Phase 2: Start with runtime configuration
bin/kc.sh start --optimized \
  --hostname=auth.example.com \
  --db-url-host=postgres.example.com \
  --db-username=keycloak \
  --db-password=secret \
  --https-certificate-file=/path/to/cert.pem \
  --https-certificate-key-file=/path/to/key.pem
```

**Benefits:**
- Faster startup time (build optimizations cached)
- Smaller container image layers
- Better for containerized deployments (build once, run many)

### Example Configuration File

`conf/keycloak.conf`:
```properties
# Database
db=postgres
db-username=keycloak
db-password=${DB_PASSWORD}
db-url-host=postgres.example.com
db-url-database=keycloak

# Hostname
hostname=auth.example.com

# HTTPS
https-certificate-file=/opt/keycloak/conf/certs/cert.pem
https-certificate-key-file=/opt/keycloak/conf/certs/key.pem

# Proxy
proxy-headers=forwarded
proxy-trusted-addresses=10.0.0.0/8

# Caching
cache=ispn
cache-stack=kubernetes
```

---

## 7. Reverse Proxy Requirements

When deploying Keycloak behind a reverse proxy or load balancer:

### Port Configuration

- **Proxy port 8080 or 8443** - Application traffic
- **Do NOT proxy port 9000** - Management endpoints (health checks, metrics)

### Essential Settings

**Required options:**
```bash
--proxy-headers=forwarded    # or 'xforwarded' for X-Forwarded-* headers
--proxy-trusted-addresses=<proxy-ips>  # Security critical!
```

**Warning:** Misconfiguration might leave the server exposed to security vulnerabilities (IP spoofing, open redirects).

### Best Practices

#### 1. Limited Path Exposure

**Expose publicly:**
- `/realms/` - Authentication endpoints
- `/resources/` - Static resources
- `/.well-known/` - Discovery documents

**Block from public access:**
- `/admin/` - Admin console
- `/metrics` - Prometheus metrics
- `/health` - Health check endpoints

#### 2. Sticky Sessions (Session Affinity)

Configure load balancer to use **`AUTH_SESSION_ID` cookie** for session affinity. This:
- Improves performance by routing requests to nodes owning cached session data
- Reduces state transfer overhead between nodes
- Minimizes database queries

#### 3. Context Path Alignment

Keycloak expects to be exposed through the reverse proxy on the **same context path** it's configured for, or use the `--hostname` option with a full URL.

#### 4. Network Isolation

Ensure Keycloak accepts connections **only from the proxy** in your network configuration (firewall rules, security groups).

---

## 8. High Availability and Clustering

### Deployment Architectures

Keycloak supports two primary HA architectures:

#### Single-Cluster Deployment

- Multiple Keycloak pods/nodes in **one Kubernetes cluster**
- Can span **multiple availability zones** with proper network/database configuration
- Shared database across all nodes
- Infinispan distributed caching (`--cache=ispn`)
- Session affinity recommended

**Advantages:**
- Simpler architecture
- Lower cost
- No external dependencies

**Limitations:**
- Single point of failure at Kubernetes control-plane level
- Limited to one region/datacenter (unless multi-AZ)

#### Multi-Cluster Deployment

- Separate Keycloak clusters in **different regions/datacenters**
- External load balancer for traffic distribution
- Separate Infinispan clusters per site
- Database replication between sites

**Advantages:**
- Tolerates availability-zone failure
- Tolerates Kubernetes cluster failure
- Supports regulatory compliance (data residency)

**Challenges:**
- Higher complexity
- Increased cost (additional infrastructure, load balancers)
- Network latency considerations
- Database replication overhead

### Clustering Requirements

For multi-node deployments, Keycloak requires:

1. **Database connectivity** - The default `jdbc-ping` discovery mechanism uses the configured database to track nodes joining the cluster
2. **Network communication:**
   - Port 7800 (configurable) - Unicast data transmission
   - Port 57800 - Failure detection
3. **JGroups transport stack** - Handles reliable inter-node communication with optional TLS encryption (enabled by default for TCP stacks)
4. **Session affinity** - Recommended to minimize state transfer overhead

### Caching Architecture

Keycloak uses **Infinispan** (high-performance, distributable in-memory data grid) for caching:

**Cache modes:**
- `--cache=local` - Development mode, no clustering
- `--cache=ispn` - Production mode, distributed caching

**Cache types:**
1. **Local caches** - Realm data, users, authorization info, keys (per node)
2. **Replicated caches** - `work` cache propagates invalidation messages across cluster
3. **Distributed caches** - Authentication sessions, user sessions, client sessions, offline sessions, login failures, action tokens

---

## 9. Initial Setup Workflow

### Create a Realm

1. Access admin console at `https://{hostname}/admin`
2. Click "Create Realm" next to "Current realm"
3. Enter realm name (e.g., `journeys`)
4. Click "Create"

### Configure Social Identity Providers

For each provider (e.g., Google, Facebook, Apple):

1. Navigate to realm settings → **Identity Providers**
2. Select provider from the list (e.g., "Google")
3. Configure provider settings:
   - **Client ID** (from provider's developer console)
   - **Client Secret** (from provider's developer console)
   - **Scopes** (e.g., `openid email profile`)
4. Copy the **Redirect URI** displayed by Keycloak
5. Register the redirect URI in the provider's developer console
6. Save configuration

**Example for Google:**
- Create OAuth 2.0 Client in Google Cloud Console
- Configure authorized redirect URI: `https://auth.example.com/realms/journeys/broker/google/endpoint`
- Copy Client ID and Secret to Keycloak

### Create Client for Your Application

1. Navigate to **Clients** in admin console
2. Click "Create client"
3. Configure:
   - **Client Type:** OpenID Connect
   - **Client ID:** `journey-api`
   - **Client authentication:** On (for confidential clients)
   - **Authorization:** Off (unless using fine-grained permissions)
   - **Standard flow:** Enabled
   - **Direct access grants:** Disabled (unless needed)
4. Configure **Valid redirect URIs** (e.g., `https://app.example.com/callback`)
5. Configure **Web origins** (e.g., `https://app.example.com`)
6. Save configuration
7. Copy **Client Secret** from Credentials tab

### Create Users (Optional)

If you want to support local users in addition to social login:

1. Navigate to **Users**
2. Click "Create new user"
3. Fill in username, first name, last name
4. Click "Credentials" tab
5. Set password with "Temporary" toggled to "Off"
6. Save

---

## 10. Summary: Keycloak for Journey Tracking API

### Advantages

| Benefit | Description |
|---------|-------------|
| ✅ **Single Integration Point** | Integrate your app with Keycloak once, easily add/remove social providers through admin console without code changes |
| ✅ **All Major Social Providers** | Google, Facebook, Apple (via OIDC), GitHub, Microsoft, LinkedIn, Twitter, and more supported out-of-the-box |
| ✅ **Standards Compliant** | Full OAuth2/OIDC/SAML 2.0 support, ensuring compatibility and future-proofing |
| ✅ **Flexible Deployment** | Docker, Kubernetes, bare metal - choose what fits your infrastructure |
| ✅ **Open Source** | No vendor lock-in, free to use, active community, CNCF project |
| ✅ **Extensible** | Custom themes, providers, authenticators, SPI for customization |
| ✅ **Centralized Management** | Admin console for managing users, realms, clients, policies |
| ✅ **Advanced Features** | Fine-grained authorization, user federation (LDAP/AD), MFA, social login, identity brokering |
| ✅ **High Availability** | Clustering support for production deployments |
| ✅ **User Management** | Built-in user registration, password reset, account management |

### Challenges

| Challenge | Description |
|-----------|-------------|
| ⚠️ **Additional Infrastructure** | Requires PostgreSQL database, TLS certificates, networking configuration |
| ⚠️ **Operational Overhead** | Self-hosted solution requires monitoring, updates, backups, security patching |
| ⚠️ **Complexity** | Learning curve for configuration, deployment, and administration |
| ⚠️ **Resource Requirements** | Memory and CPU overhead (sizing guide needed for production) |
| ⚠️ **Dependency** | External service for authentication (though self-hosted, adds another component to manage) |
| ⚠️ **Initial Setup Time** | More complex than direct provider integration (3-5 days estimated) |
| ⚠️ **Debugging Complexity** | Additional layer between your app and identity providers can complicate troubleshooting |

### Minimum Production Requirements

| Component | Requirement |
|-----------|-------------|
| **Database** | PostgreSQL 13+ (managed service recommended: AWS RDS, Azure Database, GCP Cloud SQL) |
| **TLS Certificates** | Valid SSL/TLS certificates (Let's Encrypt, cert-manager, or commercial CA) |
| **Compute** | Kubernetes cluster or Docker host with 2+ CPU cores, 4GB+ RAM per instance |
| **Memory** | 1-2GB RAM per Keycloak instance (varies by usage, see sizing guide) |
| **Storage** | Database storage (scales with users and sessions) |
| **Networking** | Load balancer, ingress controller (Kubernetes), or reverse proxy |
| **Monitoring** | Health checks, metrics collection (Prometheus), logging |
| **Backup** | Database backup strategy, configuration backup |

### Estimated Costs (Monthly, AWS Example)

- **RDS PostgreSQL (db.t4g.small):** ~$25-30
- **EKS Kubernetes Cluster:** ~$75 (control plane)
- **EC2 Instances (2x t3.medium for Keycloak):** ~$60
- **Load Balancer (ALB):** ~$20
- **Total:** ~$180-200/month minimum

**Note:** Costs can be reduced using managed Keycloak services or optimized further with reserved instances.

---

## 11. Comparison: Direct Provider Integration vs. Keycloak

| Factor | Direct Provider Integration | Keycloak |
|--------|----------------------------|----------|
| **Development Time** | 2-3 days per provider | 3-5 days initial setup, then <1 day per provider |
| **Operational Complexity** | Low (no additional infrastructure) | High (database, certificates, monitoring) |
| **Adding New Providers** | Code changes required | Configuration only (no code changes) |
| **User Management** | Must build yourself | Built-in admin console |
| **Social Providers** | Must integrate each separately | 12+ providers built-in + generic OAuth2/OIDC |
| **Authorization** | Must build yourself | Fine-grained authorization built-in |
| **Cost** | $0 infrastructure (provider APIs free) | ~$180-200/month infrastructure |
| **Vendor Lock-in** | Provider-specific code | Standards-based, portable |
| **Security Updates** | Your responsibility per provider | Keycloak team handles core updates |
| **MFA/2FA** | Must integrate separately | Built-in support |
| **User Federation (LDAP/AD)** | Must build yourself | Built-in support |
| **Customization** | Full control | Limited to Keycloak's extension points |
| **Debugging** | Simpler (fewer layers) | More complex (additional abstraction) |

---

## 12. Recommendation Context for ADR-0003

This research supports **Option 4** in ADR-0003: "Use a unified authentication service like Auth0 or Keycloak (provider abstraction layer)".

### When to Choose Keycloak

Consider Keycloak if:
- ✅ You expect to support **5+ identity providers** eventually
- ✅ You need **centralized user management** across providers
- ✅ You require **fine-grained authorization** beyond basic authentication
- ✅ You want to **avoid vendor lock-in** to commercial IAM services
- ✅ You have **Kubernetes infrastructure** already in place
- ✅ You need **LDAP/Active Directory integration**
- ✅ You have **DevOps resources** to manage the infrastructure
- ✅ Long-term scalability and flexibility are priorities

### When to Choose Direct Integration

Consider direct provider integration if:
- ✅ You only need **2-3 identity providers** (Google, Facebook, Apple)
- ✅ You want **minimal operational overhead**
- ✅ You prioritize **faster time to market** (weeks vs. months)
- ✅ You have **limited DevOps resources**
- ✅ You don't need centralized user management or advanced authorization
- ✅ You're comfortable with **provider-specific code** in your application
- ✅ Infrastructure costs are a concern

---

## 13. References

- **Official Documentation:** https://www.keycloak.org/documentation
- **Getting Started (Docker):** https://www.keycloak.org/getting-started/getting-started-docker
- **Getting Started (Kubernetes):** https://www.keycloak.org/getting-started/getting-started-kube
- **Server Configuration:** https://www.keycloak.org/server/configuration
- **Database Configuration:** https://www.keycloak.org/server/db
- **Reverse Proxy Guide:** https://www.keycloak.org/server/reverseproxy
- **High Availability:** https://www.keycloak.org/high-availability/introduction
- **Caching:** https://www.keycloak.org/server/caching
- **Container Image:** quay.io/keycloak/keycloak:26.4.2
- **GitHub Repository:** https://github.com/keycloak/keycloak
- **CNCF Project Page:** https://www.cncf.io/projects/keycloak/

---

## 14. Next Steps

If choosing Keycloak for the journey tracking API:

1. **Proof of Concept (1-2 weeks)**
   - Deploy Keycloak locally with Docker
   - Configure Google and GitHub providers
   - Integrate with sample Go application using golang.org/x/oauth2
   - Test authentication flows

2. **Production Planning (1 week)**
   - Choose deployment platform (Kubernetes recommended)
   - Design database strategy (managed PostgreSQL)
   - Plan TLS certificate management (cert-manager on K8s)
   - Design monitoring and alerting strategy

3. **Production Deployment (2-3 weeks)**
   - Deploy Keycloak to staging environment
   - Configure all required social providers
   - Set up monitoring (Prometheus, Grafana)
   - Security hardening and penetration testing
   - Load testing

4. **Integration (1-2 weeks)**
   - Update journey tracking API to use Keycloak
   - Implement JWT token validation
   - Update user registration flow
   - End-to-end testing

**Total Estimated Timeline:** 6-8 weeks from decision to production
