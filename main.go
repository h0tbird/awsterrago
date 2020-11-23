package main

import (

	// stdlib
	"fmt"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/terraform-providers/terraform-provider-aws/aws"
)

func main() {

	provider := aws.Provider()

	rc := &terraform.ResourceConfig{
		ComputedKeys: []string{},
		Raw:          map[string]interface{}{},
		Config:       map[string]interface{}{},
	}

	provider.Configure(nil, rc)

	dataSources := provider.DataSources()
	for _, v := range dataSources {
		fmt.Println(v.Name)
	}
}
