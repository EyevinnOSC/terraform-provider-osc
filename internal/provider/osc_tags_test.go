package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestTagsValidator(t *testing.T) {
	many := make([]string, maxTagsPerResource+1)
	for i := range many {
		many[i] = "t" + strings.Repeat("x", i)
	}
	cases := []struct {
		name  string
		tags  []string
		error string
	}{
		{"valid", []string{"shop", "Ünïcode", "with space", "a/b", strings.Repeat("a", maxTagLength)}, ""},
		{"none", []string{}, ""},
		{"too long", []string{strings.Repeat("a", maxTagLength+1)}, "Invalid tag"},
		{"empty", []string{""}, "Invalid tag"},
		{"blank", []string{"   "}, "Invalid tag"},
		{"padded", []string{" shop"}, "Invalid tag"},
		{"case duplicate", []string{"Shop", "shop"}, "Duplicate tag"},
		{"too many", many, "Too many tags"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := validator.SetRequest{Path: path.Root("tags"), ConfigValue: tagsToSet(c.tags)}
			var resp validator.SetResponse
			tagsValidator{}.ValidateSet(context.Background(), req, &resp)
			if c.error == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected error: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() || resp.Diagnostics.Errors()[0].Summary() != c.error {
				t.Fatalf("want %q, got %v", c.error, resp.Diagnostics)
			}
		})
	}

	// Unknown values are validated once they are known.
	unknown := types.SetValueMust(types.StringType, []attr.Value{types.StringUnknown(), types.StringValue("shop")})
	var resp validator.SetResponse
	tagsValidator{}.ValidateSet(context.Background(), validator.SetRequest{Path: path.Root("tags"), ConfigValue: unknown}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown element: %v", resp.Diagnostics)
	}
}

// tagListing answers the resource tag listing with entries.
func tagListing(t *testing.T, entries string, requests *[]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requests != nil {
			*requests = append(*requests, r.Method+" "+r.URL.RequestURI())
		}
		if r.Method != http.MethodGet || r.URL.Path != "/resourcetags" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusTeapot)
			return
		}
		_, _ = w.Write([]byte(`{"resources":` + entries + `}`))
	}
}

func TestGetResourceTags(t *testing.T) {
	var requests []string
	ctx := deployManager(t, tagListing(t, `[
		{"resourceType":"instance","serviceId":"minio-minio","resourceId":"shop","tags":["other"],"updatedAt":"x"},
		{"resourceType":"instance","serviceId":"valkey-io-valkey","resourceId":"shop","tags":["Shop","cache"],"updatedAt":"x"}
	]`, &requests))

	tags, err := getResourceTags(ctx, tagTypeInstance, "valkey-io-valkey", "shop")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(tags, ",") != "Shop,cache" {
		t.Errorf("an instance is matched on service and name, got %v", tags)
	}
	if requests[0] != "GET /resourcetags?resourceType=instance" {
		t.Errorf("request %q", requests[0])
	}

	tags, err = getResourceTags(ctx, tagTypeInstance, "valkey-io-valkey", "web")
	if err != nil || tags == nil || len(tags) != 0 {
		t.Errorf("an untagged resource has no tags, got %v, %v", tags, err)
	}
}

func TestListResourceTagsRejectsEmptyAnswer(t *testing.T) {
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	if _, err := getResourceTags(ctx, tagTypeMyApp, "", "shop"); err == nil {
		t.Fatal("a listing without resources must not read as no tags")
	}
	if _, err := listTagCounts(ctx); err == nil {
		t.Fatal("a tag listing without tags must not read as none")
	}
}

func TestListResourceTagsFilters(t *testing.T) {
	var requests []string
	ctx := deployManager(t, tagListing(t, `[]`, &requests))
	if _, err := listResourceTags(ctx, tagTypeMyPage, "My Project"); err != nil {
		t.Fatal(err)
	}
	if requests[0] != "GET /resourcetags?resourceType=mypage&tag=My+Project" {
		t.Errorf("request %q", requests[0])
	}
}

func TestPutAndDeleteResourceTags(t *testing.T) {
	type call struct{ method, path, body string }
	var calls []call
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		calls = append(calls, call{r.Method, r.URL.EscapedPath(), string(b)})
		if strings.Contains(r.URL.Path, "gone") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"reason":"My App 'gone' not found"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))

	if err := putResourceTags(ctx, tagTypeInstance, "valkey-io-valkey", "shop", []string{"a b"}); err != nil {
		t.Fatal(err)
	}
	if err := putResourceTags(ctx, tagTypeMyApp, "", "shop", nil); err != nil {
		t.Fatal(err)
	}
	if err := deleteResourceTags(ctx, tagTypeMyPage, "", "docs"); err != nil {
		t.Fatal(err)
	}
	if err := deleteResourceTags(ctx, tagTypeMyApp, "", "gone"); err != nil {
		t.Errorf("deleting the tags of a resource that is gone should succeed: %v", err)
	}

	want := []call{
		{"PUT", "/resourcetags/instance/valkey-io-valkey/shop", `{"tags":["a b"]}`},
		{"PUT", "/resourcetags/myapp/shop", `{"tags":[]}`},
		{"DELETE", "/resourcetags/mypage/docs", ""},
		{"DELETE", "/resourcetags/myapp/gone", ""},
	}
	if len(calls) != len(want) {
		t.Fatalf("calls %+v", calls)
	}
	for i, c := range calls {
		var got, exp interface{}
		_ = json.Unmarshal([]byte(c.body), &got)
		_ = json.Unmarshal([]byte(want[i].body), &exp)
		if c.method != want[i].method || c.path != want[i].path || !jsonEqual(got, exp) {
			t.Errorf("call %d = %+v, want %+v", i, c, want[i])
		}
	}
}

func jsonEqual(a, b interface{}) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func TestApplyTagsOnlyWhenChanged(t *testing.T) {
	var puts atomic.Int32
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		puts.Add(1)
		_, _ = w.Write([]byte(`{}`))
	}))
	null := types.SetNull(types.StringType)
	a := tagsToSet([]string{"a", "b"})
	b := tagsToSet([]string{"b", "a"})

	for _, c := range []struct {
		name        string
		plan, state types.Set
		want        int32
	}{
		{"unmanaged", null, a, 0},
		{"unknown", types.SetUnknown(types.StringType), a, 0},
		{"unchanged in another order", b, a, 0},
		{"create", a, null, 1},
		{"clear", tagsToSet(nil), a, 1},
	} {
		puts.Store(0)
		if d := applyTags(ctx, c.plan, c.state, tagTypeMyApp, "", "shop"); d.HasError() {
			t.Fatalf("%s: %v", c.name, d)
		}
		if puts.Load() != c.want {
			t.Errorf("%s: %d requests, want %d", c.name, puts.Load(), c.want)
		}
	}
}

// readWithTags runs Read on a page whose state holds tags, or null tags when tags is nil.
func readWithTags(t *testing.T, r resource.Resource, tags []string) *resource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	state := tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)}
	for k, v := range map[string]string{"id": "docs", "name": "docs"} {
		if d := state.SetAttribute(ctx, path.Root(k), v); d.HasError() {
			t.Fatalf("set %s: %v", k, d)
		}
	}
	value := types.SetNull(types.StringType)
	if tags != nil {
		value = tagsToSet(tags)
	}
	if d := state.SetAttribute(ctx, path.Root("tags"), value); d.HasError() {
		t.Fatalf("set tags: %v", d)
	}
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	return resp
}

func stateTags(t *testing.T, resp *resource.ReadResponse) []string {
	t.Helper()
	var tags types.Set
	if d := resp.State.GetAttribute(context.Background(), path.Root("tags"), &tags); d.HasError() {
		t.Fatal(d)
	}
	if tags.IsNull() {
		return nil
	}
	out := setToTags(tags)
	sort.Strings(out)
	return out
}

func TestMyPageReadTags(t *testing.T) {
	fastReads(t)
	var tagReads atomic.Int32
	handler := func(listing http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/resourcetags") {
				tagReads.Add(1)
				listing(w, r)
				return
			}
			_, _ = w.Write([]byte(`{"id":"docs","name":"docs","status":"live"}`))
		}
	}
	listing := tagListing(t, `[{"resourceType":"mypage","resourceId":"docs","tags":["web","Shop"],"updatedAt":"x"}]`, nil)

	// Managed tags are read, including ones added outside Terraform.
	resp := readWithTags(t, &MyPageResource{osaasContext: deployManager(t, handler(listing))}, []string{"shop"})
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got := strings.Join(stateTags(t, resp), ","); got != "Shop,web" {
		t.Errorf("tags %q", got)
	}

	// Unmanaged tags are not read at all.
	tagReads.Store(0)
	resp = readWithTags(t, &MyPageResource{osaasContext: deployManager(t, handler(listing))}, nil)
	if resp.Diagnostics.HasError() || stateTags(t, resp) != nil || tagReads.Load() != 0 {
		t.Errorf("unmanaged tags were read: %v %v", stateTags(t, resp), resp.Diagnostics)
	}

	// A failed tag listing fails the read and keeps the resource.
	resp = readWithTags(t, &MyPageResource{osaasContext: deployManager(t, handler(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))}, []string{"shop"})
	if !resp.Diagnostics.HasError() || resp.State.Raw.IsNull() {
		t.Errorf("want an error with the page kept, got %v", resp.Diagnostics)
	}
}
