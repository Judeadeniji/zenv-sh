package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSecretDataSource(t *testing.T) {
	url, vaultKey, teardown := setupMockAPI(t)
	defer teardown()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "zenv" {
	api_url   = "%s"
	token     = "test-token"
	project   = "prj_test"
	env       = "env_1"
	vault_key = "%s"
}

data "zenv_secret" "api_key" {
	name = "API_KEY"
}

data "zenv_secret" "db_pass" {
	name = "DB_PASS"
}
`, url, vaultKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zenv_secret.api_key", "value", "secret-api-key"),
					resource.TestCheckResourceAttr("data.zenv_secret.db_pass", "value", "secret-db-pass"),
				),
			},
		},
	})
}
