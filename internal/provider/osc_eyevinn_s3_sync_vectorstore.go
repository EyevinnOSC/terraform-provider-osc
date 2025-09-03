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
	_ resource.Resource              = &eyevinns3syncvectorstore{}
	_ resource.ResourceWithConfigure = &eyevinns3syncvectorstore{}
)

func Neweyevinns3syncvectorstore() resource.Resource {
	return &eyevinns3syncvectorstore{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinns3syncvectorstore)
}

func (r *eyevinns3syncvectorstore) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinns3syncvectorstore is the resource implementation.
type eyevinns3syncvectorstore struct {
	osaasContext *osaasclient.Context
}

type eyevinns3syncvectorstoreModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Cmdlineargs         types.String       `tfsdk:"cmd_line_args"`
	Openaiapikey         types.String       `tfsdk:"openai_api_key"`
	Purpose         types.String       `tfsdk:"purpose"`
	Awsregion         types.String       `tfsdk:"aws_region"`
	S3endpoint         types.String       `tfsdk:"s3_endpoint"`
	Awsaccesskeyid         types.String       `tfsdk:"aws_access_key_id"`
	Awssecretaccesskey         types.String       `tfsdk:"aws_secret_access_key"`
}

func (r *eyevinns3syncvectorstore) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_s3_sync_vectorstore"
}

// Schema defines the schema for the resource.
func (r *eyevinns3syncvectorstore) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly sync your AWS S3 bucket with an OpenAI vector store using our tool. Streamline your data integration and boost AI capabilities instantly. Ideal for seamless data management!`,
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
				Description: "Name of s3-sync-vectorstore",
			},
			"cmd_line_args": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"openai_api_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"purpose": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_access_key_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_secret_access_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinns3syncvectorstore) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinns3syncvectorstoreModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-s3-sync-vectorstore")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-s3-sync-vectorstore", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"cmdLineArgs": plan.Cmdlineargs.ValueString(),
		"OpenaiApiKey": plan.Openaiapikey.ValueString(),
		"Purpose": plan.Purpose.ValueString(),
		"AwsRegion": plan.Awsregion.ValueString(),
		"S3Endpoint": plan.S3endpoint.ValueString(),
		"AwsAccessKeyId": plan.Awsaccesskeyid.ValueString(),
		"AwsSecretAccessKey": plan.Awssecretaccesskey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-s3-sync-vectorstore", instance["name"].(string), serviceAccessToken)
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
	state := eyevinns3syncvectorstoreModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-s3-sync-vectorstore"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Cmdlineargs: plan.Cmdlineargs,
		Openaiapikey: plan.Openaiapikey,
		Purpose: plan.Purpose,
		Awsregion: plan.Awsregion,
		S3endpoint: plan.S3endpoint,
		Awsaccesskeyid: plan.Awsaccesskeyid,
		Awssecretaccesskey: plan.Awssecretaccesskey,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinns3syncvectorstore) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinns3syncvectorstore) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinns3syncvectorstore) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinns3syncvectorstoreModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-s3-sync-vectorstore")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-s3-sync-vectorstore", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
