# Every resource in a project, which in OSC is a tag.
data "osc_resource_tags" "project" {
  tag = "my-project"
}

output "project_instances" {
  value = [
    for r in data.osc_resource_tags.project.resources : "${r.service_id}/${r.resource_id}"
    if r.resource_type == "instance"
  ]
}

# All projects in the workspace, with how many resources each has.
data "osc_resource_tags" "all" {}

output "projects" {
  value = { for t in data.osc_resource_tags.all.tags : t.tag => t.count }
}
