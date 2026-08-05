* [x] 1\. Add provider selection to lookup configuration

  * Add a configuration field and environment variable for lookup provider selection.

  * Default the provider to `steam_public` when no explicit provider is configured.

  * Return a clear configuration error for unsupported provider values.

* [x] 2\. Rename the current Steam-backed implementation

  * Rename the current Steam lookup client and related constructors/tests to the `steam_public` naming model.

  * Preserve current request behavior, normalized output mapping, and retry behavior.

* [x] 3\. Introduce provider-based client construction

  * Add a small factory or selection switch that constructs the configured provider adapter.

  * Keep provider-specific validation inside the selected provider path.

  * Ensure lookup command execution continues to call the same normalized lookup workflow.

* [x] 4\. Add tests for provider selection behavior

  * Add coverage for default provider selection.

  * Add coverage for explicit supported provider selection.

  * Add coverage for unsupported provider configuration failures.

* [x] 5\. Update documentation and validate the change

  * Update `CLI_USAGE.md` and any lookup configuration guidance to document configuration-based provider selection.

  * Run `go test ./...` with the project's working test invocation.

  * Run `openspec validate "add-lookup-provider-selection" --type change`.
