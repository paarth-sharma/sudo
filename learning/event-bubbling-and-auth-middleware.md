# Event Bubbling + Authentication Middleware Issues

## Problem
Board deletion failing with "NetworkError when attempting to fetch resource" and redirecting to board instead of deleting.

## Root Causes

### 1. Authentication Middleware Blocking Fetch Requests
```javascript
// BROKEN: fetch without credentials
fetch('/boards/123', { method: 'DELETE' })
```
- Protected routes require session cookies
- Regular `fetch()` doesn't include cookies by default
- Browser blocks request at network level

### 2. Event Bubbling in Nested Clickable Elements  
```html
<!-- BROKEN: Delete button inside clickable card -->
<div onclick="navigateToBoard()">
  <button hx-delete="/boards/123">Delete</button>
</div>
```
- Click on delete button bubbles up to parent card
- Parent's `onclick` triggers navigation instead of deletion

## Solutions

### 1. Use HTMX Instead of Fetch
```html
<!-- FIXED: HTMX includes credentials automatically -->
<button hx-delete="/boards/123" 
        hx-confirm="Are you sure?">Delete</button>
```

### 2. Stop Event Propagation
```html
<!-- FIXED: Prevent bubbling to parent -->
<button hx-delete="/boards/123" 
        onclick="event.stopPropagation()">Delete</button>
```

### 3. Backend Returns HTMX Redirect
```go
// FIXED: Proper HTMX redirect
c.Header("HX-Redirect", "/dashboard")
c.Status(http.StatusOK)
```

## Pattern Recognition

**Symptoms:**
- "NetworkError when attempting to fetch resource"
- Action redirects instead of executing
- No request logs on server

**Check For:**
- Missing `credentials: 'include'` in fetch requests
- Nested clickable elements without `stopPropagation()`
- Protected routes requiring authentication

## Takeaways

1. **Always use `credentials: 'include'` with fetch** for authenticated routes
2. **HTMX handles authentication automatically** - prefer over vanilla fetch
3. **Add `event.stopPropagation()`** to buttons inside clickable containers
4. **Use browser dev tools** to distinguish network vs. logic errors
5. **Check server logs first** - no logs = request never reached server