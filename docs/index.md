---
page_title: "osc Provider"
subcategory: ""
description: |-
  Manages service instances, secrets, My Apps, My Pages, custom domains and parameter stores in Eyevinn Open Source Cloud https://www.osaas.io (OSC). A single generic osc_instance resource creates an instance of any service in the OSC catalog, with the service's parameters read from the catalog at plan and apply time.
---

# osc Provider

Manages service instances, secrets, My Apps, My Pages, custom domains and parameter stores in [Eyevinn Open Source Cloud](https://www.osaas.io) (OSC). A single generic `osc_instance` resource creates an instance of any service in the OSC catalog, with the service's parameters read from the catalog at plan and apply time.

This page is the complete guide to writing a configuration with the provider. It is
written to be read by people and by coding agents, and it is deliberately generic: the
provider learns each service's parameters from the OSC catalog at run time, so nothing
here needs to change when services are added or updated.

## What the provider manages

| Block | Purpose |
|---|---|
| `osc_instance` (resource) | A running instance of any service in the OSC catalog. One resource type covers every service. |
| `osc_secret` (resource) | A named secret stored in OSC and referenced from instance parameters, so the value never appears in the instance configuration. |
| `osc_my_app` (resource) | A My App: your own application, built from a git repository and run on Web Runner. |
| `osc_my_page` (resource) | A My Page static site at `<name>.pages.osaas.io`, with its custom domain and basic auth. The files are published by CI, not Terraform. |
| `osc_domain` (resource) | A custom domain mapped to a service instance or a My App. |
| `osc_parameter_store` (resource) | A parameter store, the source of a My App's environment variables. |
| `osc_parameter` (resource) | One value, plain or secret, in a parameter store. |
| `osc_mailbox` (resource) | The workspace mailbox, `{tenantId}@users.osaas.io`, with SMTP and IMAP access. It is how a My App sends mail. |
| `osc_service` (data source) | A service's catalog entry, including the parameters an `osc_instance` of it accepts. |
| `osc_workspace` (data source) | The workspace and user the current token belongs to. |

The provider does not manage OSC "My Jobs" (run to completion jobs), nor the files of a My
Page or the releases of a My App beyond what `source_ref` and `rebuild_trigger` express. Use
the OSC web console, CLI or MCP server for those.

Every service in the catalog has its own page under **Guides → Services** in the sidebar, with a
ready to paste `osc_instance` block and the parameter table. The pages and the machine readable
mirror behind them, `catalog/services.json` in the provider repository, are regenerated weekly
from the OSC catalog. Agents can fetch the mirror directly from
`https://raw.githubusercontent.com/EyevinnOSC/terraform-provider-osc/main/catalog/services.json`.

## Concepts

- **Workspace** (also called tenant). Every OSC access token is bound to exactly one workspace
  and everything a configuration creates lands there. Set `workspace` on the provider, or the
  `OSC_WORKSPACE` environment variable, and the provider refuses to run with a token for a
  different workspace. Without it the plan warns which workspace the token targets. Always set
  it when the user has more than one workspace.
- **Service** and **service id**. A service is an open source project published in the OSC
  catalog. Its id has the form `{contributor}-{name}`, for example `valkey-io-valkey`,
  `apache-couchdb` or `eyevinn-encore-callback-listener`. Ids cannot be derived from the
  project name; they must be looked up (see below).
- **Subscription**. A workspace must be subscribed to a service before it can create instances
  of it. Subscriptions are free and `osc_instance` subscribes automatically on first use.
- **Instance**. A deployment of a service in a workspace, identified by service id and
  instance name. Names are usually lowercase letters and digits and must be unique per service
  within the workspace.
- **Parameters**. Each service declares its own parameters (name, type, required, allowed
  values, pattern, sensitive) in the catalog. In Terraform they are a map of strings keyed by
  the exact parameter name.

## Authentication

The provider resolves the access token in this order:

1. The `pat` provider attribute.
2. The `OSC_ACCESS_TOKEN` environment variable.
3. The token saved by `osc login` from the OSC CLI (`~/.osc/token`, or `~/.osc/token-<env>` for
   non production environments).

Personal access tokens are created in the OSC web console under the workspace settings. Tokens
from `osc login` expire after roughly an hour, so automation and agents should use a personal
access token. An expired token is rejected with a clear error before anything is planned.

Use the `osc_workspace` data source to see which workspace and user a token belongs to and when
it expires.

```terraform
terraform {
  required_providers {
    osc = {
      source = "EyevinnOSC/osc"
    }
  }
}

# The token is resolved from the pat attribute, then the OSC_ACCESS_TOKEN
# environment variable, then the token saved by `osc login`.
# workspace (or OSC_WORKSPACE) pins the target; a token for any other workspace is refused.
provider "osc" {
  environment = "prod"
  workspace   = "mytenant"
}
```

## Writing a configuration, step by step

### 1. Find the service id

Never guess a service id. Get it from one of these sources:

- **OSC MCP server (recommended for agents)** at `https://mcp.osaas.io/mcp`. The
  `list-available-services` tool searches the catalog and returns ids and descriptions, and
  `get-service-schema` returns the parameters used in the next step.
- **OSC web console** at `https://app.osaas.io`. Open the service in the catalog; the id is shown on
  the service page and in its URL.
- **An existing instance**. The OSC CLI (`npm install -g @osaas/cli`) lists instances with
  `osc list <serviceId>`, and the same id is used in Terraform.

### 2. Read the service's parameter schema

Parameter names are case sensitive and differ per service, so read the schema before writing
the block. Use the first option that is available to you:

1. **The OSC MCP server (recommended).** Call `get-service-schema` with the service id. It
   needs no Terraform run and no subscription, and it returns everything the block needs:
   parameter names, types, which are required, allowed values, patterns, defaults, and which
   are sensitive. Agents should connect the server at `https://mcp.osaas.io/mcp` (or the
   read only surface at `https://mcp.osaas.io/mcp/read-only`) with the same personal access
   token the provider uses. Setup instructions are at `https://www.osaas.io/mcp`.
2. **The `osc_service` data source.** Use this when the MCP server is not available. Set
   `subscribe = true` so it works before the workspace is subscribed, then run
   `terraform apply` or `terraform console` and read `parameters`:

   ```terraform
   data "osc_service" "svc" {
     service_id = "valkey-io-valkey"
     subscribe  = true
   }

   output "svc_parameters" {
     value = data.osc_service.svc.parameters
   }
   ```

   Each entry has `name`, `type` (string, boolean, enum or list), `required`, `enum`, `pattern`,
   `default`, `sensitive` and `description`.
3. **The provider's plan output**, as a last resort. Write the `osc_instance` block with a best
   guess and run `terraform plan`. Once the workspace is subscribed, mistakes fail at plan time
   and the error lists the accepted parameters of the service, with a "Did you mean" hint for
   misspelled or wrongly cased names, and a link to the service's guide page. If the workspace
   is not subscribed yet, plan validates against the weekly catalog mirror instead and warns
   that apply will subscribe first. Plan errors do not say which parameters are sensitive, so
   check the schema for that before settling on the block.

### 3. Write the `osc_instance` block

```terraform
resource "osc_instance" "cache" {
  service_id = "valkey-io-valkey"
  name       = "mycache"

  parameters = {
    # Non secret parameters, keyed by the exact name from the schema.
  }

  sensitive_parameters = {
    Password = var.valkey_password
  }
}
```

Rules that apply to every service:

- `service_id` and `name` are the identity of the instance. Changing either replaces it.
- Do not put `name` in `parameters`; it is the top level attribute.
- Values are strings. Boolean parameters take `"true"` or `"false"`. List parameters take the
  string form the service documents.
- Put passwords, tokens and keys, and anything the schema marks `sensitive`, in
  `sensitive_parameters`. A parameter goes in exactly one of the two maps.
- Required parameters must be set. Optional parameters can be omitted and the service applies
  its default.
- `wait_for_ready` (default true) makes create and update wait up to five minutes for the
  instance to report healthy. Set it to false for services that never report health.

### 4. Connect instances to each other

Every instance exposes `url`, and services that expose a raw TCP or UDP port also expose
`external_ip` and `external_port`. Reference them from other instances' parameters and
Terraform orders the creation for you:

```terraform
resource "osc_instance" "callback" {
  service_id = "eyevinn-encore-callback-listener"
  name       = "mycallback"

  parameters = {
    RedisUrl  = "redis://${osc_instance.cache.external_ip}:${osc_instance.cache.external_port}"
    EncoreUrl = trimsuffix(osc_instance.encore.url, "/")
  }
}
```

`url` normally ends with a slash. Use `trimsuffix` where a service wants a bare base URL.
The full instance document as returned by the service is available as the `instance` map, with
nested values JSON encoded and sensitive values redacted.

### 5. Keep secrets out of instance configuration

`osc_secret` stores a value in OSC for one or more services and exposes `ref`, a
`{{secrets.<name>}}` placeholder that OSC resolves when the instance starts. Pass `ref`
as the parameter value instead of the secret itself:

```terraform
resource "osc_secret" "db_password" {
  service_ids  = ["apache-couchdb"]
  secret_name  = "mydbadminpw"
  secret_value = var.db_password
}

resource "osc_instance" "db" {
  service_id = "apache-couchdb"
  name       = "mydb"

  parameters = {
    AdminPassword = osc_secret.db_password.ref
  }

  lifecycle {
    replace_triggered_by = [osc_secret.db_password]
  }
}
```

Secrets are read at instance start, so changing a secret does not restart the instance by
itself. `replace_triggered_by` makes the instance follow the secret. The secret value still
lives in the Terraform state, so protect the state as you would any credential store.

### 6. Plan, apply, verify

```shell
terraform init
terraform validate
terraform plan
terraform apply
```

Read plan output with these diagnostics in mind:

- **Service not yet subscribed** (warning). Parameters could not be checked at plan time because
  the workspace has not used the service before. Apply subscribes and validates. A wrong service
  id fails at this point.
- **Unknown parameter**, **Missing required parameter**, **Invalid parameter value** (errors).
  The detail text contains the full accepted schema of the service. Fix the block and plan again.
- **Unsupported service type** (error). The id belongs to a run to completion job, which
  `osc_instance` does not manage.
- **Instance not confirmed running** (warning after apply). The instance was created but did not
  report healthy within five minutes. Check it in the console or with `osc logs`.

After apply, `terraform output` shows the URLs. Instances are also visible in the OSC web
console and with `osc list <serviceId>`.

## Lifecycle behaviour

| Change | Effect |
|---|---|
| Edit `parameters` or `sensitive_parameters` | Updated in place with a rolling restart where the service supports it. |
| Change `service_id` or `name` | The instance is destroyed and a new one created. |
| Change `use_latest` | Replaced, since OSC picks the image at creation time. |
| Instance deleted outside Terraform | Detected on the next plan and recreated. |
| Parameter changed outside Terraform | Visible in the refreshed `instance` map. Configured `parameters` are not read back, so re-apply is not triggered; run `terraform apply` to push the configuration again. |
| `terraform destroy` | Deletes the instances and secrets. Data held by the service is lost unless the service persists it elsewhere. |

Existing instances can be brought under management without recreating them:

```shell
terraform import osc_instance.cache valkey-io-valkey/mycache
```

The import id is always `<service_id>/<instance name>`, the same value as the resource's `id`.
Import reads the instance from OSC, so `parameters` and `sensitive_parameters` land in state
and `terraform plan -generate-config-out=generated.tf` with an `import` block writes a complete
resource block.

## Bringing a whole workspace under Terraform

With Terraform 1.14 or later the provider can discover every instance in the workspace and
generate the configuration for it. Put a `list` block in a `.tfquery.hcl` file:

```hcl
list "osc_instance" "all" {
  provider = osc
}
```

Then run:

```shell
terraform query -generate-config-out=generated.tf
```

`generated.tf` gets an `import` block and an `osc_instance` block, parameters included, for
every instance that is not already in state. `terraform plan` should then show only the imports,
and `terraform apply` adopts the instances without recreating them. Rename the generated resources
(`all_0`, `all_1`, ...) to something meaningful first; nothing is in state yet, so it is a plain
text edit.

Passwords and tokens are handled without ever landing in a file. Terraform does not write
sensitive values into generated configuration, so parameters the catalog marks sensitive come
out as `sensitive_parameters = null # sensitive`. The import reads those values into state, and
an unset `sensitive_parameters` keeps whatever is in state, so the instance keeps its password
and the plan is clean. To take control of a value later, set it in `sensitive_parameters`,
preferably as an `osc_secret` reference; to remove all sensitive parameters set
`sensitive_parameters = {}`.

Set `service_id` in the list block's `config` to limit the query to one service.

## Patterns that come up often

- **Several environments in one workspace.** Prefix instance names with a variable, for example
  `"${var.env}cache"`. Names must stay within the service's allowed pattern, usually lowercase
  letters and digits.
- **Predicting an instance URL before it exists.** Some services need the URL of a sibling
  (for example a CORS origin) at creation time. Instance URLs follow the platform convention
  `https://{workspace}-{name}.{service host}` where the service host is the `api_url` host of the
  `osc_service` data source without its `api-` prefix. The `examples/open-live` directory in the
  provider repository shows this end to end, including a postcondition that checks the guess.
- **One off setup after creation.** Use a `terraform_data` resource with a `local-exec`
  provisioner that depends on the instance, for tasks such as creating a database inside a
  freshly started database server.
- **Hosting your own app.** `osc_parameter_store`, one `osc_parameter` per environment variable,
  then `osc_my_app` with `config_service` set to the store and `depends_on` the parameters, so the
  app starts with its environment in place. Parameters are read when the app starts: include their
  values, or a hash of them, in the app's `rebuild_trigger` to restart it when they change. For a
  custom domain add an `osc_domain` with `service_id = osc_my_app.<name>.domain_service_id` and
  `instance_name = osc_my_app.<name>.id`, after creating a CNAME to the app's `managed_domain`.
- **Sending mail from an app.** Create the workspace's `osc_mailbox` (one per workspace, paid plan)
  and pass `smtp_host`, `smtp_port`, `email` as the user and the mailbox password to the app as
  `osc_parameter` values, the password as `secret_value`. The mailbox can send only as its own
  address, so send as `email` and set Reply-To to an inbox that is read. If the workspace already
  has a mailbox, import it with `terraform import osc_mailbox.<name> mailbox`.
- **Migrating from the CLI.** `osc create <serviceId> <name> -o Key=Value` uses the same
  parameter names as `parameters`, so an existing CLI command translates directly into an
  `osc_instance` block.

## Where to look for more

- Provider resource and data source reference: the pages in this documentation.
- OSC web console and service catalog: `https://app.osaas.io`
- OSC platform documentation: `https://docs.osaas.io`
- OSC MCP server for agents: `https://mcp.osaas.io/mcp`, documented at `https://www.osaas.io/mcp`
- OSC CLI: `npm install -g @osaas/cli`
- Provider source, examples and changelog: `https://github.com/EyevinnOSC/terraform-provider-osc`

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `environment` (String) Which OSC environment to use, e.g. 'dev' or 'prod'. Defaults to 'prod'.
- `pat` (String, Sensitive) Access token used when communicating with the OSC API. Resolved in this order: this attribute, the `OSC_ACCESS_TOKEN` environment variable, then the token saved by `osc login` (`~/.osc/token`, or `~/.osc/token-<environment>` for non-prod). Tokens from `osc login` are short lived; use a personal access token for CI and agents.
- `workspace` (String) Workspace (tenant) id this configuration targets. Falls back to the `OSC_WORKSPACE` environment variable. Every OSC access token is bound to one workspace, and the provider refuses to run if the resolved token belongs to a different workspace. When neither is set the plan warns which workspace the token belongs to. Set it whenever the user has access to more than one workspace. Use the `osc_workspace` data source to see the workspace of the current token.
