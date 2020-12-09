package main

//----------------------------------------------------------------
// Imports
//----------------------------------------------------------------

import (

	// stdlib
	"context"
	"fmt"
	"io/ioutil"
	"log"
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

	// Send all logs to /dev/null
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

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

	// Initial state
	state0 := &terraform.InstanceState{
		ID: "my-nice-bucket",
		Attributes: map[string]string{
			"acl":           "private",
			"force_destroy": "false",
		},
	}

	// Resource to configure
	AWSS3Bucket := p.ResourcesMap["aws_s3_bucket"]

	// Refresh
	state1, diags := AWSS3Bucket.RefreshWithoutUpgrade(ctx, state0, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error reading the instance state: %s", d.Summary)
			}
		}
	}

	// Diff
	diff1, err := AWSS3Bucket.Diff(ctx, state1, resourceConfig, p.Meta())
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if diff1 == nil {
		fmt.Println("All good 1")
		os.Exit(0)
	}

	// Apply
	state2, diags := AWSS3Bucket.Apply(ctx, state1, diff1, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				fmt.Printf("error configuring S3 bucket: %s", d.Summary)
			}
		}
	}

	// Diff
	diff2, err := AWSS3Bucket.Diff(ctx, state2, resourceConfig, p.Meta())
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if diff2 == nil {
		fmt.Println("All good 2")
		os.Exit(0)
	}
}
