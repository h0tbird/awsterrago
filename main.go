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

	state0 := &terraform.InstanceState{}
	AWSS3Bucket := provider.ResourcesMap["aws_s3_bucket"]

	fmt.Println("\nState before")
	fmt.Println("------------")
	fmt.Println(state0)

	//-------------------------------------------------------------------------
	// Round-1
	//-------------------------------------------------------------------------

	// Diff-1
	diff1, err := AWSS3Bucket.Diff(ctx, state0, resourceConfig, provider.Meta())
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	fmt.Println("\nDiff-1")
	fmt.Println("------")
	fmt.Println(diff1)

	// Apply-1
	if diff1 == nil {
		os.Exit(0)
	}

	state1, diags := AWSS3Bucket.Apply(ctx, state0, diff1, provider.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error configuring S3 bucket: %s", d.Summary)
			}
		}
	}

	fmt.Println("\nState after apply-1")
	fmt.Println("-------------------")
	fmt.Println(state1)

	//-------------------------------------------------------------------------
	// Round-2
	//-------------------------------------------------------------------------

	// Diff-2
	diff2, err := AWSS3Bucket.Diff(ctx, state1, resourceConfig, provider.Meta())
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	fmt.Println("\nDiff-2")
	fmt.Println("------")
	fmt.Println(diff2)

	// Apply-2
	if diff2 == nil {
		os.Exit(0)
	}

	state2, diags := AWSS3Bucket.Apply(ctx, state1, diff2, provider.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error configuring S3 bucket: %s", d.Summary)
			}
		}
	}

	fmt.Println("\nState after apply-2")
	fmt.Println("-------------------")
	fmt.Println(state2)
}
