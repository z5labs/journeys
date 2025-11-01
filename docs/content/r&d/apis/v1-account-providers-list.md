---
title: "GET /v1/account/providers"
description: >
    Retrieve a list of OAuth2/OIDC providers linked to the authenticated user's account
type: docs
weight: 1000
status: "draft"
date: 2025-11-01
---

## API Overview

This endpoint returns a list of OAuth2/OIDC identity providers that are currently linked to the authenticated user's account. This allows users to see which external authentication providers (Google, Facebook, Apple) they have connected to their account and can use for login.

This endpoint is essential for account management features where users need visibility into their linked authentication methods, particularly in the context of the account linking functionality described in ADR-0005.

## Endpoint Details

- **Method:** `GET`
- **Path:** `/v1/account/providers`
- **Content-Type:** `application/json`
- **Base URL:** `https://api.journeys.example.com` (update with actual base URL)

## Authentication

**Required:** Yes

This endpoint requires authentication via JWT bearer token. The user must be authenticated before they can query their linked providers.

- Authentication method: OAuth2/OIDC JWT bearer token
- Required scopes: `account:read` or `account:manage`
- The JWT must contain the user's identity to retrieve their linked providers

**Example:**
```
Authorization: Bearer <access_token>
```

## Request

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | Bearer token for authentication |
| `Accept` | No | Should be `application/json` (default) |

### Path Parameters

None

### Query Parameters

None

### Request Body

This endpoint does not accept a request body.

## Response

### Success Response (200 OK)

**Schema:**

```json
{
  "providers": [
    {
      "provider": "string",
      "providerId": "string",
      "linkedAt": "2025-01-01T00:00:00Z",
      "isPrimary": boolean
    }
  ]
}
```

**Field Descriptions:**

| Field | Type | Description |
|-------|------|-------------|
| `providers` | array[object] | List of OAuth2/OIDC providers linked to the account |
| `providers[].provider` | string | Provider name (e.g., "google", "facebook", "apple") |
| `providers[].providerId` | string | User's unique identifier from the provider |
| `providers[].linkedAt` | string (ISO 8601) | Timestamp when this provider was linked to the account |
| `providers[].isPrimary` | boolean | Whether this is the primary/original provider used to create the account |

**Example:**

```json
{
  "providers": [
    {
      "provider": "google",
      "providerId": "user:google:123456789",
      "linkedAt": "2025-01-15T10:30:00Z",
      "isPrimary": true
    },
    {
      "provider": "facebook",
      "providerId": "user:facebook:987654321",
      "linkedAt": "2025-02-20T14:45:00Z",
      "isPrimary": false
    }
  ]
}
```

## Error Responses

### 401 Unauthorized

Missing or invalid authentication credentials.

```json
{
  "error": "unauthorized",
  "message": "Invalid or expired authentication token"
}
```

### 500 Internal Server Error

An unexpected error occurred on the server.

```json
{
  "error": "internal_error",
  "message": "An unexpected error occurred. Please try again later."
}
```

## Business Logic Flow

Use Mermaid syntax to create a flowchart representing the business logic:

```mermaid
graph TD
    A[Receive GET Request] --> B{Validate JWT Token}
    B -->|Invalid/Expired| C[Return 401 Unauthorized]
    B -->|Valid| D[Extract User Identity from JWT]
    D --> E[Query Database for Linked Providers]
    E --> F{Providers Found?}
    F -->|No Providers| G[Return Empty Array]
    F -->|Providers Found| H[Format Provider List]
    H --> I[Return 200 OK with Provider List]
```

The business logic flow follows these steps:

1. **Authentication Validation**: The API validates the JWT bearer token in the Authorization header
2. **User Identity Extraction**: If valid, extract the user's unique identifier from the JWT claims
3. **Database Query**: Query the user account database to retrieve all linked OAuth2/OIDC provider records
4. **Response Formation**: Format the provider data including provider name, provider-specific user ID, linking timestamp, and primary flag
5. **Success Response**: Return the list of providers with 200 OK status

## Implementation Notes

- This endpoint is read-only and does not modify any account data
- The response does not include sensitive information like OAuth tokens or refresh tokens
- Provider IDs are returned in the format `user:{provider}:{id}` following the convention from ADR-0005
- The `isPrimary` flag indicates which provider was used for initial account creation
- Empty array is returned if the user has no linked providers (edge case - normally at least one provider should exist)
- Performance: Consider caching provider lists per user as they don't change frequently
- The endpoint supports the account linking user journey documented in User Journey 0003

## Related Documentation

- [ADR-0002: SSO Authentication Strategy](../../adrs/0002-sso-authentication-strategy.md) - OAuth2/OIDC authentication approach
- [ADR-0003: OAuth2/OIDC Provider Selection](../../adrs/0003-oauth2-oidc-provider-selection.md) - Supported providers (Google, Facebook, Apple)
- [ADR-0004: Session Management](../../adrs/0004-session-management.md) - JWT token handling
- [ADR-0005: Account Linking](../../adrs/0005-account-linking.md) - Strategy for linking multiple providers
- [User Journey 0003: Account Linking](../../user-journeys/0003-account-linking.md) - User flow for linking providers
- Code implementation: `api/endpoint/get_account_providers.go` (to be implemented)

## Examples

### Example 1: User with Multiple Linked Providers

**Request:**

```bash
curl -X GET https://api.journeys.example.com/v1/account/providers \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Accept: application/json"
```

**Response:**

```json
{
  "providers": [
    {
      "provider": "google",
      "providerId": "user:google:123456789",
      "linkedAt": "2025-01-15T10:30:00Z",
      "isPrimary": true
    },
    {
      "provider": "apple",
      "providerId": "user:apple:abc.def.ghi",
      "linkedAt": "2025-03-10T16:20:00Z",
      "isPrimary": false
    }
  ]
}
```

### Example 2: New User with Single Provider

**Request:**

```bash
curl -X GET https://api.journeys.example.com/v1/account/providers \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Accept: application/json"
```

**Response:**

```json
{
  "providers": [
    {
      "provider": "facebook",
      "providerId": "user:facebook:987654321",
      "linkedAt": "2025-04-01T09:15:00Z",
      "isPrimary": true
    }
  ]
}
```

## Changelog

| Date | Version | Changes |
|------|---------|---------|
| 2025-11-01 | 1.0 | Initial API documentation |
