---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "osc_eyevinn_schedule_service Resource - osc"
subcategory: ""
description: |-
  A modular service to automatically populate schedules for FAST Engine channels. Uses AWS Dynamo DB as database.
---

# osc_eyevinn_schedule_service (Resource)

A modular service to automatically populate schedules for FAST Engine channels. Uses AWS Dynamo DB as database.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `aws_access_key_id` (String)
- `aws_region` (String)
- `aws_secret_access_key` (String)
- `name` (String) Name of schedule-service
- `table_prefix` (String)

### Read-Only

- `external_ip` (String) The external Ip of the created instance (if available).
- `external_port` (Number) The external Port of the created instance (if available).
- `instance_url` (String) URL to the created instace
- `service_id` (String) The service id for the created instance
