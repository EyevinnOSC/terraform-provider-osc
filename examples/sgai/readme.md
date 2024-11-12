1. Make sure you have OSC_TOKEN set up
```
export OSC_ACCESS_TOKEN=<OSC PERSONAL ACCESS TOKEN>
export TF_VAR_osc_pat=$OSC_ACCESS_TOKEN
```
2. `terraform apply`
3. `curl https://eyevinnlab-sgai.eyevinn-sgai-ad-proxy.auto.prod.osaas.io/command?in=0&dur=10&pod=2`

