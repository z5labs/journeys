---
title: "Mobile OAuth Security Considerations"
description: >
    Analysis of OAuth 2.0 security requirements for native mobile applications (iOS and Android)
type: docs
weight: 2
date: 2025-11-02
---

## Overview

This document analyzes the security requirements and best practices for implementing OAuth 2.0 authentication in native mobile applications (iOS and Android). Mobile apps have different security characteristics compared to web applications, requiring specific patterns to prevent authorization code interception, token theft, and other mobile-specific attack vectors.

**Key Standard**: [RFC 8252 - OAuth 2.0 for Native Apps](https://www.rfc-editor.org/rfc/rfc8252.html) (October 2017)

## Why Mobile Apps Need Special Handling

Mobile applications face unique security challenges that don't apply to traditional web applications:

### 1. No Client Secret Storage
**Problem**: Native mobile apps are distributed to end users, who can decompile the app binary and extract any embedded secrets.

**Impact**: Mobile apps cannot act as "confidential clients" in OAuth terminology. Any client secret embedded in the app bundle can be extracted by attackers using reverse engineering tools.

**Solution**: Mobile apps must use PKCE (Proof Key for Code Exchange) instead of client secrets to secure the authorization code exchange.

### 2. App Impersonation Risk
**Problem**: Malicious apps can register custom URL schemes (e.g., `myapp://callback`) that conflict with legitimate apps, allowing them to intercept authorization callbacks.

**Impact**: An attacker's app could receive the authorization code intended for the legitimate app, then exchange it for access tokens.

**Solution**: Use Universal Links (iOS) or App Links (Android) instead of custom URL schemes. These require domain ownership verification via HTTPS, preventing impersonation.

### 3. Embedded WebView Risks
**Problem**: Apps could use embedded WebViews to display OAuth login pages, allowing the app to access credentials, session cookies, and tokens directly.

**Impact**: Malicious apps could steal user credentials, session cookies from other sites, or OAuth tokens. Users can't distinguish a fake login page from a real one inside a WebView.

**Solution**: RFC 8252 mandates using the system browser (ASWebAuthenticationSession on iOS, Chrome Custom Tabs on Android) for OAuth flows. System browsers provide:
- Visible URL bar (users can verify the domain)
- Separate process/sandbox from the app
- Shared authentication state with Safari/Chrome (SSO)
- Protection from credential theft

### 4. Token Storage Security
**Problem**: Mobile devices can be lost, stolen, or compromised. Tokens stored insecurely (e.g., in UserDefaults, SharedPreferences, or plaintext files) can be extracted by attackers with physical access or malware.

**Impact**: Stolen access tokens grant attackers full API access as the victim user. Stolen refresh tokens allow long-term account access.

**Solution**: Use platform-provided secure storage with hardware-backed encryption:
- **iOS**: Keychain Services with `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`
- **Android**: Android Keystore with AES256-GCM encryption

## RFC 8252 Requirements

RFC 8252 establishes the following requirements for OAuth 2.0 in native apps:

### Requirement 1: PKCE (Proof Key for Code Exchange)

**Status**: MUST (mandatory for all native apps)

**What It Does**: PKCE prevents authorization code interception attacks by cryptographically binding the authorization request to the token exchange request.

**How It Works**:
1. App generates a random `code_verifier` (43-128 characters)
2. App computes `code_challenge = SHA256(code_verifier)`
3. App sends `code_challenge` in the authorization request
4. OAuth provider stores the challenge
5. App sends the original `code_verifier` in the token exchange
6. OAuth provider verifies `SHA256(code_verifier) == stored_code_challenge`

**Why It's Required**: An attacker who intercepts the authorization code cannot exchange it for tokens without the original `code_verifier`, which only the legitimate app possesses.

**Implementation Example (iOS/Swift)**:
```swift
import CryptoKit

func generatePKCE() -> (verifier: String, challenge: String) {
    // Generate 32 random bytes (43 base64url characters)
    var bytes = [UInt8](repeating: 0, count: 32)
    _ = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
    let verifier = Data(bytes).base64URLEncodedString()

    // Compute SHA256 hash and base64url encode
    let verifierData = verifier.data(using: .ascii)!
    let hash = SHA256.hash(data: verifierData)
    let challenge = Data(hash).base64URLEncodedString()

    return (verifier, challenge)
}
```

**Implementation Example (Android/Kotlin)**:
```kotlin
import java.security.MessageDigest
import java.security.SecureRandom
import android.util.Base64

fun generatePKCE(): Pair<String, String> {
    // Generate 32 random bytes (43 base64url characters)
    val bytes = ByteArray(32)
    SecureRandom().nextBytes(bytes)
    val verifier = Base64.encodeToString(bytes, Base64.URL_SAFE or Base64.NO_WRAP or Base64.NO_PADDING)

    // Compute SHA256 hash and base64url encode
    val digest = MessageDigest.getInstance("SHA-256")
    val hash = digest.digest(verifier.toByteArray(Charsets.US_ASCII))
    val challenge = Base64.encodeToString(hash, Base64.URL_SAFE or Base64.NO_WRAP or Base64.NO_PADDING)

    return Pair(verifier, challenge)
}
```

### Requirement 2: System Browser (External User-Agent)

**Status**: MUST (embedded WebViews MUST NOT be used)

**What It Does**: OAuth login pages must be displayed in the system browser (Safari on iOS, Chrome on Android) rather than an in-app WebView.

**Platform APIs**:
- **iOS**: `ASWebAuthenticationSession` (iOS 12+)
- **Android**: Chrome Custom Tabs (via AndroidX library)

**Why It's Required**:
- Prevents the app from accessing user credentials
- Provides visible URL bar for phishing protection
- Enables SSO (users already authenticated in browser stay logged in)
- Separates security contexts (app can't inject JavaScript, read cookies)

**Implementation Example (iOS/Swift)**:
```swift
import AuthenticationServices

class OAuthManager: NSObject, ASWebAuthenticationPresentationContextProviding {
    func authenticate(provider: String, codeChallenge: String) {
        let authURL = URL(string: "https://yourapp.com/v1/auth/\(provider)?code_challenge=\(codeChallenge)&code_challenge_method=S256&platform=mobile")!
        let callbackScheme = "https" // Use https for Universal Links

        let session = ASWebAuthenticationSession(
            url: authURL,
            callbackURLScheme: callbackScheme,
            completionHandler: { callbackURL, error in
                guard let url = callbackURL else { return }
                self.handleCallback(url: url)
            }
        )

        session.presentationContextProvider = self
        session.prefersEphemeralWebBrowserSession = false // Allow SSO
        session.start()
    }

    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        return UIApplication.shared.windows.first { $0.isKeyWindow }!
    }
}
```

**Implementation Example (Android/Kotlin)**:
```kotlin
import androidx.browser.customtabs.CustomTabsIntent
import android.net.Uri

class OAuthManager(private val activity: AppCompatActivity) {
    fun authenticate(provider: String, codeChallenge: String) {
        val authUrl = Uri.parse(
            "https://yourapp.com/v1/auth/$provider" +
            "?code_challenge=$codeChallenge" +
            "&code_challenge_method=S256" +
            "&platform=mobile"
        )

        val customTabsIntent = CustomTabsIntent.Builder()
            .setShowTitle(true)
            .setUrlBarHidingEnabled(false) // Keep URL bar visible
            .build()

        customTabsIntent.launchUrl(activity, authUrl)
    }
}
```

### Requirement 3: Deep Linking (Universal Links / App Links)

**Status**: SHOULD (strongly recommended over custom URL schemes)

**What It Does**: Allows the system browser to redirect back to your app after OAuth completion using HTTPS-based URLs that you own.

**Why It's Required**: Custom URL schemes (e.g., `myapp://callback`) can be hijacked by malicious apps. Universal Links/App Links require domain ownership verification, preventing impersonation.

**Platform Requirements**:

**iOS Universal Links**:
1. Register Associated Domains in Xcode entitlements:
   ```xml
   <key>com.apple.developer.associated-domains</key>
   <array>
       <string>applinks:yourapp.com</string>
   </array>
   ```

2. Host `apple-app-site-association` file at `https://yourapp.com/.well-known/apple-app-site-association`:
   ```json
   {
     "applinks": {
       "apps": [],
       "details": [
         {
           "appID": "TEAMID.com.yourcompany.yourapp",
           "paths": ["/auth/callback"]
         }
       ]
     }
   }
   ```

3. Handle incoming links in AppDelegate:
   ```swift
   func application(_ application: UIApplication,
                    continue userActivity: NSUserActivity,
                    restorationHandler: @escaping ([UIUserActivityRestoring]?) -> Void) -> Bool {
       guard userActivity.activityType == NSUserActivityTypeBrowsingWeb,
             let url = userActivity.webpageURL else {
           return false
       }

       // Handle OAuth callback URL
       if url.path.starts(with: "/auth/callback") {
           handleOAuthCallback(url: url)
           return true
       }

       return false
   }
   ```

**Android App Links**:
1. Declare intent filter in `AndroidManifest.xml`:
   ```xml
   <intent-filter android:autoVerify="true">
       <action android:name="android.intent.action.VIEW" />
       <category android:name="android.intent.category.DEFAULT" />
       <category android:name="android.intent.category.BROWSABLE" />
       <data
           android:scheme="https"
           android:host="yourapp.com"
           android:pathPrefix="/auth/callback" />
   </intent-filter>
   ```

2. Host Digital Asset Links file at `https://yourapp.com/.well-known/assetlinks.json`:
   ```json
   [
     {
       "relation": ["delegate_permission/common.handle_all_urls"],
       "target": {
         "namespace": "android_app",
         "package_name": "com.yourcompany.yourapp",
         "sha256_cert_fingerprints": [
           "14:6D:E9:83:C5:73:06:50:D8:EE:B9:95:2F:34:FC:64:16:A0:83:42:E6:1D:BE:A8:8A:04:96:B2:3F:CF:44:E5"
         ]
       }
     }
   ]
   ```

3. Handle incoming links in Activity:
   ```kotlin
   override fun onCreate(savedInstanceState: Bundle?) {
       super.onCreate(savedInstanceState)

       val action = intent?.action
       val data = intent?.data

       if (Intent.ACTION_VIEW == action && data != null) {
           handleOAuthCallback(data)
       }
   }
   ```

### Requirement 4: Secure Token Storage

**Status**: MUST (tokens must be protected from unauthorized access)

**What It Does**: Stores access tokens, refresh tokens, and session tokens using platform-provided secure storage with hardware-backed encryption.

**iOS Keychain**:
```swift
import Security

class TokenStorage {
    func saveToken(_ token: String, forKey key: String) {
        let data = token.data(using: .utf8)!

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        ]

        // Delete existing item
        SecItemDelete(query as CFDictionary)

        // Add new item
        let status = SecItemAdd(query as CFDictionary, nil)
        guard status == errSecSuccess else {
            fatalError("Failed to save token: \(status)")
        }
    }

    func loadToken(forKey key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess,
              let data = result as? Data,
              let token = String(data: data, encoding: .utf8) else {
            return nil
        }

        return token
    }
}
```

**Android Keystore**:
```kotlin
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKeys

class TokenStorage(context: Context) {
    private val masterKeyAlias = MasterKeys.getOrCreate(MasterKeys.AES256_GCM_SPEC)

    private val encryptedPrefs = EncryptedSharedPreferences.create(
        "oauth_tokens",
        masterKeyAlias,
        context,
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
    )

    fun saveToken(token: String, key: String) {
        encryptedPrefs.edit()
            .putString(key, token)
            .apply()
    }

    fun loadToken(key: String): String? {
        return encryptedPrefs.getString(key, null)
    }
}
```

## Security Comparison: Mobile vs Web

| Security Aspect | Web Applications | Mobile Applications |
|----------------|------------------|---------------------|
| **Client Secret** | Can securely store client secret server-side | Cannot store secrets (distributed to users) |
| **PKCE Requirement** | Optional (recommended) | Mandatory (RFC 8252) |
| **OAuth UI Display** | In-page or redirect to provider | Must use system browser (no WebViews) |
| **Callback Mechanism** | HTTP redirect to same domain | Deep linking (Universal Links/App Links) |
| **Token Storage** | httpOnly cookies (server-side sessions) | Keychain (iOS) / Keystore (Android) |
| **State Parameter** | Stored in cookie or server session | Generated and held in memory during flow |
| **Primary Attack Vector** | CSRF, session hijacking | Authorization code interception, token theft |

## Attack Scenarios and Mitigations

### Attack 1: Authorization Code Interception

**Scenario**: Attacker registers malicious app with same custom URL scheme (e.g., `myapp://callback`), intercepts authorization code intended for legitimate app.

**Mitigation**:
- Use Universal Links (iOS) or App Links (Android) instead of custom URL schemes
- Universal Links/App Links require HTTPS domain ownership verification
- OS prevents multiple apps from claiming the same HTTPS URL

**Status**: Mitigated by RFC 8252 deep linking requirements

### Attack 2: Authorization Code Replay Without PKCE

**Scenario**: Attacker intercepts authorization code during network transmission, exchanges it for access tokens before legitimate app.

**Mitigation**:
- Implement PKCE (mandatory in RFC 8252)
- Attacker cannot exchange code without the `code_verifier`
- OAuth provider validates `SHA256(code_verifier) == code_challenge`

**Status**: Mitigated by mandatory PKCE

### Attack 3: Credential Theft via Embedded WebView

**Scenario**: Malicious app displays OAuth login page in embedded WebView, injects JavaScript to steal credentials or session cookies.

**Mitigation**:
- Use system browser (ASWebAuthenticationSession / Chrome Custom Tabs)
- System browser runs in separate process/sandbox from app
- App has no access to browser DOM, JavaScript, or cookies

**Status**: Mitigated by RFC 8252 external user-agent requirement

### Attack 4: Token Theft from Insecure Storage

**Scenario**: Attacker gains physical access to device or installs malware, extracts tokens from insecure storage (UserDefaults, SharedPreferences).

**Mitigation**:
- Store tokens in iOS Keychain with `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`
- Store tokens in Android Keystore with AES256-GCM encryption
- Use `EncryptedSharedPreferences` library on Android

**Status**: Mitigated by secure storage APIs

### Attack 5: Man-in-the-Middle (MITM) Attacks

**Scenario**: Attacker intercepts network traffic between app and OAuth provider, steals authorization code or tokens.

**Mitigation**:
- Enforce HTTPS for all OAuth endpoints (authorization, token exchange)
- Implement certificate pinning for high-security apps
- PKCE prevents code theft (attacker still can't exchange without `code_verifier`)
- Use short-lived access tokens with refresh token rotation

**Status**: Partially mitigated by HTTPS + PKCE (certificate pinning recommended for high-security scenarios)

## Mobile-Specific OAuth Flow (Complete Example)

This section provides a complete end-to-end example of a mobile OAuth flow following RFC 8252.

### 1. User Initiates Login

```swift
// iOS Example
let (codeVerifier, codeChallenge) = generatePKCE()
let state = generateRandomState() // 32 bytes, base64url encoded

// Store code_verifier and state in memory (instance variables)
self.codeVerifier = codeVerifier
self.expectedState = state

let authURL = URL(string: "https://yourapp.com/v1/auth/google?code_challenge=\(codeChallenge)&code_challenge_method=S256&state=\(state)&platform=mobile")!

let session = ASWebAuthenticationSession(
    url: authURL,
    callbackURLScheme: "https",
    completionHandler: handleOAuthCallback
)
session.start()
```

### 2. Backend Redirects to OAuth Provider

```
Backend receives: GET /v1/auth/google?code_challenge=ABC...&code_challenge_method=S256&state=XYZ...&platform=mobile

Backend generates server-side state parameter (separate from mobile state)
Backend redirects to: https://accounts.google.com/o/oauth2/v2/auth?
  client_id=YOUR_CLIENT_ID
  &redirect_uri=https://yourapp.com/v1/auth/google/callback
  &response_type=code
  &scope=openid email profile
  &code_challenge=ABC...
  &code_challenge_method=S256
  &state=SERVER_STATE
```

### 3. User Authenticates with Google

System browser displays Google login page. User authenticates and approves permissions.

### 4. Google Redirects to Callback

```
Google redirects to: https://yourapp.com/v1/auth/google/callback?code=4/AUTHORIZATION_CODE&state=SERVER_STATE
```

### 5. Backend Validates and Redirects to App

```
Backend validates server-side state
Backend redirects to Universal Link: https://yourapp.com/auth/callback?code=4/AUTHORIZATION_CODE&state=XYZ...&session_token=SESSION_TOKEN
```

### 6. App Receives Deep Link

```swift
// iOS Example
func handleOAuthCallback(callbackURL: URL?, error: Error?) {
    guard let url = callbackURL else { return }

    let components = URLComponents(url: url, resolvingAgainstBaseURL: false)
    let code = components?.queryItems?.first(where: { $0.name == "code" })?.value
    let state = components?.queryItems?.first(where: { $0.name == "state" })?.value
    let sessionToken = components?.queryItems?.first(where: { $0.name == "session_token" })?.value

    // Validate state parameter
    guard state == self.expectedState else {
        print("Invalid state parameter - possible CSRF attack")
        return
    }

    // Store session token securely
    TokenStorage().saveToken(sessionToken!, forKey: "session_token")

    // Navigate to dashboard
    DispatchQueue.main.async {
        self.showDashboard()
    }
}
```

### 7. Authenticated API Requests

```swift
// iOS Example
func makeAuthenticatedRequest() {
    guard let sessionToken = TokenStorage().loadToken(forKey: "session_token") else {
        return
    }

    var request = URLRequest(url: URL(string: "https://yourapp.com/v1/journeys")!)
    request.setValue("Bearer \(sessionToken)", forHTTPHeaderField: "Authorization")

    URLSession.shared.dataTask(with: request) { data, response, error in
        // Handle response
    }.resume()
}
```

## Industry Best Practices (2024)

### Recommended: AppAuth SDK

Both iOS and Android have mature, well-maintained OAuth libraries that implement RFC 8252:

**iOS**: [AppAuth-iOS](https://github.com/openid/AppAuth-iOS)
```swift
import AppAuth

let request = OIDAuthorizationRequest(
    configuration: serviceConfig,
    clientId: "YOUR_CLIENT_ID",
    scopes: ["openid", "profile", "email"],
    redirectURL: URL(string: "https://yourapp.com/auth/callback")!,
    responseType: OIDResponseTypeCode,
    additionalParameters: ["platform": "mobile"]
)

let appDelegate = UIApplication.shared.delegate as! AppDelegate
appDelegate.currentAuthorizationFlow = OIDAuthState.authState(
    byPresenting: request,
    presenting: self
) { authState, error in
    if let authState = authState {
        // Save authState (contains access token, refresh token, etc.)
        self.setAuthState(authState)
    }
}
```

**Android**: [AppAuth-Android](https://github.com/openid/AppAuth-Android)
```kotlin
import net.openid.appauth.*

val serviceConfig = AuthorizationServiceConfiguration(
    Uri.parse("https://accounts.google.com/o/oauth2/v2/auth"),
    Uri.parse("https://oauth2.googleapis.com/token")
)

val request = AuthorizationRequest.Builder(
    serviceConfig,
    "YOUR_CLIENT_ID",
    ResponseTypeValues.CODE,
    Uri.parse("https://yourapp.com/auth/callback")
).setScopes("openid", "profile", "email")
 .setAdditionalParameters(mapOf("platform" to "mobile"))
 .build()

val authService = AuthorizationService(this)
val authIntent = authService.getAuthorizationRequestIntent(request)
startActivityForResult(authIntent, RC_AUTH)
```

### Refresh Token Best Practices

**Recommendation**: Use refresh tokens with rotation for long-lived sessions

**Why**: Access tokens should be short-lived (15-60 minutes). Refresh tokens allow the app to obtain new access tokens without requiring the user to re-authenticate.

**Security Considerations**:
- Store refresh tokens in Keychain/Keystore (same as access tokens)
- Implement refresh token rotation (OAuth provider issues new refresh token with each refresh)
- Revoke old refresh token immediately after rotation
- Detect refresh token theft via anomaly detection (multiple refreshes from different IPs)

**Implementation Example**:
```swift
func refreshAccessToken(completion: @escaping (String?) -> Void) {
    guard let refreshToken = TokenStorage().loadToken(forKey: "refresh_token") else {
        completion(nil)
        return
    }

    var request = URLRequest(url: URL(string: "https://yourapp.com/v1/auth/refresh")!)
    request.httpMethod = "POST"
    request.setValue("Bearer \(refreshToken)", forHTTPHeaderField: "Authorization")

    URLSession.shared.dataTask(with: request) { data, response, error in
        guard let data = data,
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: String],
              let newAccessToken = json["access_token"],
              let newRefreshToken = json["refresh_token"] else {
            completion(nil)
            return
        }

        // Store new tokens
        TokenStorage().saveToken(newAccessToken, forKey: "access_token")
        TokenStorage().saveToken(newRefreshToken, forKey: "refresh_token")

        completion(newAccessToken)
    }.resume()
}
```

## Recommendations for This Project

Based on ADR-0007 (User Registration) and this analysis:

### For Custom-Built OAuth UI (Chosen Option)

1. **Backend OAuth Endpoints** (`/v1/auth/{provider}` and `/v1/auth/{provider}/callback`):
   - Accept `code_challenge` query parameter from mobile apps
   - Store code challenge in encrypted server-side session
   - Validate `code_verifier` during token exchange
   - Return session token via Universal Link redirect

2. **Mobile App Integration**:
   - Use AppAuth-iOS and AppAuth-Android SDKs (battle-tested, RFC 8252 compliant)
   - Generate PKCE in mobile app, send challenge to backend
   - Use ASWebAuthenticationSession (iOS) / Chrome Custom Tabs (Android)
   - Register Universal Links (iOS) / App Links (Android) for `https://yourapp.com/auth/callback`
   - Store session tokens in Keychain (iOS) / Keystore (Android)

3. **OAuth Scopes**:
   - Web: `openid email profile`
   - Mobile: `openid email profile offline_access` (include refresh tokens for long-lived sessions)

4. **Token Lifetime**:
   - Access tokens: 15-60 minutes
   - Refresh tokens: 30-90 days with rotation
   - Session tokens (web): 7-30 days

### Security Checklist

- [ ] PKCE implemented and mandatory for all mobile OAuth flows
- [ ] System browser used (ASWebAuthenticationSession / Chrome Custom Tabs)
- [ ] Universal Links (iOS) and App Links (Android) configured with domain verification files
- [ ] Tokens stored in Keychain (iOS) with `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`
- [ ] Tokens stored in Android Keystore with AES256-GCM encryption via `EncryptedSharedPreferences`
- [ ] State parameter validation on both client and server side
- [ ] HTTPS enforced for all OAuth endpoints
- [ ] Refresh token rotation implemented
- [ ] Short-lived access tokens (15-60 minutes)
- [ ] Certificate pinning considered for high-security scenarios

## Related Documentation

- [ADR-0007: User Registration](../../adrs/0007-user-registration.md) - Mobile integration for all three implementation options
- [OAuth State and PKCE Storage Alternatives](pkce.md) - Analysis of OAuth state storage for web applications
- [RFC 8252: OAuth 2.0 for Native Apps](https://www.rfc-editor.org/rfc/rfc8252.html)
- [RFC 7636: Proof Key for Code Exchange (PKCE)](https://www.rfc-editor.org/rfc/rfc7636.html)
