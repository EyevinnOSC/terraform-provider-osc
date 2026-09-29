## Unreleased

BUG FIXES:
- `osc_instance` stores the values of `sensitive_parameters` in OSC service secrets and passes the instance `{{secrets.<name>}}` references, the way the OSC CLIs do. Before, they were sent as plain instance options, so the service API returned passwords, tokens and URLs with credentials in plain text to anyone with access to the workspace. `secret_names` shows which secret holds each parameter. The provider updates a secret when its value changes and restarts the instance so it reads the new value. It deletes secrets that are no longer used, and deletes all of an instance's secrets when the instance is deleted. A value that already is a `{{secrets.<name>}}` reference is passed on as it is.

UPGRADE NOTES:
- After upgrading, `terraform plan` shows an in-place update for every `osc_instance` with `sensitive_parameters`. Applying it moves the values into secrets and updates the instance in place, with a restart where the service supports it. It never replaces the instance. If a service does not support in-place updates, the apply fails and the instance is left as it is. Set `sensitive_parameters_as_secrets = false` for a service that does not resolve secret references; its values then stay in plain text.
- OSC does not return secret values, so a secret changed outside Terraform is not detected. An instance option changed from its secret reference to something else is detected, and the next plan updates it back.

## 1.1.3 (2026-09-28)

BUG FIXES:
- A refresh no longer drops a resource from state because OSC failed to answer. A resource counts as deleted only when the API says it does not exist on every one of four reads over about 15 seconds; a 401, a 5xx, an HTML page, a timeout, or a 404 for a route the API does not serve now fails the plan instead of planning to recreate the resource. This applies to every resource. Before, a flaky OSC API could make a plan propose recreating a parameter store and all its parameters, or a My App in the middle of a rebuild.
- `osc_my_app` waits up to 90 seconds for a `failed` build status to turn `running` before failing the apply, and waits through failing status requests the same way. OSC can report a build that succeeds as failed for a while, and failing the apply tainted the app.
- `osc_parameter_store` waits until the store's config API answers before create returns, and `osc_parameter` retries while the store answers 404 for its routes (`Route POST:/api/v1/config not found`), as a store that has just started does.
- `osc_instance` retries creation for up to 3 minutes while the service cannot be reached (connection or TLS errors, 502/503/504, `ORCHESTRATOR_UNAVAILABLE`), and only lists the service's accepted parameters when the service actually rejected the request.

## 1.1.2 (2026-09-24)

BUG FIXES:
- `osc_my_app` waits through the app being missing for up to a minute while a rebuild recreates its instance, instead of failing the apply.

## 1.1.1 (2026-09-24)

DOCUMENTATION:
- The provider overview page lists `osc_mailbox` and shows how to send mail from a My App through it.

## 1.1.0 (2026-09-24)

FEATURES:
- `osc_my_app` resource: a My App built from a git repository and run on Web Runner. Manages the runtime, repository, `source_ref`, `sub_path`, git credentials (`git_token`, or a stored `git_credential` by name), the parameter store binding and high availability, each changed in place where the platform allows it. `rebuild_trigger` rebuilds the app from CI, and apply waits for the build and fails if it fails. Exposes `url`, `managed_domain` and `domain_service_id`. Importable by app id.
- `osc_my_page` resource: a My Page static site, with its custom domain and basic auth. Files stay with CI. Importable by name.
- `osc_domain` resource: a custom domain mapped to a service instance or a My App, with an optional origin path. Importable as `<service_id>/<instance_name>/<domain>`.
- `osc_parameter_store` resource: a parameter store created by the platform with its encryption and API keys, waiting until its API answers over HTTPS. Importable by name.
- `osc_parameter` resource: one plain (`value`) or secret (`secret_value`) value in a parameter store. Secrets are read back in plain text with the store's API key, so drift is detected. Importable as `<parameter_store>/<key>`.
- `osc_mailbox` resource: the workspace mailbox, `{tenantId}@users.osaas.io`, which is how a My App sends mail on OSC. Exposes the SMTP and IMAP hosts, ports and encryption for wiring into `osc_parameter`; a changed `password` is set in place. A workspace has one mailbox, so creating a second fails with an import hint, and any id imports it.

BUG FIXES:
- Requests without a body no longer send `Content-Type: application/json`, which Fastify based OSC APIs reject.

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
