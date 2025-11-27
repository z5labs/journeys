---
title: "htmx Data Transfer Analysis"
type: docs
linkTitle: "htmx Data Transfer"
weight: 10
description: >
  Analysis of htmx request and response content types, headers, and data transfer mechanisms between client and server.
---

## Overview

htmx is a JavaScript library that enables modern browser features directly from HTML attributes, allowing any element to issue AJAX requests. Unlike traditional AJAX applications that exchange JSON, **htmx uses HTML as the primary data format**, keeping applications within the Hypermedia As The Engine Of Application State (HATEOAS) model.

## Request Data Transfer

### Default Encoding: application/x-www-form-urlencoded

By default, htmx sends requests using **`application/x-www-form-urlencoded`** encoding, the same format used by standard HTML forms.

**How parameters are collected:**
- The triggering element's value (if it has one) is included
- For form elements, all input values within the form are included
- For non-GET requests, values from the associated form are automatically included
- Parameter names come from the `name` attribute of input elements

**Example:**
```html
<button hx-post="/clicked" hx-target="#result">
    Click Me
</button>
```

### Multipart Form Data Encoding

For file uploads, use the `hx-encoding` attribute to switch to **`multipart/form-data`**:

**Attribute:** `hx-encoding="multipart/form-data"`

This causes htmx to use the `FormData` API to construct the request body, properly encoding file data along with other form fields.

**Example:**
```html
<form hx-encoding='multipart/form-data' hx-post='/upload'>
    <input type='file' name='file'>
    <button>Upload</button>
    <progress id='progress' value='0' max='100'></progress>
</form>
<script>
    htmx.on('#form', 'htmx:xhr:progress', function(evt) {
        htmx.find('#progress').setAttribute('value', 
            evt.detail.loaded/evt.detail.total * 100)
    });
</script>
```

**Key Points:**
- File uploads require `multipart/form-data` encoding
- Server-side handling differs significantly from URL-encoded requests
- htmx fires `htmx:xhr:progress` events during upload for progress tracking
- `multipart/form-data` is inherited and can be set on parent elements

### Adding Custom Parameters

#### hx-vals Attribute

Add additional parameters using JSON notation:

```html
<!-- Static JSON values -->
<div hx-get="/example" hx-vals='{"myVal": "My Value"}'>
    Get Some HTML
</div>

<!-- Dynamic JavaScript evaluation -->
<div hx-get="/example" hx-vals='js:{myVal: calculateValue()}'>
    Get Some HTML with Dynamic Value
</div>

<!-- Access event object -->
<div hx-get="/example" hx-trigger="keyup" hx-vals='js:{lastKey: event.key}'>
    <input type="text" />
</div>

<!-- Spread operator for multiple values -->
<div hx-get="/example" hx-vals='js:{...foo()}'>
    Get Some HTML, Including All Values from foo()
</div>
```

**Format:**
- Default: Valid JSON (not dynamically evaluated)
- With `javascript:` or `js:` prefix: JavaScript expression evaluated at runtime
- `this` refers to the element with the `hx-vals` attribute
- Can access the `event` object in event handlers

**Security:** Using `javascript:` prefix introduces XSS risks with user-generated content.

#### hx-include Attribute

Include values from other elements using CSS selectors:

```html
<div hx-get="/search" hx-include="[name='filter']">
    Search
</div>
```

#### hx-params Attribute

Filter which parameters are sent:

```html
<!-- Include only specific params -->
<div hx-post="/update" hx-params="username,email">

<!-- Exclude specific params -->
<div hx-post="/update" hx-params="* -password">

<!-- Send no params -->
<div hx-post="/update" hx-params="none">
```

## Request Headers

htmx automatically adds custom headers to every AJAX request for server-side detection and context:

| Header | Description | Value |
|--------|-------------|-------|
| `HX-Request` | Identifies htmx request | Always `"true"` (except history restore if disabled) |
| `HX-Trigger` | ID of triggered element | Element's `id` attribute if it exists |
| `HX-Trigger-Name` | Name of triggered element | Element's `name` attribute if it exists |
| `HX-Target` | ID of target element | Target element's `id` if it exists |
| `HX-Current-URL` | Current browser URL | Full URL string |
| `HX-Boosted` | Indicates boosted request | `"true"` if using `hx-boost` |
| `HX-History-Restore-Request` | History restoration | `"true"` for local history cache misses |
| `HX-Prompt` | User response to prompt | Value from `hx-prompt` dialog |

**Server Usage:**
```go
// Detect htmx requests
if r.Header.Get("HX-Request") == "true" {
    // Return HTML fragment
} else {
    // Return full page
}
```

### Request Configuration (hx-request)

Configure request behavior with JSON-like syntax:

```html
<div hx-get="/data" hx-request='{"timeout":100}'>
    Fast Timeout Request
</div>

<div hx-get="/data" hx-request='{"credentials":true}'>
    Send Credentials
</div>

<div hx-get="/data" hx-request='{"noHeaders":true}'>
    Strip All Headers
</div>

<!-- Dynamic evaluation -->
<div hx-get="/data" hx-request='js:{timeout:getTimeoutSetting()}'>
    Dynamic Timeout
</div>
```

**Options:**
- `timeout`: Request timeout in milliseconds
- `credentials`: Whether to send credentials (cookies, authorization headers)
- `noHeaders`: Removes all headers from the request

## Response Data Transfer

### Primary Content Type: text/html

**htmx expects HTML responses by default.** The server should return HTML fragments that will be swapped into the DOM, not JSON data.

**Example server response:**
```html
<div class="user-profile">
    <h2>John Doe</h2>
    <p>Email: john@example.com</p>
</div>
```

This HTML is swapped into the target element specified by `hx-target` (or the triggering element by default).

### Response Headers

htmx recognizes special response headers for client-side control:

| Header | Description | Example |
|--------|-------------|---------|
| `HX-Location` | Client-side redirect (no full reload) | `{"path":"/new-page","target":"#main"}` |
| `HX-Push-Url` | Push URL into browser history | `/new-url` or `false` |
| `HX-Redirect` | Full page redirect | `/login` |
| `HX-Refresh` | Trigger full page refresh | `true` |
| `HX-Replace-Url` | Replace current URL in history | `/current-state` |
| `HX-Reswap` | Override swap method | `innerHTML`, `outerHTML`, `beforeend` |
| `HX-Retarget` | Change swap target | `#different-element` |
| `HX-Reselect` | Select part of response | `.content` |
| `HX-Trigger` | Trigger client-side events | `{"showMessage":"Data saved"}` (JSON) |
| `HX-Trigger-After-Settle` | Events after settle phase | `{"updateChart":null}` |
| `HX-Trigger-After-Swap` | Events after swap phase | `{"highlight":null}` |

**Important:** These headers are NOT processed with 3xx redirect responses (e.g., HTTP 302). The browser intercepts redirects and returns headers from the redirected URL. Use status 200 when you need htmx to process response headers.

### HX-Trigger Headers (Event Triggering)

Trigger custom JavaScript events from server responses:

**Simple event:**
```
HX-Trigger: myEvent
```

**Event with JSON payload:**
```
HX-Trigger: {"showMessage": {"level":"info", "text":"Data saved"}}
```

**Multiple events:**
```
HX-Trigger: {"event1": null, "event2": {"data": "value"}}
```

**Timing variants:**
- `HX-Trigger`: Fires immediately after receiving response
- `HX-Trigger-After-Swap`: Fires after content is swapped into DOM
- `HX-Trigger-After-Settle`: Fires after settle phase (animations, focus)

### Swap Strategies

The `hx-swap` attribute controls how response HTML is inserted:

| Strategy | Description |
|----------|-------------|
| `innerHTML` | Replace inner HTML (default) |
| `outerHTML` | Replace entire element including itself |
| `beforebegin` | Insert before element |
| `afterbegin` | Insert as first child |
| `beforeend` | Insert as last child |
| `afterend` | Insert after element |
| `delete` | Delete target element |
| `none` | Do not swap (still processes OOB swaps and headers) |

**Advanced options:**
```html
<div hx-swap="innerHTML swap:200ms settle:400ms">
    <!-- 200ms delay before swap, 400ms settle time -->
</div>

<div hx-swap="innerHTML transition:true">
    <!-- Use View Transitions API -->
</div>
```

## Request Lifecycle

### Order of Operations

1. **Element triggered** → Request begins
2. **Values gathered** from element and associated form
3. **`htmx-request` class** applied to element
4. **AJAX request issued** asynchronously
5. **Response received**
6. **`htmx-swapping` class** applied to target
7. **Optional swap delay** (configured via `hx-swap`)
8. **Content swapped** into DOM
   - `htmx-swapping` class removed
   - `htmx-added` class added to new content
   - `htmx-settling` class applied to target
9. **Settle delay** (default 20ms)
10. **DOM settled** (animations, focus)
    - `htmx-settling` class removed
    - `htmx-added` class removed

### CSS Classes for Styling

htmx applies CSS classes during request lifecycle:

| Class | Applied To | When | Purpose |
|-------|-----------|------|---------|
| `htmx-request` | Triggering element or `hx-indicator` target | During request | Show loading state |
| `htmx-indicator` | Elements with this class | Auto-shown when `htmx-request` present | Loading indicators (opacity:1) |
| `htmx-swapping` | Target element | Before content swap | Transition animations |
| `htmx-settling` | Target element | After swap, before settle | Settlement animations |
| `htmx-added` | New content | During settle phase | Animate new content |

**Example:**
```css
.htmx-request .spinner {
    display: inline-block;
}
.htmx-swapping {
    opacity: 0;
    transition: opacity 200ms;
}
.htmx-settling {
    opacity: 1;
}
```

## Key Differences from Traditional AJAX

### HTML vs JSON

| Aspect | Traditional AJAX | htmx |
|--------|------------------|------|
| **Response format** | JSON data | HTML fragments |
| **Client processing** | Parse JSON, update DOM with JavaScript | Direct DOM swap |
| **Server responsibility** | Return data | Return presentation (HTML) |
| **Architecture** | Separate frontend/backend | Hypermedia-driven |

### No Post/Redirect/Get Pattern

With htmx, you don't need the Post/Redirect/Get pattern. After processing a POST:
- ❌ Don't return HTTP 302 redirect
- ✅ Return the new HTML fragment directly (200 OK)
- ✅ Optionally use `HX-Push-Url` header to update browser URL

## HTTP Methods

htmx supports all standard HTTP verbs:

| Attribute | HTTP Method |
|-----------|-------------|
| `hx-get` | GET |
| `hx-post` | POST |
| `hx-put` | PUT |
| `hx-patch` | PATCH |
| `hx-delete` | DELETE |

**Example:**
```html
<button hx-delete="/users/123" hx-confirm="Delete this user?">
    Delete User
</button>
```

## Progressive Enhancement (hx-boost)

The `hx-boost` attribute progressively enhances normal links and forms:

```html
<div hx-boost="true">
    <a href="/about">About</a>  <!-- AJAXified GET -->
    <form action="/search">     <!-- AJAXified POST -->
        <input name="q">
        <button>Search</button>
    </form>
</div>
```

- Converts regular links to AJAX requests
- Swaps response into `<body>` element
- Updates browser URL automatically
- Falls back to normal behavior if JavaScript disabled

## Event System

htmx emits events throughout the request lifecycle for custom JavaScript integration:

**Key events:**
- `htmx:configRequest` - Modify request before sending
- `htmx:beforeRequest` - Cancel or modify request
- `htmx:afterRequest` - Access response
- `htmx:xhr:progress` - File upload progress
- `htmx:beforeSwap` - Modify swap behavior
- `htmx:afterSwap` - DOM updated
- `htmx:beforeSettle` - Before settle phase
- `htmx:afterSettle` - After settle complete

**Example:**
```javascript
document.body.addEventListener('htmx:configRequest', function(evt) {
    evt.detail.parameters['csrf_token'] = getCsrfToken();
});
```

## Summary: Content Types and Data Flow

### Client → Server

**Content-Type:**
- Default: `application/x-www-form-urlencoded`
- File uploads: `multipart/form-data` (via `hx-encoding`)

**Data format:**
- Form-encoded key-value pairs
- Or multipart MIME for files

**Custom headers:**
- `HX-Request: true` (and 7 other context headers)

### Server → Client

**Content-Type:**
- Primary: `text/html` (HTML fragments)
- Server should return ready-to-render HTML

**Special headers:**
- 11 `HX-*` headers for client-side control
- `HX-Trigger` variants for custom events

**Status codes:**
- Use 200 OK for successful responses with HX headers
- Avoid 3xx redirects (browser intercepts HX headers)
- Use `HX-Redirect` header for client-side redirects

## Server-Side Implementation Notes

### Detecting htmx Requests

```go
func handler(w http.ResponseWriter, r *http.Request) {
    if r.Header.Get("HX-Request") == "true" {
        // Return HTML fragment
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        fmt.Fprintf(w, `<div class="result">Success!</div>`)
    } else {
        // Return full page
        renderFullPage(w, r)
    }
}
```

### Setting Response Headers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("HX-Trigger", `{"showNotification": {"message": "Saved"}}`)
    w.Header().Set("HX-Push-Url", "/items/123")
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, `<div>Item saved</div>`)
}
```

### File Upload Handling

```go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form (32MB max memory)
    err := r.ParseMultipartForm(32 << 20)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()
    
    // Process file...
    
    // Return HTML response
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprintf(w, `<div class="success">Uploaded: %s</div>`, header.Filename)
}
```

## References

- [htmx Documentation](https://htmx.org/docs/)
- [htmx Reference](https://htmx.org/reference/)
- [Request Headers Reference](https://htmx.org/reference/#request_headers)
- [Response Headers Reference](https://htmx.org/reference/#response_headers)
- [File Upload Example](https://htmx.org/examples/file-upload/)
- [hx-vals Attribute](https://htmx.org/attributes/hx-vals/)
- [hx-encoding Attribute](https://htmx.org/attributes/hx-encoding/)
- [hx-request Attribute](https://htmx.org/attributes/hx-request/)
