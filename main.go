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
	diags := provider.Configure(ctx, &terraform.ResourceConfig{
		Config: map[string]interface{}{
			"region": "us-east-2",
		},
	})

	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error configuring the provider: %s", d.Summary)
			}
		}
	}

	// Configure the resource
	resourceConfig := &terraform.ResourceConfig{
		Config: map[string]interface{}{
			"bucket": "my-nice-bucket",
		},
	}

	stateBefore := &terraform.InstanceState{}
	AWSS3Bucket := provider.ResourcesMap["aws_s3_bucket"]

	// Diff
	instanceDiff, err := AWSS3Bucket.Diff(ctx, stateBefore, resourceConfig, provider.Meta())
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// Apply
	stateAfter, diags := AWSS3Bucket.Apply(ctx, stateBefore, instanceDiff, provider.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error configuring S3 bucket: %s", d.Summary)
			}
		}
	}

	fmt.Println("\nState before")
	fmt.Println("------------")
	fmt.Println(stateBefore)

	fmt.Println("\nDiff")
	fmt.Println("------------")
	fmt.Println(instanceDiff)

	fmt.Println("\nState after")
	fmt.Println("-----------")
	fmt.Println(stateAfter)
}
