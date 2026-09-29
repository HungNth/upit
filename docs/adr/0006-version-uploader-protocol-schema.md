# Version Uploader protocol changes as schema version 2

Upit v0.3 makes a clean cutover of `custom-uploader.json` from version 1 to version 2 because adding request body modes and response extractor types changes the strict Uploader contract. Version 1 is rejected rather than migrated or accepted through a compatibility reader; global `config.json` remains version 2 and `custom-shortener.json` remains version 1.
