# Use UTF-8 text templates for form and JSON uploads

Upit v0.3 treats `form` and `json` Request Body Modes as text-upload protocols: the selected file must be valid UTF-8 and replaces exactly one full-string `{input}` value in configured fields or a JSON object. Binary files continue to use `multipart` or `binary`; Upit does not invent an implicit base64 convention, filename placeholder, or unbounded buffering for form and JSON endpoints.
