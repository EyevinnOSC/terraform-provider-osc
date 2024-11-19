---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "osc_eyevinn_sgai_ad_proxy Resource - osc"
subcategory: ""
description: |-
  Boost viewer engagement with our Server-Guided Ad Insertion Proxy! Automatically embed ads into video streams with precision timing. Enhance monetization effortlessly while maintaining a seamless user experience.
---

# osc_eyevinn_sgai_ad_proxy (Resource)

Boost viewer engagement with our Server-Guided Ad Insertion Proxy! Automatically embed ads into video streams with precision timing. Enhance monetization effortlessly while maintaining a seamless user experience.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `insertion_mode` (Number)
- `name` (String) Name of sgai-ad-proxy
- `origin_url` (String)
- `vast_endpoint` (String)

### Optional

- `couch_db_endpoint` (String)
- `couch_db_password` (String)
- `couch_db_table` (String)
- `couch_db_user` (String)

### Read-Only

- `external_ip` (String) The external Ip of the created instance (if available).
- `external_port` (Number) The external Port of the created instance (if available).
- `instance_url` (String) URL to the created instace
- `service_id` (String) The service id for the created instance