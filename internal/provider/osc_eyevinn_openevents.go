package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource              = &eyevinnopenevents{}
	_ resource.ResourceWithConfigure = &eyevinnopenevents{}
)

func Neweyevinnopenevents() resource.Resource {
	return &eyevinnopenevents{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnopenevents)
}

func (r *eyevinnopenevents) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	osaasContext, ok := req.ProviderData.(*osaasclient.Context)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *OscClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.osaasContext = osaasContext
}

// eyevinnopenevents is the resource implementation.
type eyevinnopenevents struct {
	osaasContext *osaasclient.Context
}

type eyevinnopeneventsModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Nextauthsecret         types.String       `tfsdk:"nextauth_secret"`
	Stripesecretkey         types.String       `tfsdk:"stripe_secret_key"`
	Stripepublishablekey         types.String       `tfsdk:"stripe_publishable_key"`
	Stripewebhooksecret         types.String       `tfsdk:"stripe_webhook_secret"`
	S3endpoint         types.String       `tfsdk:"s3_endpoint"`
	S3region         types.String       `tfsdk:"s3_region"`
	S3bucketname         types.String       `tfsdk:"s3_bucket_name"`
	S3accesskeyid         types.String       `tfsdk:"s3_access_key_id"`
	S3secretaccesskey         types.String       `tfsdk:"s3_secret_access_key"`
	Smtphost         types.String       `tfsdk:"smtp_host"`
	Smtpport         types.String       `tfsdk:"smtp_port"`
	Smtpuser         types.String       `tfsdk:"smtp_user"`
	Smtppassword         types.String       `tfsdk:"smtp_password"`
	Fromemail         types.String       `tfsdk:"from_email"`
	Sitename         types.String       `tfsdk:"site_name"`
	Siteurl         types.String       `tfsdk:"site_url"`
	Databaseurl         types.String       `tfsdk:"database_url"`
}

func (r *eyevinnopenevents) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_openevents"
}

// Schema defines the schema for the resource.
func (r *eyevinnopenevents) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Streamline your event management with OpenEvents, the comprehensive platform for dynamic event planning and seamless ticketing. Enhance attendee experience with real-time tracking, multiple ticketing options, and secure payment integration, all effortlessly managed through an intuitive dashboard.`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"service_id": schema.StringAttribute{
				Computed: true,
				Description: "The service id for the created instance",
			},
			"external_ip": schema.StringAttribute{
				Computed: true,
				Description: "The external Ip of the created instance (if available).",
			},
			"external_port": schema.Int32Attribute{
				Computed: true,
				Description: "The external Port of the created instance (if available).",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of openevents",
			},
			"nextauth_secret": schema.StringAttribute{
				Required: true,
				Description: "Secret key used by NextAuth.js for encrypting JWT tokens and session data",
			},
			"stripe_secret_key": schema.StringAttribute{
				Required: true,
				Description: "Stripe secret API key for processing online payments",
			},
			"stripe_publishable_key": schema.StringAttribute{
				Required: true,
				Description: "Stripe publishable API key for client-side payment form integration",
			},
			"stripe_webhook_secret": schema.StringAttribute{
				Required: true,
				Description: "Stripe webhook endpoint secret for verifying payment event notifications",
			},
			"s3_endpoint": schema.StringAttribute{
				Required: true,
				Description: "S3-compatible storage endpoint URL for file uploads",
			},
			"s3_region": schema.StringAttribute{
				Required: true,
				Description: "AWS region or S3-compatible storage region setting",
			},
			"s3_bucket_name": schema.StringAttribute{
				Required: true,
				Description: "Name of the S3 bucket for storing uploaded files",
			},
			"s3_access_key_id": schema.StringAttribute{
				Required: true,
				Description: "Access key ID for S3-compatible storage authentication",
			},
			"s3_secret_access_key": schema.StringAttribute{
				Required: true,
				Description: "Secret access key for S3-compatible storage authentication",
			},
			"smtp_host": schema.StringAttribute{
				Optional: true,
				Description: "SMTP server hostname for sending emails",
			},
			"smtp_port": schema.StringAttribute{
				Optional: true,
				Description: "SMTP server port number for email delivery",
			},
			"smtp_user": schema.StringAttribute{
				Optional: true,
				Description: "Username for SMTP server authentication",
			},
			"smtp_password": schema.StringAttribute{
				Optional: true,
				Description: "Password for SMTP server authentication",
			},
			"from_email": schema.StringAttribute{
				Optional: true,
				Description: "Email address used as the sender for outgoing emails",
			},
			"site_name": schema.StringAttribute{
				Optional: true,
				Description: "Name of the event platform displayed in the application",
			},
			"site_url": schema.StringAttribute{
				Optional: true,
				Description: "Base URL of the deployed application",
			},
			"database_url": schema.StringAttribute{
				Required: true,
				Description: "PostgreSQL database connection string",
			},
		},
	}
}

func (r *eyevinnopenevents) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnopeneventsModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-openevents")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-openevents", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"nextauthSecret": plan.Nextauthsecret.ValueString(),
		"stripeSecretKey": plan.Stripesecretkey.ValueString(),
		"stripePublishableKey": plan.Stripepublishablekey.ValueString(),
		"stripeWebhookSecret": plan.Stripewebhooksecret.ValueString(),
		"s3Endpoint": plan.S3endpoint.ValueString(),
		"s3Region": plan.S3region.ValueString(),
		"s3BucketName": plan.S3bucketname.ValueString(),
		"s3AccessKeyId": plan.S3accesskeyid.ValueString(),
		"s3SecretAccessKey": plan.S3secretaccesskey.ValueString(),
		"smtpHost": plan.Smtphost.ValueString(),
		"smtpPort": plan.Smtpport.ValueString(),
		"smtpUser": plan.Smtpuser.ValueString(),
		"smtpPassword": plan.Smtppassword.ValueString(),
		"fromEmail": plan.Fromemail.ValueString(),
		"siteName": plan.Sitename.ValueString(),
		"siteUrl": plan.Siteurl.ValueString(),
		"databaseUrl": plan.Databaseurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-openevents", instance["name"].(string), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
		return
	}

	var externalPort = 0
	var externalIp = ""
	if len(ports) > 0 {
		port := ports[0]
		externalPort = port.ExternalPort
		externalIp = port.ExternalIP
	}


	// Update the state with the actual data returned from the API
	state := eyevinnopeneventsModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-openevents"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Nextauthsecret: plan.Nextauthsecret,
		Stripesecretkey: plan.Stripesecretkey,
		Stripepublishablekey: plan.Stripepublishablekey,
		Stripewebhooksecret: plan.Stripewebhooksecret,
		S3endpoint: plan.S3endpoint,
		S3region: plan.S3region,
		S3bucketname: plan.S3bucketname,
		S3accesskeyid: plan.S3accesskeyid,
		S3secretaccesskey: plan.S3secretaccesskey,
		Smtphost: plan.Smtphost,
		Smtpport: plan.Smtpport,
		Smtpuser: plan.Smtpuser,
		Smtppassword: plan.Smtppassword,
		Fromemail: plan.Fromemail,
		Sitename: plan.Sitename,
		Siteurl: plan.Siteurl,
		Databaseurl: plan.Databaseurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnopenevents) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnopenevents) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnopenevents) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnopeneventsModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-openevents")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-openevents", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
