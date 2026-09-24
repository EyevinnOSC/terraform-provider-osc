package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplyMailbox(t *testing.T) {
	// The shape GET /mymail answers with.
	body := `{"tenantId":"acme","email":"acme@users.osaas.io",
		"smtp":{"server":"mail.osaas.io","port":587,"encryption":"STARTTLS"},
		"imap":{"server":"mail.osaas.io","port":993,"encryption":"SSL/TLS"},
		"createdAt":"2026-09-24T08:00:00.000Z"}`
	var m mailbox
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatal(err)
	}
	model := MailboxResourceModel{Password: types.StringValue("correct horse battery")}
	applyMailbox(&m, &model)

	if model.ID.ValueString() != "acme" || model.Email.ValueString() != "acme@users.osaas.io" {
		t.Errorf("id/email: %+v", model)
	}
	if model.SMTPHost.ValueString() != "mail.osaas.io" || model.SMTPPort.ValueInt64() != 587 || model.SMTPEncryption.ValueString() != "STARTTLS" {
		t.Errorf("smtp: %+v", model)
	}
	if model.IMAPPort.ValueInt64() != 993 || model.CreatedAt.ValueString() != "2026-09-24T08:00:00.000Z" {
		t.Errorf("imap/created: %+v", model)
	}
	// The platform never returns the password, so it must survive a refresh.
	if model.Password.ValueString() != "correct horse battery" {
		t.Errorf("password was overwritten: %q", model.Password.ValueString())
	}
}
