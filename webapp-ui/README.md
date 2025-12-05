# Webapp UI (React + TypeScript + Vite)

This folder contains the new single-page application that mirrors the legacy AngularJS UI. It is built with React, TypeScript, and Vite.

## Prerequisites

- Node.js **20+** (required by Vite 7). Use a version manager such as `nvm` if your system default is older.
- npm (bundled with Node.js).

## Install dependencies

```bash
cd webapp-ui
npm install
```

## Run in development

Starts Vite's dev server with hot reload on <http://localhost:5173> by default.

```bash
npm run dev
```

If you need to access a backend running on a different host/port, configure your proxy or `vite.config.ts` accordingly.

## Build for production

Builds the TypeScript project and outputs a static bundle to `webapp-ui/dist`.

```bash
npm run build
```

## Preview the production build

Serves the production build locally for validation.

```bash
npm run preview
```

## Lint

Run ESLint across the codebase.

```bash
npm run lint
```

## Notes

- The app fetches configuration (title, feature flags, abuse contact, etc.) from the server's configuration endpoint on load. Ensure the backend is reachable when running locally.
- Authenticated routes and menu visibility follow the same logic as the legacy UI.
