# Frontend

React + TypeScript + Ant Design client for the Go social-media API.

## Running

The backend, Postgres and NATS must be up first (see the repository root). Then:

```bash
npm install
npm run dev          # http://localhost:5173
```

Vite proxies `/api` to `http://localhost:8080`, including the WebSocket upgrade,
so the browser sees a single origin. Point it elsewhere with `VITE_BACKEND_URL`.

## Scripts

| Script | Purpose |
|---|---|
| `npm run dev` | Dev server with HMR |
| `npm run build` | Typecheck and production build |
| `npm run typecheck` | `tsc --noEmit` |
| `npm run generate:api` | Regenerate `src/api/schema.d.ts` from `../docs/openapi.json` |
| `npm run e2e` | Browser smoke test (needs backend + `npm run dev` running) |

Run `generate:api` whenever `docs/openapi.json` changes, so the client types stay
in step with the server.

## Layout

```
src/
  api/        generated schema, fetch client, one module per resource, query keys
  auth/       token store (Zustand + localStorage), JWT decoding, route guard
  ws/         feed socket: connect, reconnect with backoff, cache invalidation
  features/   one directory per screen
  components/ layout and shared presentational pieces
```

## Notes on the API

A few backend behaviours shape the client and are easy to trip over:

- **Login takes a user UUID, not an email.** Registration returns that id, which
  the register screen hands to the login screen so it can be prefilled.
- **Tokens last 24h and there is no refresh endpoint.** Any 401 clears the
  session and returns to sign-in; that is the only recovery.
- **Some 400/401 responses have an empty body**, so error handling cannot assume
  a `{message}` payload.
- **The WebSocket authenticates with `?token=`**, because the browser
  `WebSocket` constructor cannot set an `Authorization` header.
- **Only post-creation is pushed** over the socket — never updates or deletes,
  and never your own posts, since fan-out targets an author's followers. Those
  cases invalidate the cache client-side instead.
- **Search requires both name fields** and returns 400 if either is missing.
- **No endpoint returns a total count**, so lists use Previous/Next rather than
  numbered pages.

## The e2e smoke test

`e2e/smoke.mjs` drives two browser sessions through the whole app: register,
sign in, follow, post, receive that post live over the WebSocket without a
reload, exchange messages, then edit and delete a post. It also fails on any
console or page error, which is what catches deprecated component APIs.

It is a manual verification tool, not a CI suite — it needs a live backend, a
live dev server, and a real database, and it writes into that database.
