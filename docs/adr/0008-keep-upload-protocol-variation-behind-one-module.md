# Keep upload protocol variation behind one deep module

Upit v0.3 keeps request construction, HTTP execution, bounded response handling, extraction, URL validation, and structured failures behind the existing upload module interface. Body writers and Response Extractors may be separated into private files or functions inside `internal/app`, but Upit does not create pass-through request/response packages or strategy interfaces; the CLI remains the primary external seam.
