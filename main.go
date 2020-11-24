package main

import (

	// stdlib
	"context"
	"fmt"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/terraform-providers/terraform-provider-aws/aws"
)

func main() {

	ctx := context.Background()
	provider := aws.Provider()

	// Configure the provider
	provider.Configure(ctx, &terraform.ResourceConfig{
		Config: map[string]interface{}{
			"region": "us-east-2",
		},
	})

	// List all data sources
	fmt.Println("--[Data sources]----------------")
	dataSources := provider.DataSources()
	for _, v := range dataSources {
		fmt.Println(v.Name)
	}

	// List all resources
	fmt.Println("--[Resources]----------------")
	resources := provider.Resources()
	for _, v := range resources {
		fmt.Println(v.Name)
	}
}
