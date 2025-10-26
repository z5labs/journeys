---
title: "[0002] SSO Authentication Strategy"
description: >
    Architectural decision for implementing Single Sign-On (SSO) authentication in the journey tracking REST API
type: docs
weight: 2
status: "accepted"
date: 2025-10-26
deciders: []
consulted: []
informed: []
---

## Context and Problem Statement

The journey tracking API needs to implement user authentication to secure endpoints and associate journeys with specific users. We need to decide on an authentication strategy that balances security, developer experience, and operational complexity. Given that users may already have accounts in existing identity providers, we need to determine whether to implement our own authentication system or integrate with external SSO providers.

## Decision Drivers

* Security requirements for protecting user data and API endpoints
* User experience - minimizing the need for users to manage multiple credentials
* Development complexity and time to market
* Operational overhead of managing authentication infrastructure
* Scalability and ability to support multiple authentication providers in the future
* Compliance requirements (e.g., GDPR, data protection)
* Integration with existing identity providers (Google, GitHub, Microsoft, etc.)

## Considered Options

* [option 1] Implement custom JWT-based authentication with username/password
* [option 2] Use OAuth 2.0 with OpenID Connect (OIDC) for SSO with external providers
* [option 3] Hybrid approach: support both custom authentication and SSO providers

## Decision Outcome

Chosen option: "[option 2]", because it provides the best balance of security, user experience, and operational simplicity by leveraging well-established authentication providers rather than building and maintaining our own authentication infrastructure.

### Consequences

* Good, because users can authenticate using existing accounts (Google, GitHub, etc.)
* Good, because we delegate security concerns to established identity providers
* Good, because reduced operational overhead - no password storage or reset flows to manage
* Good, because better security through provider-managed MFA and security policies
* Good, because easier compliance with data protection regulations
* Bad, because dependency on external services for authentication
* Bad, because requires integration code and handling of provider-specific flows
* Bad, because users must have accounts with supported providers

### Confirmation

Implementation will be confirmed through:
- Integration tests validating OAuth 2.0/OIDC flows with test providers
- Security review of token validation and session management
- Documentation of supported providers and configuration
- Monitoring of authentication success/failure rates in production

## Pros and Cons of the Options

### [option 1] Implement custom JWT-based authentication with username/password

Custom authentication system where we manage user credentials and issue JWT tokens.

* Good, because complete control over authentication flow
* Good, because no external dependencies
* Good, because can work offline/in isolated environments
* Neutral, because JWT is a standard, well-understood token format
* Bad, because responsible for secure password storage (hashing, salting)
* Bad, because must implement password reset, email verification flows
* Bad, because must maintain security infrastructure (breach detection, rate limiting)
* Bad, because users need to create and remember another set of credentials
* Bad, because higher development and maintenance cost

### [option 2] Use OAuth 2.0 with OpenID Connect (OIDC) for SSO with external providers

Integrate with external identity providers using standard OAuth 2.0/OIDC protocols.

* Good, because leverages existing user accounts
* Good, because established providers handle security concerns
* Good, because reduced development time - no custom auth UI needed
* Good, because providers often include MFA, security monitoring
* Good, because easier regulatory compliance
* Good, because standard protocols (OAuth 2.0, OIDC) are well-documented
* Neutral, because still need to manage user sessions and tokens after authentication
* Bad, because dependency on external services
* Bad, because requires internet connectivity
* Bad, because integration complexity for multiple providers
* Bad, because users must have accounts with supported providers

### [option 3] Hybrid approach: support both custom authentication and SSO providers

Implement both custom authentication and SSO provider integration.

* Good, because maximum flexibility for users
* Good, because can work in various environments
* Good, because no forced dependency on external providers
* Neutral, because increased code complexity
* Bad, because highest development and maintenance cost
* Bad, because must maintain security infrastructure for custom auth
* Bad, because two authentication systems to secure and monitor
* Bad, because increased testing surface area

## More Information

Related considerations:
- Initial implementation should focus on 1-2 major providers (e.g., Google and GitHub)
- Provider selection should be configurable to allow different deployments
- Consider using a library like `golang.org/x/oauth2` for standard OAuth 2.0 flows
- Token validation should use provider's public keys (JWKS endpoint)
- Session management strategy needs to be defined (see future ADR on session storage)
- Consider rate limiting on authentication endpoints to prevent abuse

Related requirements:
- User registration user journey (docs/content/r&d/user-journeys/)
- API endpoint security requirements
- Future multi-tenancy considerations
