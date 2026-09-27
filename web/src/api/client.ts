import createClient from "openapi-fetch";
import type { paths } from "../gen/api";

// Typed client generated from api/openapi.yaml (npm run gen).
// fetch is resolved per request so that tests can stub it.
export const api = createClient<paths>({
  baseUrl: new URL("/api/v1", window.location.origin).toString(),
  fetch: (request) => globalThis.fetch(request),
});
