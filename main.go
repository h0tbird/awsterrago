package main

//----------------------------------------------------------------
//
//----------------------------------------------------------------

import (

	// stdlib
	"context"
	"fmt"
	"os"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/terraform-providers/terraform-provider-aws/aws"
)

//----------------------------------------------------------------
//
//----------------------------------------------------------------

func main() {

	ctx := context.Background()
	provider := aws.Provider()

	// Configure the provider
	provider.Configure(ctx, &terraform.ResourceConfig{
		Config: map[string]interface{}{
			"region": "us-east-2",
		},
	})

	// Configure the resource
	resourceConfig := &terraform.ResourceConfig{
		Config: map[string]interface{}{
			"bucket": "my-nice-bucket",
		},
	}

	instanceState := &terraform.InstanceState{}
	AWSS3Bucket := provider.ResourcesMap["aws_s3_bucket"]

	// Diff
	instanceDiff, err := AWSS3Bucket.Diff(ctx, instanceState, resourceConfig, provider.Meta())
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// Apply
	istate, diags := AWSS3Bucket.Apply(ctx, instanceState, instanceDiff, provider.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error configuring S3 bucket: %s", d.Summary)
			}
		}
	}

	fmt.Println(istate)
}
