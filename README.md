# OSC Terraform Provider
(This repository is based on https://github.com/hashicorp/terraform-provider-scaffolding-framework .)

POC for an OSC Terraform provider.

# Requirements

Since this terraform provider is currently not publicly available, the below needs to be added to `$HOME/.terraformrc`

```
provider_installation {

  dev_overrides {
      "eyevinn.se/terraform/osc" = "$HOME/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}

```


# Testing the provider

* Do the steps in the [Requirements](#requirements) section.
* install the provider locally by running `go install .`.
* Change to the `examples/provider-install-verification` directory.
* Optionally edit `main.tf` to change the name of the encore instance that will be created
* Get a peronall access token for the OSC environment you wish to use.
* Set Secrets and tokens:
```sh
export OSC_ACCESS_TOKEN=<OSC PERSONAL ACCESS TOKEN>
export TF_VAR_osc_pat=$OSC_ACCESS_TOKEN
export TF_VAR_aws_keyid=<AWS KEYID>
export TF_VAR_aws_secret=<AWS SECRET>
```
* run `terraform apply`. This will create an encore instance in the selected OSC environment.
* start an encore job using the provided script, e.g. `./examples/provider-install-verification/encoreJob.sh "http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/WeAreGoingOnBullrun.mp4"`
* run `terraform destroy` to clean up.

