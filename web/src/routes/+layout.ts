// SPA mode: no SSR (backend serves a static build; all data comes from
// /api/* + /ws at runtime). prerender builds the shell HTML at build time.
export const prerender = true;
export const ssr = false;
