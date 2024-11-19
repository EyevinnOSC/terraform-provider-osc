---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "osc_eyevinn_qr_generator Resource - osc"
subcategory: ""
description: |-
  Effortlessly create and customize QR codes with dynamic text and logos. Perfect for projects requiring quick updates. Launch your instance and deploy multiple codes seamlessly on the Open Source Cloud.
---

# osc_eyevinn_qr_generator (Resource)

Effortlessly create and customize QR codes with dynamic text and logos. Perfect for projects requiring quick updates. Launch your instance and deploy multiple codes seamlessly on the Open Source Cloud.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `goto_url` (String)
- `name` (String) Name of qr-generator

### Optional

- `logo_url` (String)

### Read-Only

- `external_ip` (String) The external Ip of the created instance (if available).
- `external_port` (Number) The external Port of the created instance (if available).
- `instance_url` (String) URL to the created instace
- `service_id` (String) The service id for the created instance