# OSC Terraform Provider

Terraform provider for [Eyevinn Open Source Cloud](https://www.osaas.io) (OSC).

## Documentation
Documentation is available on the [OSC Terraform Registry](https://registry.terraform.io/providers/EyevinnOSC/osc/latest). The provider overview page there is a complete, step by step guide to writing a configuration, including how to find service ids and parameter schemas. It is generated from `templates/index.md.tmpl`; edit the template and run `make generate`, never `docs/` directly.

## Authentication and workspaces
The access token is resolved in this order: the `pat` provider attribute, the `OSC_ACCESS_TOKEN` environment variable, then the token saved by `osc login` at `~/.osc/token` (`~/.osc/token-<env>` for non-prod). Tokens from `osc login` expire after about an hour, so CI and agents should use a personal access token.

Every OSC token is bound to exactly one workspace (tenant), and all resources are created in that workspace. Set `workspace` on the provider, or the `OSC_WORKSPACE` environment variable, and the provider fails before touching anything if the token belongs to a different workspace. Without it the plan warns which workspace the token targets. Always set it if you have more than one workspace. The `osc_workspace` data source reports the workspace and user of the current token.

## Resources

### `osc_instance` (recommended)
A single generic resource that manages an instance of **any** service in the OSC catalog. The service's parameter schema is read from the catalog at plan and apply time, so new services in OSC work without a provider update.

```hcl
provider "osc" {
  environment = "prod"
  workspace   = "mytenant" # or OSC_WORKSPACE; tokens for other workspaces are refused
}

resource "osc_instance" "cache" {
  service_id = "valkey-io-valkey"
  name       = "mycache"
  sensitive_parameters = {
    Password = var.valkey_password
  }
}

resource "osc_instance" "callback" {
  service_id = "eyevinn-encore-callback-listener"
  name       = "mycallback"
  parameters = {
    RedisUrl   = "redis://${osc_instance.cache.external_ip}:${osc_instance.cache.external_port}"
    EncoreUrl  = trimsuffix(osc_instance.encore.url, "/")
    RedisQueue = "transfer"
  }
}
```

* `service_id` is the id from the OSC catalog (`{contributor}-{name}`). The tenant is subscribed to the service automatically on first use.
* `parameters` and `sensitive_parameters` are maps of strings keyed by the parameter names in the service's schema. Unknown names, missing required parameters, invalid enum values and pattern violations are reported at plan time with the accepted schema in the error message.
* Parameter changes are applied in place with a rolling restart. Changing `service_id` or `name` replaces the instance.
* Outputs: `url`, `external_ip`, `external_port` and the full `instance` document as a map.
* Import existing instances with `terraform import osc_instance.x <service_id>/<name>`.

Use the `osc_service` data source to read a service's parameter schema from Terraform:

```hcl
data "osc_service" "valkey" {
  service_id = "valkey-io-valkey"
}
output "params" { value = data.osc_service.valkey.parameters }
```

### `osc_secret`
Creates a secret in one or more services and exposes a `ref` (`{{secrets.<name>}}`) to use as a parameter value in `osc_instance`. OSC resolves the reference when the instance starts, so the secret never appears in the instance configuration.

See `examples/open-live` for a complete stack: CouchDB, Open Live and Open Live Studio wired together with secrets, a database bootstrap step and a predicted studio URL for CORS.

### Migrating from the removed per-service resources
Versions before 0.2.0 shipped one generated resource per service (`osc_valkey_io_valkey`, `osc_encore`, ...). They are gone. Move each one to an `osc_instance` with the same service id and instance name without touching the running instance:

```sh
terraform state rm osc_valkey_io_valkey.cache
terraform import osc_instance.cache valkey-io-valkey/mycache
```

Then write the matching `osc_instance` block, with the old attributes as `parameters` keyed by the parameter names from the OSC catalog (`osc_service` shows them).

## Testing the provider locally
Build and install the provider and point Terraform at it with a dev override:

```sh
go install .
cat > dev.tfrc <<CFG
provider_installation {
  dev_overrides {
    "EyevinnOSC/osc" = "$(go env GOPATH)/bin"
  }
  direct {}
}
CFG
export TF_CLI_CONFIG_FILE=$PWD/dev.tfrc
export OSC_ACCESS_TOKEN=<OSC PERSONAL ACCESS TOKEN>
```

Then run `terraform plan` / `terraform apply` in a directory with a configuration, for example `examples/provider-install-verification`. Unit tests run with `make test`. Documentation is regenerated with `make generate`.
