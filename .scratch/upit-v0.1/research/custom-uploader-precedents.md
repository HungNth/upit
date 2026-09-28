# Custom uploader precedents for Upit v0.1

Research date: 2026-09-28. Sources are first-party ShareX source/examples and the Go standard library source.

## ShareX custom uploader contract

ShareX's official [`CustomUploaders` repository](https://github.com/ShareX/CustomUploaders) stores uploader definitions as JSON `.sxcu` files. Its [`Example.sxcu`](https://raw.githubusercontent.com/ShareX/CustomUploaders/master/Example.sxcu) shows the relevant fields together:

- `RequestMethod` and `RequestURL` select the HTTP method and endpoint. `Headers` is a string-to-string map. The implementation also supports a separate `Parameters` map for query parameters; [`CustomUploaderItem.GetRequestURL`](https://github.com/ShareX/ShareX/blob/develop/ShareX.UploadersLib/CustomUploader/CustomUploaderItem.cs) parses the URL, evaluates `Parameters`, and appends them to the URL.
- `Body: "MultipartFormData"` selects multipart encoding. `FileFormName` names the file part, while `Arguments` is a separate map of ordinary multipart form fields. ShareX passes both maps, plus headers and the configured method, to its file-request routine in [`CustomFileUploader.UploadCoreAsync`](https://github.com/ShareX/ShareX/blob/develop/ShareX.UploadersLib/FileUploaders/CustomFileUploader.cs). The official [`Jirafeau.sxcu`](https://raw.githubusercontent.com/ShareX/CustomUploaders/master/Jirafeau.sxcu) and [`Discord webhook (Image uploader).sxcu`](https://raw.githubusercontent.com/ShareX/CustomUploaders/master/Discord%20webhook%20(Image%20uploader).sxcu) examples demonstrate static `Arguments` alongside `FileFormName`.
- Header and field values are not limited to literals: `GetHeaders` and `GetArguments` run ShareX's custom syntax parser, while imported custom-uploader values disable arbitrary local file reads ([`CustomUploaderItem.cs`](https://github.com/ShareX/ShareX/blob/develop/ShareX.UploadersLib/CustomUploader/CustomUploaderItem.cs)).
- Successful responses use the `URL` expression as the result URL; if it is empty, ShareX falls back to the response body. The same configuration has `ThumbnailURL` and `DeletionURL`. Failed responses evaluate `ErrorMessage` and add the result as an uploader error ([`ParseResponse`](https://github.com/ShareX/ShareX/blob/develop/ShareX.UploadersLib/CustomUploader/CustomUploaderItem.cs)). The official example uses `URL: "$json:url$"` and `ErrorMessage: "$json:error$"` ([`Example.sxcu`](https://raw.githubusercontent.com/ShareX/CustomUploaders/master/Example.sxcu)).
- Response extraction is a capability, not only a fixed JSON field: ShareX's parser exposes response text (`response`), the final response URL (`responseurl`), JSONPath (`json`), XPath (`xml`), and regular-expression extraction (`regex`) ([`CustomUploaderFunctionResponse.cs`](https://github.com/ShareX/ShareX/blob/develop/ShareX.UploadersLib/CustomUploader/Functions/CustomUploaderFunctionResponse.cs), [`CustomUploaderFunctionResponseURL.cs`](https://github.com/ShareX/ShareX/blob/develop/ShareX.UploadersLib/CustomUploader/Functions/CustomUploaderFunctionResponseURL.cs), [`CustomUploaderFunctionJson.cs`](https://github.com/ShareX/ShareX/blob/develop/ShareX.UploadersLib/CustomUploader/Functions/CustomUploaderFunctionJson.cs), [`CustomUploaderFunctionXml.cs`](https://github.com/ShareX/ShareX/blob/develop/ShareX.UploadersLib/CustomUploader/Functions/CustomUploaderFunctionXml.cs), [`CustomUploaderFunctionRegex.cs`](https://github.com/ShareX/ShareX/blob/develop/ShareX.UploadersLib/CustomUploader/Functions/CustomUploaderFunctionRegex.cs)).

## Go `os.UserConfigDir` portability

The Go standard library documents and implements `os.UserConfigDir` in [`src/os/file.go`](https://github.com/golang/go/blob/master/src/os/file.go):

| Platform | Returned root | Documented environment/error behavior |
| --- | --- | --- |
| Linux and other Unix systems | `$XDG_CONFIG_HOME` when non-empty; otherwise `$HOME/.config` | A relative `XDG_CONFIG_HOME` is an error. If it is empty and `HOME` is unset, the call errors because neither location can be determined. |
| macOS (Darwin) | `$HOME/Library/Application Support` | Errors when `HOME` is unset. |
| Windows | `%AppData%` | Errors when `AppData` is unset. |

The API returns the root only; its documentation says callers should create an application-specific subdirectory within it. The Linux behavior follows the [freedesktop.org Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html). These are documented behaviors in the Go source comments and the function body, not assumptions about a particular shell or desktop environment.

## Implications for Upit v0.1

Evidence-backed choices:

1. Represent a custom request with distinct method, URL, headers, query/static parameters, body kind, multipart file-field name, and ordinary multipart fields. This directly mirrors ShareX's independently configurable `RequestMethod`, `RequestURL`, `Headers`, `Parameters`, `Body`, `FileFormName`, and `Arguments`.
2. Make the multipart file field explicit and keep static multipart fields separate from it. ShareX's established contract sends both a named file part and ordinary `Arguments` fields.
3. Give a successful upload a configurable response URL extractor, with at least a response-body path expression; provide a separate configurable error extractor for failed responses. ShareX's `URL` and `ErrorMessage` fields, plus its JSONPath implementation, demonstrate this need.
4. Resolve the per-user config root with `os.UserConfigDir()` and append an Upit-specific subdirectory. This preserves XDG behavior on Linux, the Application Support convention on macOS, and `%AppData%` on Windows; callers must propagate an error when the required environment cannot determine a path.

Open tradeoff: ShareX supports JSONPath, XPath, regex, raw response text, and response URL extraction. The precedent establishes the value of configurable extraction, but does not determine whether Upit's v0.1 should implement all of those syntaxes or begin with one documented extractor (for example, JSON paths) and expand later.
