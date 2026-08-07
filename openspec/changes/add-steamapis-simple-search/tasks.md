- [x] 1.1 Add provider-specific default base URL and Steam App ID to game-code validation
- [x] 1.2 Add an authenticated SteamApis v2 direct-search client using the existing lookup interface
- [x] 1.3 Normalize marketplace price, offer count, app name, stable ID, and shared observation time without persistence

- [x] 2.1 Cover successful search, authentication header, query/limit parameters, null price, and no matches
- [x] 2.2 Cover missing API key, unsupported app, invalid payload, dependency failure, and bounded rate-limit failure
- [x] 2.3 Prove invalid input stops before provider work and lookup failure does not append records

- [x] 3.1 Document SteamApis selection, secret injection, direct-search limitations, price provenance, and rollback
- [x] 3.2 Update browser copy so lookup results are not described as official Steam public prices
- [x] 3.3 Describe SteamApis and OpenAI as optional, replaceable, environment-configured adapters and make degraded-state copy preserve the deterministic website boundary
- [x] 3.4 Classify exhausted Steam official `429` retries and recommend a supported third-party Steam Market Provider without labeling the website unavailable

- [x] 4.1 Run OpenSpec validation, formatting, JavaScript syntax validation, vet, race tests, and all binary builds
- [x] 4.2 Smoke-test one deterministic local SteamApis-shaped endpoint without writing records
- [ ] 4.3 Run one bounded real SteamApis query after a temporary API key is supplied; do not claim live provider verification before then
