# URL-shortener API precedents for Upit v0.2

Research date: 2026-09-28. Sources below are first-party API specifications or first-party source repositories. `{input}` means the long URL supplied by Upit.

## Kutt

- **Request:** `POST https://<kutt-host>/api/v2/links` (the official OpenAPI source declares the server as `https://kutt.it/api/v2`; current Kutt source mounts the API under `/api/v2`). The request is JSON; `target` is required. Optional fields are `description`, `expire_in`, `password`, `customurl`, `reuse`, and `domain`. [`api.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/docs/api/api.js), [`link.routes.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/server/routes/link.routes.js), [`validators.handler.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/server/handlers/validators.handler.js)
- **Authentication/headers/query:** API-key auth is the `X-API-KEY` header. No query parameter is needed for creation. [`api.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/docs/api/api.js)
- **Template and success:** `{"target":"{input}"}` is sufficient for basic shortening. Current creation code returns status `201` and a link object whose short URL is the `link` field; the OpenAPI example also shows `address`, `id`, `target`, and timestamps. [`links.handler.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/server/handlers/links.handler.js), [`utils.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/server/utils/utils.js), [`api.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/docs/api/api.js)
- **Errors:** Kutt's API error middleware returns JSON `{ "error": "<message>" }`; validation errors are `400`, strict authentication errors are `401`, banned users are `403`, and an unspecified error defaults to `500`. [`helpers.handler.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/server/handlers/helpers.handler.js), [`auth.handler.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/server/handlers/auth.handler.js), [`utils.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/server/utils/utils.js)
- **Basic requirements:** `target` is the only required creation field. A custom URL and custom domain are optional and authenticated-user/domain-dependent; Kutt generates an address when `customurl` is omitted. [`api.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/docs/api/api.js), [`validators.handler.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/server/handlers/validators.handler.js), [`links.handler.js`](https://raw.githubusercontent.com/thedevs-network/kutt/develop/server/handlers/links.handler.js)

## Bitly

- **Request:** `POST https://api-ssl.bitly.com/v4/shorten`. JSON body schema: required `long_url`; optional `domain` (default `bit.ly`), `group_guid`, and `force_new_link`. [`v4.json`](https://dev.bitly.com/v4/v4.json) (`/shorten`, `Shorten` schema)
- **Authentication/headers/query:** HTTP Bearer authentication, normally sent as `Authorization: Bearer {TOKEN}`, plus `Content-Type: application/json` in the official example. No query parameter is needed. [`v4.json`](https://dev.bitly.com/v4/v4.json), [Bitly API reference](https://dev.bitly.com/api-reference/)
- **Template and success:** `{"long_url":"{input}"}` is sufficient according to the required/optional schema. Success is `200` or `201`; the returned short URL is `link`. [`v4.json`](https://dev.bitly.com/v4/v4.json) (`/shorten` response and `ShortenBitlinkBody`)
- **Errors:** The endpoint documents `400`, `403`, `417`, `422`, `429`, `500`, and `503` (the related full Bitlink endpoint additionally documents `402` and `404`). Bitly's error models use `message`, `description`, and `resource`, with optional field-level `errors` entries containing `field`, `error_code`, and `message`. The endpoint description names provider codes including `BRANDED_LINK_MONTHLY_LIMIT_EXCEEDED` and `DNS_CONFIGURATION_ERROR`. [`v4.json`](https://dev.bitly.com/v4/v4.json) (`/shorten`, `Error`, `SimplifiedError`, and `FieldError` schemas)
- **Basic requirements:** No custom domain or arbitrary metadata is required; `domain` defaults to `bit.ly`, and only `long_url` is required by the `Shorten` schema. [`v4.json`](https://dev.bitly.com/v4/v4.json)

## Shlink

- **Request:** `POST https://<shlink-host>/rest/v3/short-urls`. The current first-party v5.1.7 OpenAPI path is `/rest/v{version}/short-urls`; the source route defines `/short-urls` with POST and the first-party config applies the `/rest/v{version:1|2|3}` prefix. The JSON body requires `longUrl`; the schema also accepts fixed optional fields such as `validSince`, `validUntil`, `customSlug`, `pathPrefix`, `maxVisits`, `findIfExists`, `domain`, `shortCodeLength`, `tags`, `title`, `crawlable`, and `forwardQuery`. [`v1_short-urls.json`](https://raw.githubusercontent.com/shlinkio/shlink/v5.1.7/docs/swagger/paths/v1_short-urls.json), [`CreateShortUrlAction.php`](https://raw.githubusercontent.com/shlinkio/shlink/develop/module/Rest/src/Action/ShortUrl/CreateShortUrlAction.php), [`ConfigProvider.php`](https://raw.githubusercontent.com/shlinkio/shlink/develop/module/Rest/src/ConfigProvider.php), [`ShortUrlCreation.php`](https://raw.githubusercontent.com/shlinkio/shlink/develop/module/Core/src/ShortUrl/Model/ShortUrlCreation.php)
- **Authentication/headers/query:** Every endpoint requires `X-Api-Key: {api_key}`. Shlink's API documentation recommends `Accept: application/json` so unexpected errors do not become HTML. The version is a path parameter (`v3` here); no query parameter is needed for basic creation. [`authentication`](https://shlink.io/documentation/api-docs/authentication/), [API docs](https://shlink.io/documentation/api-docs/), [`v1_short-urls.json`](https://raw.githubusercontent.com/shlinkio/shlink/v5.1.7/docs/swagger/paths/v1_short-urls.json)
- **Template and success:** `{"longUrl":"{input}"}` is sufficient when optional fields are omitted. Success is `200`; the response's short URL is `shortUrl`. [`v1_short-urls.json`](https://raw.githubusercontent.com/shlinkio/shlink/v5.1.7/docs/swagger/paths/v1_short-urls.json), [`AbstractCreateShortUrlAction.php`](https://raw.githubusercontent.com/shlinkio/shlink/develop/module/Rest/src/Action/ShortUrl/AbstractCreateShortUrlAction.php), [`ShortUrlDataTransformer.php`](https://raw.githubusercontent.com/shlinkio/shlink/develop/module/Core/src/ShortUrl/Transformer/ShortUrlDataTransformer.php)
- **Errors:** The create operation documents `400` for invalid data and a `default` unexpected-error response; both use `application/problem+json`. Problem Details fields are `type`, `detail`, `title`, and `status`; extra fields such as `invalidElements`, `customSlug`, or `domain` can occur. Authentication failure is `401` with the documented invalid-key shape. [`v1_short-urls.json`](https://raw.githubusercontent.com/shlinkio/shlink/v5.1.7/docs/swagger/paths/v1_short-urls.json), [`error-management`](https://shlink.io/documentation/api-docs/error-management/), [`authentication`](https://shlink.io/documentation/api-docs/authentication/)
- **Basic requirements:** `longUrl` is required; `domain` and custom slug are optional model fields, so a custom domain is not required for basic shortening. The request schema exposes a fixed set of named fields rather than an arbitrary custom-field map. [`v1_short-urls.json`](https://raw.githubusercontent.com/shlinkio/shlink/v5.1.7/docs/swagger/paths/v1_short-urls.json), [`ShortUrlCreation.php`](https://raw.githubusercontent.com/shlinkio/shlink/develop/module/Core/src/ShortUrl/Model/ShortUrlCreation.php)

## Dub

- **Request:** `POST https://api.dub.co/links`. JSON body requires `url`; `domain` is optional and defaults to the workspace's primary domain or `dub.sh`. The official schema also lists many named optional fields, including `key`, `keyLength`, `externalId`, `tenantId`, `prefix`, tracking, expiry, password, and preview fields. [`create.md`](https://dub.co/docs/api-reference/links/create.md)
- **Authentication/headers/query:** Bearer authentication uses the `Authorization: Bearer DUB_API_KEY` convention from the `token` security scheme. No query parameter is needed for creation. [`create.md`](https://dub.co/docs/api-reference/links/create.md)
- **Template and success:** `{"url":"{input}"}` is sufficient for basic shortening. Success is `200`; the response's full short URL is `shortLink`. [`create.md`](https://dub.co/docs/api-reference/links/create.md)
- **Errors:** The operation documents `400`, `401`, `403`, `404`, `409`, `410`, `422`, `429`, and `500`. Each uses an outer `error` object with required `code` and `message`; the code identifies categories such as `bad_request`, `unauthorized`, `conflict`, `unprocessable_entity`, `rate_limit_exceeded`, and `internal_server_error`. [`create.md`](https://dub.co/docs/api-reference/links/create.md)
- **Basic requirements:** `url` is the only required body field. A custom domain, custom key, and other metadata are optional; the schema describes named fields, not arbitrary user-defined fields. [`create.md`](https://dub.co/docs/api-reference/links/create.md)

## Common-denominator analysis

All four basic operations can be described by an endpoint URL, a POST method, static headers, and a JSON object template with the input substituted into one JSON string value:

| Service | JSON body template | Success short-URL JSONPath | Representative error JSONPath |
| --- | --- | --- | --- |
| Kutt | `{"target":"{input}"}` | `$.link` | `$.error` |
| Bitly | `{"long_url":"{input}"}` | `$.link` | `$.message` (or `$.errors[*].message`) |
| Shlink | `{"longUrl":"{input}"}` | `$.shortUrl` | `$.detail` |
| Dub | `{"url":"{input}"}` | `$.shortLink` | `$.error.message` |

The paths above are RFC 9535 JSONPath expressions; RFC 9535 defines JSONPath syntax and the query result model ([RFC 9535](https://www.rfc-editor.org/rfc/rfc9535)). The table is a representation of observed response fields, not a proposal for which extractor behavior Upit should select.

The basic cases provide no evidence that query parameters are required, that a non-POST creation method is needed, or that `{input}` must be substituted outside a JSON string value. Provider-specific logic remains for authentication-header conventions, body property names, success-field names, status policy, and error payload traversal. Optional capabilities can introduce additional requirements: Kutt's `domain` is a body field and requires the domain to be registered for the user; Bitly's branded-domain/group behavior is provider/account-specific; Shlink's and Dub's domain/custom-slug options are body fields; and Dub's richer options are named provider fields rather than arbitrary keys. Sources: the service citations above.

## Evidence-only findings

1. Each surveyed service has a JSON POST creation endpoint with one required destination URL field, but the field names differ (`target`, `long_url`, `longUrl`, `url`). Sources: service citations above.
2. Each service returns a JSON short-link field, but the field names differ (`link`, `shortUrl`, `shortLink`), and error payloads are not uniform. Sources: service citations above.
3. Authentication is header-based in all four examples, but Kutt and Shlink use API-key headers while Bitly and Dub use Bearer authorization. Sources: service citations above.
4. Basic shortening does not require a caller-supplied custom domain or arbitrary custom fields in any of the four documented schemas. Sources: service citations above.
5. Basic creation is POST-only in the surveyed endpoints; no surveyed basic case requires query parameters or placeholder substitution outside a JSON string value. Sources: service citations above.

## Open product tradeoffs

- Whether Upit should expose one `{input}` placeholder restricted to JSON string values, or support substitution in headers, query parameters, URL paths, and non-string JSON values for future providers.
- Whether one configured success JSONPath and one error JSONPath are enough, or whether status-specific/multiple fallback extractors are needed for Bitly-style alternate error shapes.
- Whether to model only static headers or also first-class authentication schemes, given the API-key versus Bearer split.
- Whether to expose named optional provider fields, a generic JSON object, or neither; the evidence shows rich fixed options but no need for arbitrary fields in basic shortening.
- Whether to require callers to configure `Accept: application/json` and `Content-Type: application/json`, or have the client infer/add them for JSON requests.
