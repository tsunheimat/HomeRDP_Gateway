# RDP Gateway UI Templates

This directory contains the web UI templates and static assets used by the gateway.

## Template Files

### `dashboard.html`
Authenticated OpenID dashboard page (`/`) that renders:
- Current user details
- Entry cards loaded from `/api/v1/entries`
- Admin link (shown only when `/api/v1/user` reports `isAdmin: true`)

### `admin.html`
Authenticated OpenID admin page (`/admin`) for users in configured admin groups:
- Host entry create form
- Template upload form
- Direct-auth user create form
- Entry list management area
- Direct-auth user management area

### `index.html`
Legacy web interface template used by existing non-dashboard flows.

## JavaScript Files

### `dashboard.js`
Dashboard logic:
- Fetches `/api/v1/user` and `/api/v1/entries`
- Renders entry cards
- Starts entry downloads via `/connect/entries/{id}.rdp`
- Surfaces API failures in-page

### `admin.js`
Admin panel logic:
- Fetches `/api/v1/admin/entries`
- Fetches `/api/v1/admin/auth-users`
- Creates host entries with `POST /api/v1/admin/entries/host`
- Uploads template entries with `POST /api/v1/admin/entries/template`
- Updates/deletes entries via `PUT`/`DELETE /api/v1/admin/entries/{id}`
- Creates direct-auth users with `POST /api/v1/admin/auth-users`
- Updates/deletes direct-auth users via `PUT`/`DELETE /api/v1/admin/auth-users/{username}`
- Surfaces API failures in-page

### `app.js`
Legacy web interface logic used by `index.html`.

## Shared Styling

### `style.css`
Shared stylesheet for dashboard/admin and legacy templates.  
Update this file to keep visual consistency across all pages.

## Static Routes

- `/static/style.css`
- `/static/dashboard.js`
- `/static/admin.js`
- `/static/app.js` (legacy flow)
- `/assets/connect.svg`
- `/assets/icon.svg`

## OpenID Dashboard Routes

- `/` -> dashboard page
- `/admin` -> admin page (admin-only)
- `/api/v1/user` -> dashboard user info
- `/api/v1/entries` -> visible dashboard entries
- `/api/v1/admin/entries` -> admin entry list
- `/api/v1/admin/entries/host` -> create host entry
- `/api/v1/admin/entries/template` -> create template entry
- `/api/v1/admin/entries/{id}` -> update/delete entry
- `/api/v1/admin/auth-users` -> direct-auth user list/create
- `/api/v1/admin/auth-users/{username}` -> direct-auth user update/delete
- `/connect/entries/{id}.rdp` -> dashboard RDP download route

If `dashboard.html` or `admin.html` are missing, the server uses embedded fallback HTML.
