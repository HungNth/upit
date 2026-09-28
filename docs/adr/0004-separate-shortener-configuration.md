# Keep Shortener definitions in a separate configuration file

Upit v0.2 stores named Shorteners in strict `custom-shortener.json` version 1 while global `config.json` advances to version 2 to add the optional `defaultShortener`. This preserves a clear schema boundary between Uploaders and Shorteners and avoids forcing credential-bearing definitions with different request contracts into one versioned document; v0.2 is a clean cutover and does not accept global config version 1.
