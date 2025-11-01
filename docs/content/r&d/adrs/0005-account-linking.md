---
title: "[0005] Account Linking Strategy for Multiple OAuth2/OIDC Providers"
description: >
    Strategy for allowing users to link multiple OAuth2/OIDC provider identities (Google, Facebook, Apple) to a single user account.
type: docs
weight: 5
status: "accepted"
date: 2025-11-01
deciders: []
consulted: []
informed: []
---

## Context and Problem Statement

The application supports authentication through multiple OAuth2/OIDC providers (Google, Facebook, Apple) as established in [ADR-0003](/r&d/adrs/0003-oauth2-oidc-provider-selection/). Users may want to sign in with different providers to access the same account, or may forget which provider they originally used for registration (see [User Journey 0001: User Registration](/r&d/user-journeys/0001-user-registration/) REQ-AC-004 and [User Journey 0002: User Login via SSO](/r&d/user-journeys/0002-user-login-via-sso/) REQ-OT-010).

**The key question:** How should the system handle the linking of multiple provider identities to a single user account?

Current implementation already supports storing multiple provider identities per user with the pattern `{provider}:{sub}` (e.g., `google:123456789`, `facebook:789`), but the strategy for _when_ and _how_ to link accounts remains undecided.

## Decision Drivers

* **User Convenience** - Users who registered with Google should be able to sign in with Facebook later without creating a duplicate account
* **Security** - Prevent unauthorized account takeover through malicious account linking
* **Privacy** - Avoid automatically linking accounts without user awareness
* **User Confusion** - Some users forget which provider they used for registration
* **Email Verification Trust** - OAuth providers offer varying levels of email verification guarantees
* **UX Complexity** - Minimize friction while maintaining security
* **Implementation Effort** - Balance feature value against development and testing cost
* **Support Burden** - Reduce user support requests about "wrong account" or "can't access my data"

## Considered Options

* **Option 1: Automatic Linking via Verified Email Match**
* **Option 2: Explicit User-Initiated Linking Only**
* **Option 3: Hybrid Approach (Automatic with User Confirmation)**
* **Option 4: No Account Linking (Each Provider = Separate Account)**

## Decision Outcome

Chosen option: "**Option 2: Explicit User-Initiated Linking Only**", because it provides the most secure approach while giving users full control and transparency over their account linking decisions.

### Consequences

* Good, because it prevents automatic account takeover scenarios where a compromised email could lead to unauthorized account linking
* Good, because users are fully aware and in control of when and how their provider identities are linked
* Good, because users must prove they control both identities before linking occurs, adding a critical security validation step
* Good, because the approach is transparent and predictable, making it easier to explain in security audits and user documentation
* Good, because users can review and manage their linked providers, including unlinking providers they no longer wish to use
* Bad, because it requires additional development effort to implement new API endpoints (`POST /v1/account/link/{provider}`, `DELETE /v1/account/unlink/{provider}`) and UI components for account management
* Bad, because it adds friction to the user experience by requiring explicit action rather than automatic linking
* Bad, because it doesn't automatically solve the "forgot which provider" problem - users must remember or try multiple providers during login
* Bad, because some users may ignore linking prompts and inadvertently create duplicate accounts with the same email
* Neutral, because while it requires more implementation work, the security benefits and user control outweigh the development cost for a P1 requirement

### Confirmation

* Account linking flows will be tested with all three OAuth providers (Google, Facebook, Apple)
* Security review to verify no account takeover vulnerabilities
* User journey documentation updated to reflect chosen approach
* Integration tests verifying link/unlink operations
* Monitoring for account linking errors and user complaints

## Pros and Cons of the Options

### Option 1: Automatic Linking via Verified Email Match

When a user authenticates with a provider, if the verified email matches an existing account, automatically link the provider identity to that account.

**Flow:**
1. User logs in with Google → email: user@example.com (verified)
2. System finds existing account registered with Facebook using same email
3. System automatically links Google identity to existing account
4. User is logged into existing account with both Google and Facebook identities linked

**Requirements:**
- Only link when email is marked as `email_verified: true` in OAuth claims
- Store all linked provider identities: `google:123`, `facebook:456`
- Update session JWT to reflect primary account
- Log account linking events for audit

* Good, because it provides seamless user experience with zero friction
* Good, because it solves the "forgot which provider" problem automatically
* Good, because Google and Apple provide reliable `email_verified` claims
* Neutral, because Facebook's email verification is less reliable (may require additional checks)
* Bad, because it could enable account takeover if attacker controls the email (e.g., compromised email account)
* Bad, because users may not realize accounts are being merged (privacy concern)
* Bad, because email addresses can be recycled (old owner loses account, new owner gains access)
* Bad, because it assumes email ownership is permanent and secure

**Security Risks:**
- **Email Account Compromise**: If attacker gains access to user's email, they can link their OAuth provider to victim's account
- **Email Recycling**: Providers may reassign email addresses (especially corporate emails after employee departure)
- **Unverified Emails**: Facebook's Graph API may return unverified emails
- **No User Awareness**: User might not notice their accounts were merged until later

### Option 2: Explicit User-Initiated Linking Only

Require users to explicitly initiate account linking through a deliberate action (e.g., settings page, or during login flow).

**Flow - During Login:**
1. User logs in with Google → email: user@example.com
2. System finds existing account with same email registered via Facebook
3. System shows: "An account with this email already exists. Sign in with Facebook to link accounts."
4. User signs in with Facebook (within same session or with special link token)
5. System links Google identity to Facebook account after confirming user controls both

**Flow - Settings Page:**
1. User is logged in with Google
2. User navigates to Account Settings → Linked Providers
3. User clicks "Link Facebook Account"
4. User completes Facebook OAuth flow
5. System verifies email match and links provider

**Requirements:**
- New API endpoints: `POST /v1/account/link/{provider}`, `DELETE /v1/account/unlink/{provider}`
- UI for managing linked providers in settings
- During login: detection of potential link + prompt to complete linking
- Linking tokens with expiration (similar to OAuth state, 10-minute TTL)

* Good, because it prevents automatic account takeover scenarios
* Good, because user is fully aware and in control of account linking
* Good, because user must prove they control both identities
* Good, because it's more predictable and transparent to users
* Neutral, because it requires additional UI and API endpoints
* Bad, because it adds friction to the user experience
* Bad, because it doesn't solve "forgot which provider" problem automatically (user must remember or try multiple providers)
* Bad, because it requires more development effort (new endpoints, UI, flows)
* Bad, because users may ignore linking prompts and create duplicate accounts anyway

**Security Benefits:**
- User explicitly proves they control both identities
- No automatic linking without user action
- Easier to explain in security audit
- User can review and unlink providers

### Option 3: Hybrid Approach (Automatic with User Confirmation)

Automatically detect matching verified emails, but require user confirmation before linking.

**Flow:**
1. User logs in with Google → email: user@example.com (verified)
2. System finds existing account with same email (Facebook)
3. System shows confirmation prompt: "You have an existing account with this email. Link your Google account to it?"
4. User confirms → accounts are linked
5. User declines → new separate account is created

**Requirements:**
- Temporary "pending link" state stored with linking token
- Confirmation UI during OAuth callback flow
- Fallback to create new account if user declines
- Email notification when linking occurs

* Good, because it balances convenience with user awareness
* Good, because it prevents silent account takeover while reducing friction
* Good, because user makes informed decision during login flow
* Good, because it solves "forgot which provider" problem with minimal friction
* Neutral, because it adds one confirmation step to the flow
* Neutral, because user might accidentally click through confirmation without reading
* Bad, because it still requires UI/UX design and implementation
* Bad, because it's more complex than fully automatic or fully explicit approaches
* Bad, because users who decline will create duplicate accounts (same email, different providers)

**Trade-offs:**
- More secure than Option 1, less secure than Option 2
- Better UX than Option 2, worse UX than Option 1
- Medium implementation complexity

### Option 4: No Account Linking (Each Provider = Separate Account)

Each OAuth provider identity creates and maintains a completely separate user account. No linking capability.

**Flow:**
1. User registers with Google → creates account A
2. User later logs in with Facebook (same email) → creates account B
3. User has two separate accounts with separate data

* Good, because it's the simplest to implement (already works this way)
* Good, because there's zero risk of account takeover via linking
* Good, because no additional development effort required
* Bad, because users will have duplicate accounts and fragmented data
* Bad, because it creates significant user confusion and support burden
* Bad, because it violates the requirements documented in REQ-AC-004 (P1) and REQ-OT-010 (P2)
* Bad, because users may not realize they created multiple accounts until much later
* Bad, because it provides poor user experience

**Why This Doesn't Meet Requirements:**
- Explicitly conflicts with REQ-AC-004: "Support multiple identity providers per user account"
- Doesn't address REQ-OT-010: "Support account linking workflow for users who forgot which provider they used"

## More Information

### Related Requirements

From [User Journey 0001: User Registration](/r&d/user-journeys/0001-user-registration/):
- **REQ-AC-004** (P1): "Support multiple identity providers per user account (account linking)" - Allows users to sign in with different providers to the same account

From [User Journey 0002: User Login via SSO](/r&d/user-journeys/0002-user-login-via-sso/):
- **REQ-OT-010** (P2): "Support account linking workflow for users who registered with different provider" - Helps users who forget which provider they used

### Related ADRs

- [ADR-0002: SSO Authentication Strategy](/r&d/adrs/0002-sso-authentication-strategy/) - Established OAuth2/OIDC approach
- [ADR-0003: OAuth2/OIDC Provider Selection](/r&d/adrs/0003-oauth2-oidc-provider-selection/) - Selected Google, Facebook, Apple
- [ADR-0004: Session Management](/r&d/adrs/0004-session-management/) - Stateless JWT approach affects account linking

### Implementation Considerations

Regardless of chosen option:

1. **Data Model**: User table must support multiple provider identities
   ```
   users table:
   - id (primary key)
   - email (from first provider)
   - created_at

   user_providers table:
   - user_id (foreign key)
   - provider (google|facebook|apple)
   - provider_user_id (sub claim)
   - email (from provider)
   - email_verified (boolean)
   - linked_at (timestamp)
   - primary key: (provider, provider_user_id)
   ```

2. **Email Verification Trust Levels**:
   - **Google**: Highly reliable `email_verified` claim
   - **Apple**: Reliable, always verified
   - **Facebook**: Less reliable, may need Graph API verification

3. **Authorization Integration**: OpenFGA relationships use `user:{provider}:{sub}` format
   - Account linking must ensure all linked identities can access same resources
   - May need to update OpenFGA tuples when linking occurs

4. **Session Management**: JWT contains user identity
   - Account linking must invalidate old sessions or update tokens
   - Consistent user ID across all linked providers

5. **Unlinking Support**: Should users be able to unlink providers?
   - Must ensure at least one provider remains linked
   - What happens to OAuth provider data after unlinking?

### Future User Journey

A dedicated [User Journey: Account Linking] should be created to document:
- Discovery flow (user realizes they have/want multiple providers)
- Linking flow (step-by-step process)
- Unlinking flow
- Error cases (email mismatch, unverified email, etc.)
- UI/UX mockups
