---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "osc_birme_osc_postgresql Resource - osc"
subcategory: ""
description: |-
  Unlock the full potential of your data with the PostgreSQL OSC image, seamlessly integrated for use in Eyevinn Open Source Cloud. Experience robust scalability, high security, and unmatched extensibility.
---

# osc_birme_osc_postgresql (Resource)

Unlock the full potential of your data with the PostgreSQL OSC image, seamlessly integrated for use in Eyevinn Open Source Cloud. Experience robust scalability, high security, and unmatched extensibility.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of osc-postgresql
- `postgres_password` (String)

### Optional

- `postgres_db` (String)
- `postgres_init_db_args` (String)
- `postgres_user` (String)

### Read-Only

- `external_ip` (String) The external Ip of the created instance (if available).
- `external_port` (Number) The external Port of the created instance (if available).
- `instance_url` (String) URL to the created instace
- `service_id` (String) The service id for the created instance
