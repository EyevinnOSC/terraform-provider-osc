package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"

	osaasclient "github.com/EyevinnOSC/client-go"

	"terraform-provider-osc/internal/catalog"
)

// The mirror is fetched at most once per provider process.
var (
	mirrorOnce sync.Once
	mirror     *catalog.Mirror
	mirrorErr  error
)

func loadMirror(ctx context.Context) (*catalog.Mirror, error) {
	mirrorOnce.Do(func() {
		mirror, mirrorErr = catalog.Fetch(ctx, catalog.MirrorURL())
	})
	return mirror, mirrorErr
}

// resetMirrorForTest clears the cache so tests can point at a different URL.
func resetMirrorForTest() {
	mirrorOnce = sync.Once{}
	mirror, mirrorErr = nil, nil
}

// fromMirror converts a mirror entry into the shape the rest of the provider uses.
func fromMirror(s *catalog.Service) *catalogService {
	c := &catalogService{
		ServiceId:              s.ServiceID,
		ApiUrl:                 s.APIURL,
		ServiceInstanceOptions: s.Options,
		ServiceType:            s.ServiceType,
		Status:                 s.Status,
		Metadata: osaasclient.ServiceMetadata{
			Title:            s.Title,
			Description:      s.Description,
			Category:         s.Category,
			DocumentationUrl: s.DocumentationURL,
			RepoUrl:          s.RepoURL,
		},
	}
	return c
}

// mirrorService looks a service up in the mirror. It returns the service, a list of
// suggestions when it is missing, and an error when the mirror itself is unavailable.
func mirrorService(ctx context.Context, serviceID string) (*catalogService, []string, error) {
	m, err := loadMirror(ctx)
	if err != nil {
		return nil, nil, err
	}
	if s := m.Find(serviceID); s != nil {
		return fromMirror(s), nil, nil
	}
	return nil, m.Suggest(serviceID, 5), nil
}

// unknownServiceDetail builds the detail text for an unknown service id.
func unknownServiceDetail(serviceID string, suggestions []string, mirrorErr error) string {
	var b strings.Builder
	if mirrorErr != nil {
		fmt.Fprintf(&b, "The workspace is not subscribed to service %q and the catalog mirror could not be read to check "+
			"whether it exists (%s). Service ids have the form {contributor}-{name} and cannot be guessed.", serviceID, mirrorErr.Error())
	} else {
		fmt.Fprintf(&b, "Service %q is not in the OSC catalog. Service ids have the form {contributor}-{name} and cannot be guessed.", serviceID)
	}
	if len(suggestions) > 0 {
		fmt.Fprintf(&b, " Did you mean: %s?", strings.Join(suggestions, ", "))
	}
	b.WriteString(" Every service has a guide page under " + catalog.GuidesBaseURL + " listing its parameters.")
	return b.String()
}
