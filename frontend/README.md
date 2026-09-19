# FITS Processor — Frontend

React 18 + TypeScript + Vite + Tailwind CSS

## Setup

```bash
cd frontend
npm install
npm run dev        # starts on http://localhost:3000
```

The dev server proxies `/api` and `/health` to `http://localhost:8080` automatically.
Make sure the backend is running (`make serve` in the project root).

## Build for production

```bash
npm run build      # outputs to frontend/dist/
```

Serve `dist/` with any static file server or configure Go to serve it directly.

## Pages

| Path | Access | Description |
|---|---|---|
| `/login` | Public | Login page |
| `/dashboard` | All | Stats cards + recent jobs |
| `/files` | All | Searchable/sortable FITS files table |
| `/files/:id` | All | File detail — metadata, headers, edit history |
| `/jobs` | All | Processing jobs list + scan trigger |
| `/jobs/:id` | All | Job detail with live progress + errors |
| `/profile` | All | My account info + change password |
| `/users` | Admin | Full user management CRUD |
