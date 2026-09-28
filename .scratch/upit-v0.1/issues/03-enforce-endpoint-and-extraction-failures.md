# 03: Enforce endpoint and extraction failure semantics

**Parent specification:** [Upit v0.1 — Headless Upload CLI](../spec.md)

**What to build:** Make endpoint behavior deterministic and safe. Upit must reject redirects and non-2xx responses, bound response memory, apply the full RFC 9535 extraction contract, sanitize endpoint-provided diagnostics, and return structured stage/status information for failures.

**Blocked by:** 02 — Configure and protect practical named Uploaders

**Status:** ready-for-agent

- [ ] Automatic HTTP redirects are disabled; every 3xx response is handled as an unsuccessful endpoint response.
- [ ] Only HTTP 2xx responses can become successful uploads.
- [ ] Decoded response content is limited to 1 MiB; exceeding the limit returns a response-stage error without unbounded allocation.
- [ ] The full RFC 9535 selector language supported by the pinned library is accepted, including object, array, wildcard, and filter selectors.
- [ ] Every URL or error extractor must select exactly one JSON string; zero matches, multiple matches, null, and non-string nodes produce distinct extraction failures without coercion.
- [ ] A successful URL remains required to be absolute and use HTTP or HTTPS.
- [ ] A configured error extractor is used when usable endpoint JSON is available; otherwise a safe status-based message is returned.
- [ ] Endpoint-provided text is scrubbed against every non-empty configured request value, and diagnostics never print request headers, query strings, multipart values, or a request URL containing its query.
- [ ] Runtime errors expose only the agreed stages: config, validation, request, network, response, and parse; HTTP status is attached only when available.
- [ ] No automatic retry occurs after request, network, response, or parse failure.
- [ ] CLI-seam tests cover 3xx, 4xx/5xx with and without extracted errors, oversized responses, nested/array/filter paths, missing/multiple/null/non-string selections, invalid result URLs, and secret reflection.
- [ ] A demo shows a failing endpoint produces a sanitized, stage-aware diagnostic and no success output.
