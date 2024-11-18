# OSC Terraform Provider

## Documentation
Documentation is available on the [OSC Terraform Registry](https://registry.terraform.io/providers/EyevinnOSC/osc/latest).

## Testing the provider
There is an example provided in `examples/provider-install-verification`.

* Change to the `examples/provider-install-verification` directory.
* Optionally edit `main.tf` to change the name of the encore instance that will be created.
* Get a peronal access token for the OSC environment you wish to use.
* Set Secrets and tokens:
```sh
export OSC_ACCESS_TOKEN=<OSC PERSONAL ACCESS TOKEN>
export TF_VAR_osc_pat=$OSC_ACCESS_TOKEN
export TF_VAR_osc_env=prod
export TF_VAR_aws_keyid=<AWS KEYID>
export TF_VAR_aws_secret=<AWS SECRET>
```
* run `terraform init`.
* run `terraform apply`. This will create an encore instance in the selected OSC environment.
* start an encore job using the provided script, e.g. `./examples/provider-install-verification/encoreJob.sh "http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/WeAreGoingOnBullrun.mp4"`
* run `terraform destroy` to clean up.

