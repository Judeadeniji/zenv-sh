package provider

import (
	"context"

	"github.com/Judeadeniji/zenv-sh/sdk-go/zenv"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure zenvProvider satisfies various provider interfaces.
var _ provider.Provider = &zenvProvider{}

type zenvProvider struct {
	client *zenv.Client
}

type zenvProviderModel struct {
	ApiUrl   types.String `tfsdk:"api_url"`
	Token    types.String `tfsdk:"token"`
	Project  types.String `tfsdk:"project"`
	Env      types.String `tfsdk:"env"`
	VaultKey types.String `tfsdk:"vault_key"`
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &zenvProvider{}
	}
}

func (p *zenvProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "zenv"
}

func (p *zenvProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "The zEnv API URL. Defaults to https://api.zenv.dev",
			},
			"token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Service token for authentication.",
			},
			"project": schema.StringAttribute{
				Required:    true,
				Description: "Project ID.",
			},
			"env": schema.StringAttribute{
				Required:    true,
				Description: "Environment name (e.g. production).",
			},
			"vault_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The cryptographic key used to decrypt secrets.",
			},
		},
	}
}

func (p *zenvProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data zenvProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiUrl := data.ApiUrl.ValueString()
	token := data.Token.ValueString()
	project := data.Project.ValueString()
	env := data.Env.ValueString()
	vaultKey := data.VaultKey.ValueString()

	client, err := zenv.NewClient(
		zenv.WithAPIURL(apiUrl),
		zenv.WithToken(token),
		zenv.WithProjectID(project),
		zenv.WithVaultKey(vaultKey),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to configure zEnv Client",
			"Could not derive cryptographic keys or authenticate: "+err.Error(),
		)
		return
	}

	p.client = client

	providerData := &ProviderData{
		Client: client,
		Env:    env,
	}
	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *zenvProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *zenvProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSecretDataSource,
		NewSecretsDataSource,
	}
}

type ProviderData struct {
	Client *zenv.Client
	Env    string
}
