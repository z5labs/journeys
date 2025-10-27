---
title: "[0003] OAuth2/OIDC Provider Selection"
description: >
    Selection of specific OAuth2/OpenID Connect identity providers for SSO authentication in the journey tracking API
type: docs
weight: 3
status: "accepted"
date: 2025-10-26
deciders: []
consulted: []
informed: []
---

## Context and Problem Statement

Following the decision to implement SSO authentication using OAuth 2.0 with OpenID Connect (see [ADR-0002](0002-sso-authentication-strategy.md)), we need to determine which specific identity providers to support in the initial release. Each provider requires integration effort, ongoing maintenance, and monitoring. We must balance user convenience (supporting providers they already use) with development cost and operational complexity. Which OAuth2/OIDC providers should we support initially, and what criteria should guide future provider additions?

## Decision Drivers

* User demographics and likely provider preferences
* Technical quality and reliability of provider OAuth2/OIDC implementations
* Provider documentation quality and developer experience
* Availability of Go libraries and SDK support
* Provider stability and long-term viability
* API rate limits and quotas for free/basic tiers
* OAuth2/OIDC standards compliance
* Development and testing effort required per provider
* Operational monitoring and maintenance overhead
* Time to market for initial release

## Considered Options

* [option 1] Support only Google (single provider, fastest time to market)
* [option 2] Support Google, Facebook, and Apple (three major providers)
* [option 3] Support Google, GitHub, Microsoft, Facebook, and Apple (five providers for maximum coverage)
* [option 4] Use a unified authentication service like Auth0 or Keycloak (provider abstraction layer)

## Decision Outcome

Chosen option: "[option 2] Support Google, Facebook, and Apple", because this provides excellent user coverage while maintaining reasonable development complexity. These three providers collectively cover the vast majority of potential users (Google for general consumers, Facebook for social users, and Apple for privacy-conscious iOS/macOS users), have OAuth2/OIDC implementations, and provide broad market coverage across different user demographics.

### Consequences

* Good, because covers most users (Google for general consumers, Facebook for social users, Apple for privacy-conscious iOS/macOS users)
* Good, because includes Apple which is required for any future iOS app distribution
* Good, because Facebook provides large user base despite privacy concerns
* Good, because manageable initial development effort (estimated 2-3 days per provider)
* Good, because establishes patterns for adding future providers
* Good, because demonstrates broad provider support (consumer, social, mobile-first)
* Bad, because Apple's implementation requires special handling (Sign in with Apple has unique requirements)
* Bad, because Facebook's API has had stability and privacy concerns historically
* Bad, because excludes developer-focused (GitHub) and enterprise (Microsoft) segments initially
* Bad, because requires maintaining integrations with three separate provider APIs
* Bad, because need to monitor health and updates for three different services
* Neutral, because can add GitHub/Microsoft later based on user demand

### Confirmation

Implementation will be confirmed through:
- Successful OAuth2/OIDC flow completion with each provider in staging environment
- Integration tests covering authentication, token refresh, and error scenarios for all providers
- Load testing to verify provider API rate limits are sufficient for expected usage
- Documentation verifying provider configuration and setup procedures
- Monitoring dashboards tracking authentication success rates per provider
- User acceptance testing with real accounts from all three providers

## Pros and Cons of the Options

### [option 1] Support only Google (single provider, fastest time to market)

Start with Google as the sole identity provider to minimize initial development effort.

* Good, because fastest time to market (single provider integration)
* Good, because Google has the largest user base globally
* Good, because simplest testing and monitoring (one provider)
* Good, because lowest maintenance overhead
* Good, because Google's OAuth2/OIDC implementation is excellent and well-documented
* Neutral, because golang.org/x/oauth2 has excellent Google support
* Bad, because excludes significant user segments (social users prefer Facebook, iOS users expect Apple login)
* Bad, because creates vendor lock-in perception
* Bad, because limits testing of multi-provider architecture
* Bad, because may require architecture changes when adding more providers later
* Bad, because cannot distribute iOS apps requiring Sign in with Apple

### [option 2] Support Google, Facebook, and Apple (three major providers)

Support three major identity providers covering different user segments.

* Good, because excellent user coverage across consumer, social, and mobile-first segments
* Good, because includes Apple which is required for future iOS app distribution
* Good, because Facebook provides access to large social user base
* Good, because validates multi-provider architecture early
* Good, because demonstrates provider neutrality to users
* Good, because reasonable development effort (6-9 days total estimated)
* Good, because covers both web and mobile use cases
* Neutral, because Apple requires special handling (Sign in with Apple mandatory for apps using social login)
* Neutral, because Facebook has large user base but privacy concerns may affect adoption
* Bad, because higher development cost than single provider
* Bad, because Apple's implementation is more complex (private email relay, app-specific IDs)
* Bad, because Facebook's API has had historical stability issues
* Bad, because excludes developer (GitHub) and enterprise (Microsoft) segments
* Bad, because three separate provider APIs to monitor and maintain

### [option 3] Support Google, GitHub, Microsoft, Facebook, and Apple (five providers for maximum coverage)

Support five major providers for maximum user convenience.

* Good, because covers nearly all potential users
* Good, because includes Apple (required for iOS apps in future)
* Good, because includes Facebook (large user base)
* Neutral, because Facebook and Apple have different OAuth2 implementation quirks
* Bad, because significantly higher development effort (10-15 days estimated)
* Bad, because Apple's implementation requires special handling (app-specific IDs)
* Bad, because Facebook's API stability and privacy concerns
* Bad, because increased testing complexity
* Bad, because more providers to monitor and maintain
* Bad, because delays time to market for initial release

### [option 4] Use a unified authentication service like Auth0 or Keycloak (provider abstraction layer)

Use a third-party authentication service that abstracts provider integrations.

* Good, because single integration point for multiple providers
* Good, because Auth0/Keycloak handle provider API changes
* Good, because additional features (user management UI, advanced security)
* Good, because reduces maintenance burden for provider integrations
* Neutral, because Auth0 has free tier but with usage limits
* Bad, because introduces external service dependency
* Bad, because cost at scale (Auth0 pricing can be significant)
* Bad, because Keycloak requires self-hosting and operational overhead
* Bad, because less control over authentication flow
* Bad, because vendor lock-in to authentication service
* Bad, because learning curve for Auth0/Keycloak configuration
* Bad, because may complicate debugging and error handling

## More Information

### Provider-Specific Details

**Google**
- OAuth2/OIDC: Excellent standards compliance, comprehensive documentation
- Library support: First-class support in golang.org/x/oauth2
- Rate limits: 10,000 requests per day (free tier), sufficient for expected usage
- User base: Largest global reach, preferred by general consumers
- Discovery document: https://accounts.google.com/.well-known/openid-configuration
- Implementation complexity: Low to medium

**Facebook**
- OAuth2/OIDC: OAuth2 support (not full OIDC), proprietary API for user data
- Library support: Supported in golang.org/x/oauth2 via facebook endpoint
- Rate limits: 200 calls per hour per user (varies by app tier), may need monitoring
- User base: Large social network user base (2.9+ billion monthly active users)
- Privacy considerations: Historical privacy concerns, may affect user trust
- Implementation complexity: Medium (requires Graph API for user profile data)
- Special requirements: App review process, privacy policy, terms of service required

**Apple**
- OAuth2/OIDC: Full OpenID Connect support (Sign in with Apple)
- Library support: Supported in golang.org/x/oauth2 via apple endpoint
- Rate limits: Generally sufficient for authentication use cases
- User base: Privacy-conscious users, iOS/macOS ecosystem (1.8+ billion active devices)
- Privacy features: Private email relay, user consent required for email sharing
- Implementation complexity: High (requires Apple Developer account, app-specific IDs, private email relay handling)
- Special requirements: Mandatory for iOS apps that offer other social login options
- Token considerations: Refresh tokens valid for 6 months, requiring periodic re-authentication

### Implementation Timeline

- Week 1: Google integration (primary focus, foundational implementation)
- Week 2: Facebook integration (Graph API integration for user profile)
- Week 3: Apple integration (Sign in with Apple, private email relay handling)
- Week 4: Integration testing, monitoring setup, documentation
- Ongoing: Apple Developer account setup, app review process for Facebook

### Future Provider Additions

Criteria for adding additional providers:
1. User demand (track requests for specific providers)
2. Strategic importance (e.g., Apple for iOS app)
3. OAuth2/OIDC standards compliance
4. Availability of Go library support
5. Provider stability and reputation
6. Rate limits sufficient for expected usage

Potential future providers (priority order):
1. GitHub (developer community, technical users)
2. Microsoft (enterprise users, work/school accounts)
3. Twitter/X (social media users, real-time communication)
4. LinkedIn (professional networking, B2B use cases)
5. Enterprise SSO (SAML, custom OIDC for organizations)

### Related Documentation

- [ADR-0002: SSO Authentication Strategy](0002-sso-authentication-strategy.md)
- [User Journey: User Registration](../../user-journeys/0001-user-registration.md)
- golang.org/x/oauth2 package: https://pkg.go.dev/golang.org/x/oauth2
- OAuth 2.0 specification: https://oauth.net/2/
- OpenID Connect specification: https://openid.net/connect/

### Configuration Approach

Provider configuration will be externalized (environment variables or config file) to allow:
- Different providers in different deployment environments
- Easy addition of new providers without code changes
- Provider-specific settings (client ID, client secret, scopes)
- Provider enable/disable without code deployment

### Integration with Fine-Grained Authorization

OAuth2/OIDC providers (Google, Facebook, Apple) handle **authentication** (verifying user identity), while fine-grained authorization (determining what resources a user can access) can be implemented using a relationship-based system like OpenFGA. This separation of concerns is a recommended pattern for modern SaaS applications.

**How it works:**
1. User authenticates with Google/Facebook/Apple → receives JWT containing user identity
2. API validates JWT and extracts user identifier from `sub` claim
3. User identifier is mapped to OpenFGA format (e.g., `user:google:123456`, `user:facebook:789`, `user:apple:xyz`)
4. API checks OpenFGA to determine if user has required relationship to resource (e.g., "Can this user view journey X?")
5. OpenFGA returns allow/deny decision based on stored relationships (owner, editor, viewer, workspace member, etc.)

**Example authorization check:**
```go
// After validating JWT from Google/Facebook/Apple
userId := fmt.Sprintf("user:%s:%s", provider, jwtClaims["sub"])

// Check with OpenFGA
allowed := fgaClient.Check(ctx, userId, "viewer", "journey:550e8400-...")
```

This approach enables sophisticated access control patterns (ownership, sharing, team access, hierarchical permissions) while maintaining the flexibility to use any OAuth2/OIDC provider for authentication. See [OpenFGA Research](../../analysis/open-source/openfga.md) for detailed integration patterns and deployment strategies.
