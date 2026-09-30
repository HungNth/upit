# Publish JSON Schemas while retaining Go semantic validation

Upit v0.4 publishes strict Draft 2020-12 schemas for the Global Configuration, Uploaders, and Shorteners, while Go validation remains the execution authority. The schemas mirror every intra-document constraint they can express; Go continues to enforce cross-document references, filesystem permissions, and protocol rules that JSON Schema cannot represent, avoiding a runtime schema-validator dependency without allowing the published contracts to drift.
