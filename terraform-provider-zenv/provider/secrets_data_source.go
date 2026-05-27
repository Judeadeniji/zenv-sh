package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &secretsDataSource{}

type secretsDataSource struct {
	client *ProviderData
}

type secretsDataSourceModel struct {
	Secrets types.Map `tfsdk:"secrets"`
}

func NewSecretsDataSource() datasource.DataSource {
	return &secretsDataSource{}
}

func (d *secretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secrets"
}

func (d *secretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Bulk fetches and decrypts all secrets for the configured zEnv project and environment.",
		Attributes: map[string]schema.Attribute{
			"secrets": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Sensitive:   true,
				Description: "A map of all decrypted secrets.",
			},
		},
	}
}

func (d *secretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected ProviderData type", "")
		return
	}
	d.client = data
}

func (d *secretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data secretsDataSourceModel

	if d.client == nil || d.client.Client == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider client is not initialized.")
		return
	}

	secretsMap, err := d.client.Client.FetchAllSecrets(d.client.Env)
	if err != nil {
		resp.Diagnostics.AddError("Failed to bulk fetch secrets", err.Error())
		return
	}

	elements := make(map[string]types.String, len(secretsMap))
	for k, v := range secretsMap {
		elements[k] = types.StringValue(v)
	}

	mapVal, diags := types.MapValueFrom(ctx, types.StringType, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Secrets = mapVal
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
