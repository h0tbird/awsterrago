package main

//----------------------------------------------------------------
// Imports
//----------------------------------------------------------------

import (

	// stdlib
	"context"
	"io/ioutil"
	"log"

	// community
	"github.com/h0tbird/awsterrago/pkg/resource"
	"github.com/sirupsen/logrus"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/terraform-providers/terraform-provider-aws/aws"
)

//----------------------------------------------------------------
// Init
//----------------------------------------------------------------

func init() {
	log.SetOutput(ioutil.Discard)
}

//----------------------------------------------------------------
// Main
//----------------------------------------------------------------

func main() {

	ctx := context.Background()
	p := aws.Provider()

	// Configure the provider
	logrus.WithFields(logrus.Fields{"region": "us-east-2"}).Info("Configuring the provider")
	diags := p.Configure(ctx, &terraform.ResourceConfig{
		Config: map[string]interface{}{
			"region": "us-east-2",
		},
	})

	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				logrus.Fatalf("error configuring the provider: %s", d.Summary)
			}
		}
	}

	//--------------------
	// Create a S3 bucket
	//--------------------

	(&resource.Handler{
		ResourceType: "aws_s3_bucket",
		ResourceConfig: &terraform.ResourceConfig{
			Config: map[string]interface{}{
				"bucket": "my-nice-bucket",
			},
		},
		InstanceState: &terraform.InstanceState{
			ID: "my-nice-bucket",
			Attributes: map[string]string{
				"acl":           "private",
				"force_destroy": "false",
			},
		},
	}).Reconcile(ctx, p)
}
