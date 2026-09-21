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
* Import existing instances with `terraform import osc_instance.x <service_id>/<name>`. Import reads the parameters from OSC, so `terraform plan -generate-config-out` produces a complete resource block.
* Discover and adopt everything in the workspace with `terraform query -generate-config-out=generated.tf` (Terraform 1.14+) and a `list "osc_instance"` block; see `examples/list-resources/osc_instance`.

Use the `osc_service` data source to read a service's parameter schema from Terraform:

```hcl
data "osc_service" "valkey" {
  service_id = "valkey-io-valkey"
}
output "params" { value = data.osc_service.valkey.parameters }
```

### Service guides and the catalog mirror
Every published service has a guide page on the registry under Guides → Services, with a ready to paste `osc_instance` block and its parameter table. The pages are generated from `catalog/services.json`, a mirror of the OSC catalog that a weekly workflow refreshes and opens a PR for (`make catalog` runs it locally with `OSC_API_KEY`). The provider reads the same mirror at plan time for services the workspace has not subscribed to yet, so parameters and service ids are validated before apply either way. Agents can fetch the mirror from the raw GitHub URL of `catalog/services.json` on `main`.

### `osc_secret`
Creates a secret in one or more services and exposes a `ref` (`{{secrets.<name>}}`) to use as a parameter value in `osc_instance`. OSC resolves the reference when the instance starts, so the secret never appears in the instance configuration.

See `examples/open-live` for a complete stack: CouchDB, Open Live and Open Live Studio wired together with secrets, a database bootstrap step and a predicted studio URL for CORS.

### Migrating from the removed per-service resources
Versions before 0.2.0 shipped one generated resource per service (`osc_valkey_io_valkey`, `osc_encore`, ...). They are gone. Move each one to an `osc_instance` with the same service id and instance name without touching the running instance:

```sh
terraform state rm osc_valkey_io_valkey.cache
terraform import osc_instance.cache valkey-io-valkey/mycache
```

Import reads the instance's parameters from OSC, so `terraform plan -generate-config-out=generated.tf` with an `import` block writes the matching `osc_instance` block for you. With Terraform 1.14+ `terraform query -generate-config-out=generated.tf` does this for every instance in the workspace at once.

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
