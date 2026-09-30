package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// Tags group a workspace's service instances, My Apps and My Pages into projects: a
// project is a tag carried by every resource in it. The deploy manager keeps them apart
// from the resources, so they are read and written through their own routes.

// Resource types as the tag API names them.
const (
	tagTypeInstance = "instance"
	tagTypeMyApp    = "myapp"
	tagTypeMyPage   = "mypage"
)

// The limits the platform enforces on tags.
const (
	maxTagLength       = 64
	maxTagsPerResource = 20
)

type resourceTagEntry struct {
	ResourceType string   `json:"resourceType"`
	ServiceID    string   `json:"serviceId"`
	ResourceID   string   `json:"resourceId"`
	Tags         []string `json:"tags"`
	UpdatedAt    string   `json:"updatedAt"`
}

type tagCount struct {
	Tag   string  `json:"tag"`
	Count float64 `json:"count"`
}

// resourceTagsPath is the route of one resource's tags. serviceID is only used for
// instances, whose names are unique per service.
func resourceTagsPath(ctx *osaasclient.Context, resourceType, serviceID, id string) string {
	if resourceType == tagTypeInstance {
		return deployURL(ctx, "/resourcetags/instance/%s/%s", serviceID, id)
	}
	return deployURL(ctx, "/resourcetags/%s/%s", resourceType, id)
}

// listResourceTags returns the tagged resources, filtered by resource type and tag when
// they are not empty. A resource without tags is not listed.
func listResourceTags(ctx *osaasclient.Context, resourceType, tag string) ([]resourceTagEntry, error) {
	q := url.Values{}
	if resourceType != "" {
		q.Set("resourceType", resourceType)
	}
	if tag != "" {
		q.Set("tag", tag)
	}
	u := deployURL(ctx, "/resourcetags")
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var out struct {
		Resources *[]resourceTagEntry `json:"resources"`
	}
	if err := deployDo(ctx, http.MethodGet, u, nil, &out); err != nil {
		return nil, err
	}
	// An empty listing is "resources": []; a body without it is not an answer to trust,
	// since reading no tags makes the plan add them all again.
	if out.Resources == nil {
		return nil, errors.New("the resource tag listing came back without resources")
	}
	return *out.Resources, nil
}

// listTagCounts returns every tag in the workspace with the number of resources carrying it.
func listTagCounts(ctx *osaasclient.Context) ([]tagCount, error) {
	var out struct {
		Tags *[]tagCount `json:"tags"`
	}
	if err := deployDo(ctx, http.MethodGet, deployURL(ctx, "/resourcetags/tags"), nil, &out); err != nil {
		return nil, err
	}
	if out.Tags == nil {
		return nil, errors.New("the tag listing came back without tags")
	}
	return *out.Tags, nil
}

// getResourceTags returns the tags of one resource, empty when it has none.
func getResourceTags(ctx *osaasclient.Context, resourceType, serviceID, id string) ([]string, error) {
	entries, err := listResourceTags(ctx, resourceType, "")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.ResourceID == id && (resourceType != tagTypeInstance || e.ServiceID == serviceID) {
			return e.Tags, nil
		}
	}
	return []string{}, nil
}

// putResourceTags replaces the resource's tags; no tags removes them all.
func putResourceTags(ctx *osaasclient.Context, resourceType, serviceID, id string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	return deployDo(ctx, http.MethodPut, resourceTagsPath(ctx, resourceType, serviceID, id), map[string]interface{}{"tags": tags}, nil)
}

// deleteResourceTags removes the resource's tags. The platform keeps the tags of an
// instance after the instance is deleted, and an instance created later with the same
// name would carry them, so they are removed with the resource.
func deleteResourceTags(ctx *osaasclient.Context, resourceType, serviceID, id string) error {
	err := deployDo(ctx, http.MethodDelete, resourceTagsPath(ctx, resourceType, serviceID, id), nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// ---------------------------------------------------------------- resources

// tagsAttribute is the `tags` attribute shared by the resources that can carry tags.
func tagsAttribute(what string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType: types.StringType,
		Optional:    true,
		Description: fmt.Sprintf("Tags on the %s. A project in OSC is a tag: give every resource in a project the same "+
			"tag to group them. Up to %d tags of 1-%d characters, without leading or trailing spaces; tags that differ "+
			"only in case count as the same tag. When set, Terraform owns all of the %s's tags and removes any added "+
			"elsewhere, and `[]` removes them all. When unset, Terraform leaves its tags alone.",
			what, maxTagsPerResource, maxTagLength, what),
		Validators: []validator.Set{tagsValidator{}},
	}
}

// tagsValidator checks tags against what the platform accepts, and against what it would
// store differently from the configuration: it trims tags and keeps only the first of
// tags that differ in case, which would make the applied state differ from the plan.
type tagsValidator struct{}

func (tagsValidator) Description(_ context.Context) string {
	return fmt.Sprintf("at most %d distinct tags of 1-%d characters without surrounding whitespace", maxTagsPerResource, maxTagLength)
}

func (v tagsValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (tagsValidator) ValidateSet(_ context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	elems := req.ConfigValue.Elements()
	if len(elems) > maxTagsPerResource {
		resp.Diagnostics.AddAttributeError(req.Path, "Too many tags",
			fmt.Sprintf("A resource can carry at most %d tags, got %d.", maxTagsPerResource, len(elems)))
	}
	seen := map[string]string{}
	for _, e := range elems {
		s, ok := e.(types.String)
		if !ok || s.IsUnknown() || s.IsNull() {
			continue
		}
		tag := s.ValueString()
		if msg := invalidTag(tag); msg != "" {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid tag", fmt.Sprintf("Tag %q %s.", tag, msg))
			continue
		}
		key := strings.ToLower(tag)
		if prev, dup := seen[key]; dup {
			resp.Diagnostics.AddAttributeError(req.Path, "Duplicate tag",
				fmt.Sprintf("Tags %q and %q differ only in case, and OSC treats them as the same tag. Keep one of them.", prev, tag))
			continue
		}
		seen[key] = tag
	}
}

// invalidTag says what is wrong with a tag, or returns "" if the platform accepts it as
// written.
func invalidTag(tag string) string {
	switch {
	case strings.TrimSpace(tag) == "":
		return "is empty"
	case strings.TrimSpace(tag) != tag:
		return "has leading or trailing whitespace, which OSC removes"
	case len([]rune(tag)) > maxTagLength:
		return fmt.Sprintf("is longer than %d characters", maxTagLength)
	}
	return ""
}

func tagsToSet(tags []string) types.Set {
	elems := make([]attr.Value, 0, len(tags))
	for _, t := range tags {
		elems = append(elems, types.StringValue(t))
	}
	return types.SetValueMust(types.StringType, elems)
}

func setToTags(s types.Set) []string {
	tags := []string{}
	for _, e := range s.Elements() {
		if v, ok := e.(types.String); ok {
			tags = append(tags, v.ValueString())
		}
	}
	return tags
}

// applyTags writes the planned tags when Terraform manages them and they changed from
// the state; pass a null state on create.
func applyTags(ctx *osaasclient.Context, plan, state types.Set, resourceType, serviceID, id string) diag.Diagnostics {
	var diags diag.Diagnostics
	if plan.IsNull() || plan.IsUnknown() || plan.Equal(state) {
		return diags
	}
	if err := putResourceTags(ctx, resourceType, serviceID, id, setToTags(plan)); err != nil {
		diags.AddAttributeError(path.Root("tags"), "Failed to set tags", fmt.Sprintf("Could not set the tags of %s: %s", id, err.Error()))
	}
	return diags
}

// refreshTags reads the tags into model when Terraform manages them, and leaves an
// unmanaged null alone.
func refreshTags(ctx *osaasclient.Context, model *types.Set, resourceType, serviceID, id string) diag.Diagnostics {
	var diags diag.Diagnostics
	if model.IsNull() {
		return diags
	}
	tags, err := getResourceTags(ctx, resourceType, serviceID, id)
	if err != nil {
		diags.AddAttributeError(path.Root("tags"), "Failed to read tags", fmt.Sprintf("Could not read the tags of %s: %s", id, err.Error()))
		return diags
	}
	*model = tagsToSet(tags)
	return diags
}

// importTags reads the tags of an imported resource. It records them only when there are
// some, so importing an untagged resource leaves them unmanaged.
func importTags(ctx *osaasclient.Context, resourceType, serviceID, id string) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	tags, err := getResourceTags(ctx, resourceType, serviceID, id)
	if err != nil {
		diags.AddAttributeError(path.Root("tags"), "Failed to read tags", fmt.Sprintf("Could not read the tags of %s: %s", id, err.Error()))
		return types.SetNull(types.StringType), diags
	}
	if len(tags) == 0 {
		return types.SetNull(types.StringType), diags
	}
	return tagsToSet(tags), diags
}

// removeTags deletes the tags of a deleted resource. The resource is gone by then, so a
// failure is a warning rather than a failed destroy.
func removeTags(ctx *osaasclient.Context, resourceType, serviceID, id string) diag.Diagnostics {
	var diags diag.Diagnostics
	if err := deleteResourceTags(ctx, resourceType, serviceID, id); err != nil {
		diags.AddWarning("Could not remove tags",
			fmt.Sprintf("%s was deleted, but its tags could not be removed: %s. A resource created later with the same "+
				"name may carry them.", id, err.Error()))
	}
	return diags
}
