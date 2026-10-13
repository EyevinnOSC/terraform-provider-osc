package provider

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPollCDNSite(t *testing.T) {
	site := func(status string) *cdnSite { return &cdnSite{SiteID: "s1", Status: status, StatusReason: "vendor said no"} }
	type step struct {
		site *cdnSite
		err  error
	}
	sequence := func(steps ...step) func() (*cdnSite, error) {
		i := 0
		return func() (*cdnSite, error) {
			if i >= len(steps) {
				return steps[len(steps)-1].site, steps[len(steps)-1].err
			}
			i++
			return steps[i-1].site, steps[i-1].err
		}
	}
	blip := errors.New("502")

	// Provisioning, a failed read, then active.
	got, err := pollCDNSite(sequence(step{site("provisioning"), nil}, step{nil, blip}, step{site("active"), nil}),
		"s1", cdnSiteActive, time.Second, time.Millisecond)
	if err != nil || got.Status != "active" {
		t.Fatalf("provisioning then active: got %+v, %v", got, err)
	}

	// Failed ends the wait at once, with the platform's reason.
	if _, err := pollCDNSite(sequence(step{site("provisioning"), nil}, step{site("failed"), nil}),
		"s1", cdnSiteActive, time.Second, time.Millisecond); err == nil || !strings.Contains(err.Error(), "vendor said no") {
		t.Fatalf("failed site: got %v", err)
	}

	// Never active within the timeout names the last status.
	if _, err := pollCDNSite(sequence(step{site("provisioning"), nil}), "s1", cdnSiteActive, 20*time.Millisecond, time.Millisecond); err == nil ||
		!strings.Contains(err.Error(), `"provisioning"`) {
		t.Fatalf("timeout: got %v", err)
	}

	// Deleting until gone.
	if _, err := pollCDNSite(sequence(step{site("deleting"), nil}, step{nil, nil}), "s1", cdnSiteGone, time.Second, time.Millisecond); err != nil {
		t.Fatalf("deleted: got %v", err)
	}

	// A failed read is not taken for gone.
	if _, err := pollCDNSite(sequence(step{nil, blip}), "s1", cdnSiteGone, 20*time.Millisecond, time.Millisecond); err == nil {
		t.Fatal("an error read as gone")
	}
}

func TestCDNSiteOrigin(t *testing.T) {
	cases := []struct {
		model CDNSiteResourceModel
		want  cdnOrigin
	}{
		{CDNSiteResourceModel{ServiceID: types.StringValue("minio-minio"), InstanceName: types.StringValue("media"), AppID: types.StringNull(), PageID: types.StringNull()},
			cdnOrigin{Kind: "instance", ServiceID: "minio-minio", InstanceName: "media"}},
		{CDNSiteResourceModel{ServiceID: types.StringNull(), InstanceName: types.StringNull(), AppID: types.StringValue("app1"), PageID: types.StringNull()},
			cdnOrigin{Kind: "myapp", AppID: "app1"}},
		{CDNSiteResourceModel{ServiceID: types.StringNull(), InstanceName: types.StringNull(), AppID: types.StringNull(), PageID: types.StringValue("page1")},
			cdnOrigin{Kind: "mypage", PageID: "page1"}},
	}
	for _, c := range cases {
		if got := c.model.origin(); got != c.want {
			t.Errorf("origin: got %+v, want %+v", got, c.want)
		}
	}
}
