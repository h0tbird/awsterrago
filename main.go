package main

//----------------------------------------------------------------
// Imports
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
// Main
//----------------------------------------------------------------

func main() {

	ctx := context.Background()
	p := aws.Provider()

	// Configure the provider
	diags := p.Configure(ctx, &terraform.ResourceConfig{
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

	state0 := &terraform.InstanceState{ID: "my-nice-bucket"}
	AWSS3Bucket := p.ResourcesMap["aws_s3_bucket"]

	//-------------------------------------------------------------------------
	// Round-1
	//-------------------------------------------------------------------------

	// Refresh-1
	state1, diags := AWSS3Bucket.RefreshWithoutUpgrade(ctx, state0, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error reading the instance state: %s", d.Summary)
			}
		}
	}

	// Diff-1
	diff1, err := AWSS3Bucket.Diff(ctx, state1, resourceConfig, p.Meta())
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// Apply-1
	if diff1 == nil {
		os.Exit(0)
	}

	state2, diags := AWSS3Bucket.Apply(ctx, state1, diff1, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error configuring S3 bucket: %s", d.Summary)
			}
		}
	}

	fmt.Println("\nState after apply-1")
	fmt.Println("-------------------")
	fmt.Println(state2)

	//-------------------------------------------------------------------------
	// Round-2
	//-------------------------------------------------------------------------

	// Diff-2
	diff2, err := AWSS3Bucket.Diff(ctx, state2, resourceConfig, p.Meta())
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// Apply-2
	if diff2 == nil {
		os.Exit(0)
	}

	state3, diags := AWSS3Bucket.Apply(ctx, state2, diff2, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error configuring S3 bucket: %s", d.Summary)
			}
		}
	}

	fmt.Println("\nState after apply-2")
	fmt.Println("-------------------")
	fmt.Println(state3)
}
