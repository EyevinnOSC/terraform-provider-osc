package provider

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// Backups of database instances are run by the deploy manager, which stores them in
// OSC-managed object storage. A backup policy per instance says whether and when the
// platform backs it up; the platform keeps a policy once it has been written, so turning
// a schedule off is the only way to remove it.

// backupServices are the services the platform can back up.
var backupServices = []string{
	"birme-osc-postgresql",
	"linuxserver-docker-mariadb",
	"valkey-io-valkey",
	"clickhouse-clickhouse",
	"apache-couchdb",
	"go-gitea-gitea",
}

const (
	defaultBackupSchedule  = "0 2 * * *"
	defaultBackupRetention = 30
)

func isBackupService(serviceID string) bool {
	for _, s := range backupServices {
		if s == serviceID {
			return true
		}
	}
	return false
}

type backupPolicy struct {
	ServiceID           string  `json:"serviceId"`
	InstanceName        string  `json:"instanceName"`
	Enabled             bool    `json:"enabled"`
	Schedule            string  `json:"schedule"`
	RetentionDays       float64 `json:"retentionDays"`
	LastBackupAt        float64 `json:"lastBackupAt"`
	LastAttemptAt       float64 `json:"lastAttemptAt"`
	LastError           string  `json:"lastError"`
	CredentialStatus    string  `json:"credentialStatus"`
	CredentialCheckedAt float64 `json:"credentialCheckedAt"`
}

type backup struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	CreatedAt float64 `json:"createdAt"`
	Source    string  `json:"source"`
	Error     string  `json:"error"`
}

// findBackupPolicy returns the policy written for the instance, or nil if none has been.
// It reads the tenant's list of policies: the policy of a single instance answers with
// defaults for any name, whether or not a policy or even the instance exists.
func findBackupPolicy(ctx *osaasclient.Context, serviceID, instanceName string) (*backupPolicy, error) {
	var out []backupPolicy
	if err := deployDo(ctx, http.MethodGet, deployURL(ctx, "/mybackups/policies"), nil, &out); err != nil {
		return nil, err
	}
	// An empty listing is "[]"; an empty body is not an answer to trust.
	if out == nil {
		return nil, errors.New("the backup policy listing came back empty")
	}
	for i := range out {
		if out[i].ServiceID == serviceID && out[i].InstanceName == instanceName {
			return &out[i], nil
		}
	}
	return nil, nil
}

// putBackupPolicy writes the whole policy. The platform keeps any field left out, so all
// of them are always sent.
func putBackupPolicy(ctx *osaasclient.Context, serviceID, instanceName string, enabled bool, schedule string, retentionDays int64) error {
	body := map[string]interface{}{"enabled": enabled, "schedule": schedule, "retentionDays": retentionDays}
	err := deployDo(ctx, http.MethodPut, deployURL(ctx, "/mybackups/%s/%s/policy", serviceID, instanceName), body, nil)
	var ae *apiError
	if errors.As(err, &ae) && ae.StatusCode == http.StatusPaymentRequired {
		return fmt.Errorf("scheduled backups require a paid OSC plan: %w", err)
	}
	return err
}

// disableBackupPolicy stops scheduled backups and keeps the schedule and retention, as
// the platform has no way to remove a policy. Existing backups are kept.
func disableBackupPolicy(ctx *osaasclient.Context, serviceID, instanceName string) error {
	return deployDo(ctx, http.MethodPut, deployURL(ctx, "/mybackups/%s/%s/policy", serviceID, instanceName),
		map[string]interface{}{"enabled": false}, nil)
}

// listBackups returns the instance's backups, newest first.
func listBackups(ctx *osaasclient.Context, serviceID, instanceName string) ([]backup, error) {
	var out []backup
	if err := deployDo(ctx, http.MethodGet, deployURL(ctx, "/mybackups/%s/%s", serviceID, instanceName), nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, errors.New("the backup listing came back empty")
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

// backupStatus is the status of a backup as the documentation names it. OSC reports some
// finished backups with the Kubernetes job condition that precedes the final one:
// SuccessCriteriaMet for a backup that completed and FailureTarget for one that failed.
func backupStatus(status string) string {
	switch status {
	case "SuccessCriteriaMet":
		return "Complete"
	case "FailureTarget":
		return "Failed"
	}
	return status
}

// unixTime formats Unix seconds as RFC 3339, or "" for zero.
func unixTime(seconds float64) string {
	if seconds <= 0 {
		return ""
	}
	return time.Unix(int64(seconds), 0).UTC().Format(time.RFC3339)
}

// ------------------------------------------------------------------ cron

// The platform stores any string as a schedule, and one its scheduler cannot parse means
// no backups are taken, so schedules are checked here. Only the standard five-field
// format with numbers is accepted, which is what the OSC documentation describes; month
// and day names and macros such as @daily are not documented and are rejected.
var cronFields = []struct {
	name     string
	min, max int
}{
	{"minute", 0, 59},
	{"hour", 0, 23},
	{"day of month", 1, 31},
	{"month", 1, 12},
	{"day of week", 0, 6},
}

// invalidCron says what is wrong with a cron expression, or returns "" if it is valid.
func invalidCron(expr string) string {
	fields := strings.Fields(expr)
	if len(fields) != len(cronFields) {
		return fmt.Sprintf("must have five fields (minute hour day-of-month month day-of-week), got %d", len(fields))
	}
	if strings.TrimSpace(expr) != expr {
		return "has leading or trailing whitespace"
	}
	for i, f := range fields {
		spec := cronFields[i]
		for _, part := range strings.Split(f, ",") {
			if msg := invalidCronPart(part, spec.min, spec.max); msg != "" {
				return fmt.Sprintf("%s field %q: %s", spec.name, f, msg)
			}
		}
	}
	return ""
}

func invalidCronPart(part string, min, max int) string {
	rangePart, step, hasStep := strings.Cut(part, "/")
	if hasStep {
		n, err := strconv.Atoi(step)
		if err != nil || n < 1 {
			return fmt.Sprintf("step %q must be a positive number", step)
		}
	}
	if rangePart == "*" {
		return ""
	}
	lo, hi, isRange := strings.Cut(rangePart, "-")
	a, err := cronNumber(lo, min, max)
	if err != "" {
		return err
	}
	if !isRange {
		return ""
	}
	b, err := cronNumber(hi, min, max)
	if err != "" {
		return err
	}
	if b < a {
		return fmt.Sprintf("range %s is backwards", rangePart)
	}
	return ""
}

func cronNumber(s string, min, max int) (int, string) {
	n, err := strconv.Atoi(s)
	if err != nil || s == "" || strings.ContainsAny(s, "+-") {
		return 0, fmt.Sprintf("%q is not a number; use numbers, *, ranges (1-5), lists (1,3) and steps (*/15)", s)
	}
	if n < min || n > max {
		return 0, fmt.Sprintf("%d is outside %d-%d", n, min, max)
	}
	return n, ""
}
