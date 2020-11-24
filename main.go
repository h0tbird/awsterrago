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

	rc := &terraform.ResourceConfig{
		ComputedKeys: []string{},
		Raw:          map[string]interface{}{},
		Config: map[string]interface{}{
			"region": "us-east-2",
		},
	}

	provider.Configure(ctx, rc)

	dataSources := provider.DataSources()
	for _, v := range dataSources {
		fmt.Println(v.Name)
	}
}
