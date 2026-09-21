## 1.0.0 (2026-09-21)

FEATURES:
- `osc_instance` resource: manages an instance of any OSC catalog service. Parameters are validated against the catalog at plan time, changes are applied in place, state is refreshed from the service (drift and out-of-band deletion are detected), and existing instances can be imported as `<service_id>/<name>`.
- The provider overview page on the Terraform Registry is now a complete guide to writing a configuration: concepts, authentication, how to find service ids and parameter schemas, wiring instances together, secrets, plan diagnostics and lifecycle. Written for people and coding agents and kept generic so it does not change as the catalog does.
- `osc_instance.use_latest` provisions an instance from the latest built image instead of the stable release, for testing pre-release builds.
- `osc_service` data source: exposes a service's parameter schema. `subscribe = true` subscribes the workspace to the service first. Unsubscribed services are served from the catalog mirror, with `subscribed` telling which.
- Catalog mirror `catalog/services.json` and a generated guide page per service on the registry, refreshed weekly by the `catalog-sync` workflow (`make catalog`). The provider validates against the mirror at plan time for services the workspace has not subscribed to, and unknown service ids get suggestions from the whole catalog.
- Example `examples/open-live`: CouchDB, Open Live and Open Live Studio with secrets, as a complete stack.
- Provider `pat` is now optional. The token is resolved from `pat`, then `OSC_ACCESS_TOKEN`, then the token saved by `osc login`. Expired tokens are rejected with a clear message.
- `osc_workspace` data source: reports the workspace, user and token type of the current token.
- Provider `workspace` attribute, with `OSC_WORKSPACE` fallback: the provider refuses to run if the token belongs to a different workspace, and warns which workspace the token targets when neither is set.
- `osc_instance` list resource: `terraform query -generate-config-out=generated.tf` (Terraform 1.14+) discovers every instance in the workspace and generates `import` and `osc_instance` blocks for them. `osc_instance` now has a resource identity (`service_id`, `name`) and can be imported by identity.
- Importing an `osc_instance` reads its parameters from OSC into `parameters` and `sensitive_parameters`, so `terraform plan -generate-config-out` produces a complete resource block instead of `parameters = null`.
- `osc_instance.sensitive_parameters` keeps the values in state when the configuration leaves it unset, so imported and generated configurations keep their passwords without writing them to a file. Set it to `{}` to remove all sensitive parameters.

BUG FIXES:
- `osc_secret.ref` rendered the name with quotes. Changes to `osc_secret` now replace the secret instead of being silently ignored, and `secret_value` is marked sensitive.

BREAKING CHANGES:
- Per-service resources no longer exist; see REMOVED.

REMOVED:
- All generated per-service resources (`osc_valkey_io_valkey`, `osc_encore`, ...), the generator that produced them (`template/`) and the weekly regeneration workflow. Migrate with `terraform state rm` and `terraform import osc_instance.<name> <service_id>/<instance name>`; the running instances are untouched.

## 0.1.0 (First Release)

FEATURES:
- Encore Resource
- Valkey Resource
- Encore Callback Listener Resource
- Encore Transfer Resource
- Secret Resource
