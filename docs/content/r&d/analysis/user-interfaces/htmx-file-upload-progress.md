---
title: "HTMX File Upload Progress Bar Analysis"
type: docs
linkTitle: "HTMX File Upload Progress"
weight: 20
description: >
  Analysis of implementing real-time file upload progress bars using HTMX's event system and the xhr:progress event.
---

## Overview

HTMX provides built-in support for displaying file upload progress through the `htmx:xhr:progress` event. This event fires periodically during file uploads, providing real-time feedback about upload progress. Unlike traditional approaches that require polling the server for progress updates, HTMX leverages the browser's native `XMLHttpRequest` progress events to deliver a smooth, client-side progress tracking experience.

**Key Advantage:** No server-side session storage or polling required—progress is tracked entirely through the browser's native upload progress events.

## Core Architecture

### The htmx:xhr:progress Event

When HTMX performs a request that supports progress tracking (primarily file uploads using `multipart/form-data`), it fires the `htmx:xhr:progress` event periodically based on the browser's standard `ProgressEvent`.

**Event Details:**
- **Event name:** `htmx:xhr:progress`
- **Frequency:** Fired multiple times during upload at intervals determined by the browser
- **Event detail properties:**
  - `loaded`: Number of bytes uploaded so far
  - `total`: Total number of bytes to upload
  - `lengthComputable`: Boolean indicating if total size is known

**Important:** The event listener attaches to `xhr.upload`, not the main `xhr` object, which is crucial for receiving actual upload progress events rather than download/response progress.

## Implementation Approaches

### 1. Pure JavaScript Implementation

The most straightforward approach uses vanilla JavaScript to listen for the progress event and update a progress element.

**HTML Structure:**
```html
<form id='form' hx-encoding='multipart/form-data' hx-post='/upload'>
  <input type='file' name='file'>
  <button type='submit'>Upload</button>
  <progress id='progress' value='0' max='100'></progress>
</form>

<script>
  htmx.on('#form', 'htmx:xhr:progress', function(evt) {
    htmx.find('#progress').setAttribute('value',
      evt.detail.loaded / evt.detail.total * 100
    );
  });
</script>
```

**Key Components:**
- `hx-encoding='multipart/form-data'` - Required for file uploads; tells HTMX to use FormData API
- `htmx.on('#form', 'htmx:xhr:progress', callback)` - Event listener scoped to the form
- `evt.detail.loaded` and `evt.detail.total` - Byte counts for progress calculation
- Progress element with `max='100'` for percentage-based display

**Event Listener Scope:**
You can scope the listener to a specific form (as shown) or listen globally:

```javascript
// Global listener (all forms)
htmx.on('htmx:xhr:progress', function(evt) {
  const progressBar = evt.target.querySelector('progress');
  if (progressBar) {
    progressBar.setAttribute('value',
      evt.detail.loaded / evt.detail.total * 100
    );
  }
});
```

### 2. Hyperscript Alternative

Hyperscript (a companion library to HTMX) provides a more concise, declarative syntax by embedding event handling directly in HTML attributes.

**HTML Structure:**
```html
<form hx-encoding='multipart/form-data'
      hx-post='/upload'
      _='on htmx:xhr:progress(loaded, total)
         set #progress.value to (loaded/total)*100'>
  <input type='file' name='file'>
  <button type='submit'>Upload</button>
  <progress id='progress' value='0' max='100'></progress>
</form>
```

**Advantages:**
- **Destructuring:** The `(loaded, total)` syntax automatically extracts properties from `evt.detail`
- **Co-location:** Event handling lives with the element, improving maintainability
- **Conciseness:** Less boilerplate than JavaScript
- **No separate script tags:** Everything defined in HTML

**When to Use Hyperscript:**
- You're already using Hyperscript in your project
- You prefer declarative over imperative code
- You want to reduce JavaScript file size
- Event handling logic is simple and doesn't require complex computation

### 3. Enhanced Progress Display

For a better user experience, combine the progress bar with additional status information:

```html
<form id='upload-form'
      hx-encoding='multipart/form-data'
      hx-post='/upload'>
  <input type='file' name='file' id='file-input'>
  <button type='submit'>Upload</button>

  <!-- Progress bar with ARIA attributes for accessibility -->
  <div class="progress-container">
    <progress id='progress'
              value='0'
              max='100'
              role="progressbar"
              aria-valuemin="0"
              aria-valuemax="100"
              aria-valuenow="0"></progress>
    <span id='progress-text'>0%</span>
  </div>
</form>

<script>
  htmx.on('#upload-form', 'htmx:xhr:progress', function(evt) {
    const percent = Math.round(evt.detail.loaded / evt.detail.total * 100);
    const progressBar = htmx.find('#progress');
    const progressText = htmx.find('#progress-text');

    // Update progress bar
    progressBar.setAttribute('value', percent);
    progressBar.setAttribute('aria-valuenow', percent);

    // Update text display
    progressText.textContent = percent + '%';
  });

  // Reset progress on new file selection
  htmx.on('#file-input', 'change', function() {
    htmx.find('#progress').setAttribute('value', 0);
    htmx.find('#progress-text').textContent = '0%';
  });
</script>
```

**Accessibility Features:**
- `role="progressbar"` - Identifies element as progress indicator
- `aria-valuemin`, `aria-valuemax` - Define progress range
- `aria-valuenow` - Current progress value (updated dynamically)
- Text display - Provides visual percentage for screen reader users

## Advanced Techniques

### Smooth Progress Animation

Add CSS transitions for smoother visual progression:

```html
<style>
  .progress-bar {
    width: 100%;
    height: 20px;
    background-color: #f0f0f0;
    border-radius: 10px;
    overflow: hidden;
    position: relative;
  }

  .progress-fill {
    height: 100%;
    background-color: #337ab7;
    width: 0%;
    transition: width 0.3s ease;
    position: relative;
  }

  .progress-fill::after {
    content: attr(data-percent);
    position: absolute;
    right: 10px;
    color: white;
    font-weight: bold;
  }
</style>

<form id='upload-form' hx-encoding='multipart/form-data' hx-post='/upload'>
  <input type='file' name='file'>
  <button type='submit'>Upload</button>

  <div class="progress-bar">
    <div id="progress-fill" class="progress-fill" data-percent="0%"></div>
  </div>
</form>

<script>
  htmx.on('#upload-form', 'htmx:xhr:progress', function(evt) {
    const percent = Math.round(evt.detail.loaded / evt.detail.total * 100);
    const fill = htmx.find('#progress-fill');

    fill.style.width = percent + '%';
    fill.setAttribute('data-percent', percent + '%');
  });
</script>
```

**CSS Key Points:**
- `transition: width 0.3s ease` - Smooths out progress bar jumps
- Custom styling provides better visual design than native `<progress>` element
- `::after` pseudo-element displays percentage without additional HTML

### File Size Display

Show uploaded bytes alongside percentage:

```javascript
htmx.on('#upload-form', 'htmx:xhr:progress', function(evt) {
  const percent = Math.round(evt.detail.loaded / evt.detail.total * 100);
  const loadedMB = (evt.detail.loaded / 1024 / 1024).toFixed(2);
  const totalMB = (evt.detail.total / 1024 / 1024).toFixed(2);

  htmx.find('#progress').setAttribute('value', percent);
  htmx.find('#status').textContent =
    `${loadedMB} MB / ${totalMB} MB (${percent}%)`;
});
```

### Upload Speed Calculation

Track upload speed in real-time:

```javascript
let lastLoaded = 0;
let lastTime = Date.now();

htmx.on('#upload-form', 'htmx:xhr:progress', function(evt) {
  const now = Date.now();
  const timeDiff = (now - lastTime) / 1000; // seconds
  const bytesDiff = evt.detail.loaded - lastLoaded;

  if (timeDiff > 0) {
    const speed = bytesDiff / timeDiff; // bytes per second
    const speedMBps = (speed / 1024 / 1024).toFixed(2);

    htmx.find('#speed').textContent = `${speedMBps} MB/s`;

    // Calculate estimated time remaining
    const remaining = evt.detail.total - evt.detail.loaded;
    const secondsLeft = Math.round(remaining / speed);
    htmx.find('#eta').textContent = `${secondsLeft}s remaining`;
  }

  lastLoaded = evt.detail.loaded;
  lastTime = now;

  // Update progress bar
  const percent = Math.round(evt.detail.loaded / evt.detail.total * 100);
  htmx.find('#progress').setAttribute('value', percent);
});

// Reset on form submit
htmx.on('#upload-form', 'htmx:beforeRequest', function() {
  lastLoaded = 0;
  lastTime = Date.now();
});
```

### Multiple File Uploads

Handle progress for multiple files:

```html
<form id='multi-upload' hx-encoding='multipart/form-data' hx-post='/upload'>
  <input type='file' name='files' multiple>
  <button type='submit'>Upload All</button>

  <div id="overall-progress">
    <h4>Overall Progress</h4>
    <progress id='progress' value='0' max='100'></progress>
    <span id='progress-text'>0%</span>
  </div>
</form>

<script>
  htmx.on('#multi-upload', 'htmx:xhr:progress', function(evt) {
    const percent = Math.round(evt.detail.loaded / evt.detail.total * 100);
    htmx.find('#progress').setAttribute('value', percent);
    htmx.find('#progress-text').textContent = percent + '%';
  });
</script>
```

**Note:** The `htmx:xhr:progress` event reports progress for the entire multipart request (all files combined), not individual file progress. For per-file progress, you'd need to upload files individually with separate requests.

## Server-Side Polling Pattern (Alternative)

While the `htmx:xhr:progress` event is ideal for upload progress, some scenarios require server-side progress tracking (e.g., long-running processing tasks after upload). HTMX supports this with polling.

**Use Case:** Upload completes quickly, but server-side processing (virus scanning, transcoding) takes time.

**HTML Structure:**
```html
<!-- Initial state: upload form -->
<form hx-post='/upload'
      hx-encoding='multipart/form-data'
      hx-target='#upload-container'>
  <input type='file' name='file'>
  <button type='submit'>Upload</button>
</form>

<!-- Server returns this after upload starts -->
<div id='upload-container'
     hx-get='/job/progress/{{jobId}}'
     hx-trigger='every 600ms'
     hx-swap='outerHTML'>
  <h3>Processing file...</h3>
  <progress id='progress' value='{{percent}}' max='100'></progress>
  <p>{{percent}}% complete</p>
</div>

<!-- Server returns this when job completes -->
<div id='upload-container' hx-trigger='none'>
  <h3>✓ Upload complete!</h3>
  <p>File processed successfully.</p>
  <button hx-get='/upload-form' hx-target='#upload-container'>
    Upload Another
  </button>
</div>
```

**Server Response Flow:**
1. **Initial POST /upload** → Server starts background job, returns progress polling HTML
2. **Repeated GET /job/progress/:id** → Server returns updated progress HTML every 600ms
3. **Final GET when complete** → Server returns completion HTML with `hx-trigger='none'` to stop polling

**Key Attributes:**
- `hx-trigger='every 600ms'` - Poll server every 600 milliseconds
- `hx-swap='outerHTML'` - Replace entire container (including trigger) with response
- `hx-trigger='none'` - Stop polling when job completes

**When to Use Polling vs xhr:progress:**
- **Upload progress:** Use `htmx:xhr:progress` (client-side, no server load)
- **Processing progress:** Use polling (server tracks job state)
- **Hybrid:** Use `xhr:progress` during upload, switch to polling for processing

## Server-Side Implementation

### Go Handler Example

```go
package endpoint

import (
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"

    "github.com/z5labs/humus"
    "github.com/z5labs/humus/rest"
)

func RegisterFileUpload(api *rest.Api) {
    h := &fileUploadHandler{
        log: humus.Logger("file-upload"),
    }
    err := api.Route(http.MethodPost, "/upload", h)
    if err != nil {
        panic(err)
    }
}

type fileUploadHandler struct {
    log *slog.Logger
}

func (h *fileUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form (32MB max memory)
    err := r.ParseMultipartForm(32 << 20)
    if err != nil {
        h.log.Error("failed to parse multipart form", "error", err)
        http.Error(w, "Failed to parse upload", http.StatusBadRequest)
        return
    }

    // Get the file from form data
    file, header, err := r.FormFile("file")
    if err != nil {
        h.log.Error("failed to get file from form", "error", err)
        http.Error(w, "No file provided", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Create destination file
    destPath := filepath.Join("/uploads", header.Filename)
    dest, err := os.Create(destPath)
    if err != nil {
        h.log.Error("failed to create destination file", "error", err)
        http.Error(w, "Failed to save file", http.StatusInternalServerError)
        return
    }
    defer dest.Close()

    // Copy uploaded file to destination
    bytesWritten, err := io.Copy(dest, file)
    if err != nil {
        h.log.Error("failed to copy file", "error", err)
        http.Error(w, "Failed to save file", http.StatusInternalServerError)
        return
    }

    h.log.Info("file uploaded successfully",
        "filename", header.Filename,
        "size", bytesWritten)

    // Return HTML fragment (HTMX expects HTML, not JSON)
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprintf(w, `
        <div class="success">
            <h3>✓ Upload Complete</h3>
            <p>Uploaded: %s (%d bytes)</p>
        </div>
    `, header.Filename, bytesWritten)
}
```

**Key Points:**
- `ParseMultipartForm(32 << 20)` - Parses form with 32MB memory limit
- `FormFile("file")` - Retrieves file by input name attribute
- Returns HTML fragment, not JSON (HTMX expects HTML responses)
- Proper error handling with user-friendly messages

### Validation and Security

Add file validation before processing:

```go
func (h *fileUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    err := r.ParseMultipartForm(32 << 20)
    if err != nil {
        h.renderError(w, "File too large (max 32MB)")
        return
    }

    file, header, err := r.FormFile("file")
    if err != nil {
        h.renderError(w, "No file provided")
        return
    }
    defer file.Close()

    // Validate file size
    if header.Size > 10*1024*1024 { // 10MB limit
        h.renderError(w, "File exceeds 10MB limit")
        return
    }

    // Validate file type by extension
    ext := filepath.Ext(header.Filename)
    allowedExts := map[string]bool{
        ".jpg": true, ".jpeg": true, ".png": true,
        ".gif": true, ".pdf": true,
    }
    if !allowedExts[ext] {
        h.renderError(w, "Invalid file type. Allowed: JPG, PNG, GIF, PDF")
        return
    }

    // Validate MIME type (more reliable than extension)
    buffer := make([]byte, 512)
    _, err = file.Read(buffer)
    if err != nil {
        h.renderError(w, "Failed to read file")
        return
    }
    file.Seek(0, 0) // Reset file pointer

    mimeType := http.DetectContentType(buffer)
    allowedMimes := map[string]bool{
        "image/jpeg": true, "image/png": true,
        "image/gif": true, "application/pdf": true,
    }
    if !allowedMimes[mimeType] {
        h.renderError(w, fmt.Sprintf("Invalid file type: %s", mimeType))
        return
    }

    // Generate safe filename (prevent directory traversal)
    safeFilename := filepath.Base(header.Filename)
    timestamp := time.Now().Unix()
    destFilename := fmt.Sprintf("%d_%s", timestamp, safeFilename)
    destPath := filepath.Join("/uploads", destFilename)

    // Save file...
    // (rest of upload logic)
}

func (h *fileUploadHandler) renderError(w http.ResponseWriter, message string) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusBadRequest)
    fmt.Fprintf(w, `
        <div class="error">
            <h3>✗ Upload Failed</h3>
            <p>%s</p>
        </div>
    `, message)
}
```

**Security Measures:**
- **Size limits:** Prevent DoS through large uploads
- **Extension validation:** Basic file type filtering
- **MIME type detection:** More reliable than extension alone
- **Filename sanitization:** Prevent directory traversal attacks (`../../../etc/passwd`)
- **Timestamp prefix:** Prevent filename collisions

## Common Pitfalls and Solutions

### 1. Progress Event Not Firing

**Problem:** The `htmx:xhr:progress` event doesn't fire during upload.

**Causes:**
- Missing `hx-encoding='multipart/form-data'` attribute
- File input not inside the form with HTMX attributes
- Server response too fast (localhost testing with small files)

**Solution:**
```html
<!-- ✗ Wrong: Missing encoding -->
<form hx-post='/upload'>
  <input type='file' name='file'>
</form>

<!-- ✓ Correct: Encoding specified -->
<form hx-encoding='multipart/form-data' hx-post='/upload'>
  <input type='file' name='file'>
</form>
```

**Testing Tip:** Use large files or network throttling in browser DevTools to see progress events on localhost.

### 2. Event Properties Undefined

**Problem:** `evt.detail.loaded` and `evt.detail.total` are `undefined`.

**Cause:** Older HTMX versions had a bug where standard DOM event properties weren't properly merged into `evt.detail`.

**Solution:** Upgrade to HTMX 1.3.0 or later, where this was fixed.

**Workaround for older versions:**
```javascript
htmx.on('#form', 'htmx:xhr:progress', function(evt) {
  // Access properties directly on event (not evt.detail)
  const loaded = evt.loaded || evt.detail.loaded;
  const total = evt.total || evt.detail.total;

  const percent = (loaded / total) * 100;
  htmx.find('#progress').setAttribute('value', percent);
});
```

### 3. Progress Bar Doesn't Reset

**Problem:** After upload completes, progress bar stays at 100% for next upload.

**Solution:** Reset progress when user selects a new file:

```javascript
// Reset on file input change
htmx.on('#file-input', 'change', function() {
  htmx.find('#progress').setAttribute('value', 0);
});

// Or reset before request starts
htmx.on('#form', 'htmx:beforeRequest', function() {
  htmx.find('#progress').setAttribute('value', 0);
});
```

### 4. Progress Updates Are Jumpy

**Problem:** Progress bar jumps between values instead of smooth progression.

**Solution:** Add CSS transitions:

```css
progress {
  transition: value 0.3s ease;
}

/* For custom progress bars */
.progress-fill {
  transition: width 0.3s ease;
}
```

**Note:** Native `<progress>` elements have limited transition support in some browsers. Custom div-based progress bars offer better control.

### 5. Multiple Progress Bars Conflict

**Problem:** Multiple upload forms on the same page interfere with each other's progress bars.

**Solution:** Scope event listeners and use relative selectors:

```javascript
// ✗ Wrong: Global IDs conflict
htmx.on('htmx:xhr:progress', function(evt) {
  htmx.find('#progress').setAttribute('value', percent);
});

// ✓ Correct: Scoped to specific form
htmx.on('#form-1', 'htmx:xhr:progress', function(evt) {
  const progressBar = this.querySelector('#progress');
  progressBar.setAttribute('value', percent);
});

// ✓ Better: Use relative selectors
htmx.on('htmx:xhr:progress', function(evt) {
  const form = evt.target;
  const progressBar = form.querySelector('progress');
  if (progressBar) {
    progressBar.setAttribute('value', percent);
  }
});
```

### 6. Progress Completes But Upload Fails

**Problem:** Progress bar reaches 100%, but server returns error.

**Solution:** Handle the `htmx:afterRequest` event to check response status:

```javascript
htmx.on('#form', 'htmx:afterRequest', function(evt) {
  if (!evt.detail.successful) {
    // Upload failed
    htmx.find('#progress').setAttribute('value', 0);
    htmx.find('#status').textContent = 'Upload failed. Please try again.';
    htmx.find('#status').className = 'error';
  } else {
    // Upload succeeded
    htmx.find('#status').textContent = 'Upload complete!';
    htmx.find('#status').className = 'success';
  }
});
```

## Best Practices

### 1. Always Include Accessibility Attributes

```html
<progress id='progress'
          value='0'
          max='100'
          role="progressbar"
          aria-valuemin="0"
          aria-valuemax="100"
          aria-valuenow="0"
          aria-label="Upload progress"></progress>
```

### 2. Provide Multiple Forms of Feedback

Don't rely solely on visual progress bars. Include:
- **Percentage text:** Screen reader compatible
- **File size indicators:** Shows actual bytes uploaded
- **Upload speed:** Helps users estimate completion time
- **Status messages:** "Uploading...", "Complete!", "Failed"

### 3. Disable Submit During Upload

Prevent duplicate uploads:

```javascript
htmx.on('#form', 'htmx:beforeRequest', function() {
  const btn = htmx.find('#submit-btn');
  btn.disabled = true;
  btn.textContent = 'Uploading...';
});

htmx.on('#form', 'htmx:afterRequest', function() {
  const btn = htmx.find('#submit-btn');
  btn.disabled = false;
  btn.textContent = 'Upload';
});
```

### 4. Validate on Client and Server

```html
<form id='form' hx-encoding='multipart/form-data' hx-post='/upload'>
  <input type='file'
         name='file'
         accept='.jpg,.jpeg,.png,.gif,.pdf'
         required>
  <button type='submit'>Upload</button>
</form>

<script>
  // Client-side validation before upload
  htmx.on('#form', 'htmx:configRequest', function(evt) {
    const fileInput = htmx.find('input[type=file]');
    const file = fileInput.files[0];

    if (!file) {
      evt.preventDefault();
      alert('Please select a file');
      return false;
    }

    if (file.size > 10 * 1024 * 1024) {
      evt.preventDefault();
      alert('File must be less than 10MB');
      return false;
    }
  });
</script>
```

**Important:** Always validate on the server too—client-side validation can be bypassed.

### 5. Handle Network Errors Gracefully

```javascript
htmx.on('#form', 'htmx:responseError', function(evt) {
  htmx.find('#status').textContent =
    'Network error. Please check your connection and try again.';
  htmx.find('#progress').setAttribute('value', 0);
});

htmx.on('#form', 'htmx:sendError', function(evt) {
  htmx.find('#status').textContent =
    'Failed to send request. Please try again.';
  htmx.find('#progress').setAttribute('value', 0);
});
```

### 6. Use Smooth UI Transitions

Keep the same element ID across responses to enable smooth settling:

```html
<!-- Initial state -->
<div id="upload-area">
  <form hx-post='/upload' hx-encoding='multipart/form-data' hx-target='#upload-area'>
    <input type='file' name='file'>
    <button type='submit'>Upload</button>
  </form>
</div>

<!-- Server response after upload -->
<div id="upload-area">
  <div class="success">
    <h3>✓ Upload Complete</h3>
    <p>File uploaded successfully</p>
  </div>
</div>
```

**Why this matters:** HTMX uses IDs to smoothly settle style changes. Reusing the same ID enables CSS transitions between states.

## Complete Working Example

Here's a production-ready implementation combining all best practices:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>File Upload with Progress</title>
  <script src="https://unpkg.com/htmx.org@1.9.10"></script>
  <style>
    .upload-container {
      max-width: 500px;
      margin: 50px auto;
      padding: 20px;
      border: 1px solid #ddd;
      border-radius: 8px;
      font-family: system-ui, -apple-system, sans-serif;
    }

    .form-group {
      margin-bottom: 15px;
    }

    label {
      display: block;
      margin-bottom: 5px;
      font-weight: 600;
    }

    input[type="file"] {
      width: 100%;
      padding: 8px;
      border: 1px solid #ddd;
      border-radius: 4px;
    }

    button {
      width: 100%;
      padding: 10px;
      background-color: #337ab7;
      color: white;
      border: none;
      border-radius: 4px;
      font-size: 16px;
      cursor: pointer;
    }

    button:hover:not(:disabled) {
      background-color: #286090;
    }

    button:disabled {
      background-color: #ccc;
      cursor: not-allowed;
    }

    .progress-container {
      margin-top: 20px;
      display: none;
    }

    .progress-container.active {
      display: block;
    }

    .progress-bar {
      width: 100%;
      height: 30px;
      background-color: #f0f0f0;
      border-radius: 15px;
      overflow: hidden;
      position: relative;
      margin-bottom: 10px;
    }

    .progress-fill {
      height: 100%;
      background: linear-gradient(90deg, #337ab7, #5bc0de);
      width: 0%;
      transition: width 0.3s ease;
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-weight: bold;
    }

    .status {
      text-align: center;
      padding: 10px;
      margin-top: 15px;
      border-radius: 4px;
      display: none;
    }

    .status.active {
      display: block;
    }

    .status.success {
      background-color: #dff0d8;
      color: #3c763d;
      border: 1px solid #d6e9c6;
    }

    .status.error {
      background-color: #f2dede;
      color: #a94442;
      border: 1px solid #ebccd1;
    }

    .status.info {
      background-color: #d9edf7;
      color: #31708f;
      border: 1px solid #bce8f1;
    }
  </style>
</head>
<body>

<div class="upload-container">
  <h2>File Upload</h2>

  <form id="upload-form"
        hx-encoding="multipart/form-data"
        hx-post="/upload"
        hx-target="#status"
        hx-swap="innerHTML">

    <div class="form-group">
      <label for="file-input">Choose file (max 10MB)</label>
      <input type="file"
             id="file-input"
             name="file"
             accept=".jpg,.jpeg,.png,.gif,.pdf"
             required>
    </div>

    <button type="submit" id="submit-btn">Upload File</button>
  </form>

  <div id="progress-container" class="progress-container">
    <div class="progress-bar">
      <div id="progress-fill" class="progress-fill"></div>
    </div>
    <div id="progress-details" style="text-align: center; color: #666;">
      <div id="progress-text">0%</div>
      <div id="speed-text"></div>
    </div>
  </div>

  <div id="status" class="status"></div>
</div>

<script>
  // Progress tracking variables
  let lastLoaded = 0;
  let lastTime = Date.now();

  // Progress event handler
  htmx.on('#upload-form', 'htmx:xhr:progress', function(evt) {
    const percent = Math.round(evt.detail.loaded / evt.detail.total * 100);
    const loadedMB = (evt.detail.loaded / 1024 / 1024).toFixed(2);
    const totalMB = (evt.detail.total / 1024 / 1024).toFixed(2);

    // Update progress bar
    const fill = document.getElementById('progress-fill');
    fill.style.width = percent + '%';
    fill.textContent = percent + '%';
    fill.setAttribute('aria-valuenow', percent);

    // Update progress text
    document.getElementById('progress-text').textContent =
      `${loadedMB} MB / ${totalMB} MB`;

    // Calculate upload speed
    const now = Date.now();
    const timeDiff = (now - lastTime) / 1000;
    if (timeDiff > 0.5) { // Update every 0.5 seconds
      const bytesDiff = evt.detail.loaded - lastLoaded;
      const speed = bytesDiff / timeDiff;
      const speedMBps = (speed / 1024 / 1024).toFixed(2);

      document.getElementById('speed-text').textContent =
        `Upload speed: ${speedMBps} MB/s`;

      lastLoaded = evt.detail.loaded;
      lastTime = now;
    }
  });

  // Before request: validate and show progress
  htmx.on('#upload-form', 'htmx:beforeRequest', function(evt) {
    const fileInput = document.getElementById('file-input');
    const file = fileInput.files[0];

    // Validate file
    if (!file) {
      evt.preventDefault();
      showStatus('Please select a file', 'error');
      return false;
    }

    if (file.size > 10 * 1024 * 1024) {
      evt.preventDefault();
      showStatus('File must be less than 10MB', 'error');
      return false;
    }

    // Reset progress tracking
    lastLoaded = 0;
    lastTime = Date.now();

    // Show progress container
    document.getElementById('progress-container').classList.add('active');
    document.getElementById('progress-fill').style.width = '0%';

    // Disable submit button
    const btn = document.getElementById('submit-btn');
    btn.disabled = true;
    btn.textContent = 'Uploading...';

    // Hide previous status
    hideStatus();
  });

  // After request: re-enable form
  htmx.on('#upload-form', 'htmx:afterRequest', function(evt) {
    const btn = document.getElementById('submit-btn');
    btn.disabled = false;
    btn.textContent = 'Upload File';

    if (evt.detail.successful) {
      // Success: server response will be swapped into #status
      document.getElementById('progress-container').classList.remove('active');
      document.getElementById('file-input').value = ''; // Reset file input
    } else {
      // Error
      showStatus('Upload failed. Please try again.', 'error');
      document.getElementById('progress-container').classList.remove('active');
    }
  });

  // Network errors
  htmx.on('#upload-form', 'htmx:responseError', function(evt) {
    showStatus('Network error. Please check your connection.', 'error');
  });

  htmx.on('#upload-form', 'htmx:sendError', function(evt) {
    showStatus('Failed to send request. Please try again.', 'error');
  });

  // Helper functions
  function showStatus(message, type) {
    const status = document.getElementById('status');
    status.textContent = message;
    status.className = 'status active ' + type;
  }

  function hideStatus() {
    const status = document.getElementById('status');
    status.className = 'status';
  }
</script>

</body>
</html>
```

**Server response example (Go):**
```go
// Success response
w.Header().Set("Content-Type", "text/html; charset=utf-8")
fmt.Fprintf(w, `
  <div class="success">
    ✓ Upload complete! File: %s (%d bytes)
  </div>
`, filename, size)

// Error response
w.Header().Set("Content-Type", "text/html; charset=utf-8")
w.WriteHeader(http.StatusBadRequest)
fmt.Fprintf(w, `
  <div class="error">
    ✗ Upload failed: %s
  </div>
`, errorMessage)
```

## Summary

HTMX provides excellent support for file upload progress bars through the `htmx:xhr:progress` event. Key takeaways:

**Implementation:**
- Use `hx-encoding='multipart/form-data'` for file uploads
- Listen to `htmx:xhr:progress` event with JavaScript or Hyperscript
- Extract `loaded` and `total` from `evt.detail` to calculate progress

**Best Practices:**
- Include accessibility attributes (ARIA roles)
- Provide multiple forms of feedback (percentage, file size, speed)
- Validate files on both client and server
- Handle errors gracefully with clear user messaging
- Reset progress state between uploads
- Use CSS transitions for smooth visual updates

**Server-Side:**
- Return HTML fragments (not JSON) for HTMX compatibility
- Validate file size, type, and MIME type
- Sanitize filenames to prevent security issues
- Use appropriate HTTP status codes (200 for success, 400/500 for errors)

**Alternatives:**
- For server-side processing progress, use polling with `hx-trigger='every 600ms'`
- Combine `xhr:progress` (upload) with polling (processing) for hybrid scenarios

## References

- [HTMX File Upload Example](https://htmx.org/examples/file-upload/)
- [HTMX Progress Bar Example](https://htmx.org/examples/progress-bar/)
- [HTMX Events Documentation](https://htmx.org/events/)
- [GitHub: htmx progress discussion](https://github.com/bigskysoftware/htmx/issues/240)
- [GitHub: File Upload with Flask and HTMX](https://github.com/scheehan/File-Upload-with-Flask-HTMX-progress-bar)
- [Medium: Progress bar with HTMX](https://medium.com/@girishvenkatachalam/progress-bar-with-htmx-0e9160c7c6f9)
