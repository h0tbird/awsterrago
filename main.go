package main

//----------------------------------------------------------------
// Imports
//----------------------------------------------------------------

import (

	// stdlib
	"context"
	"io/ioutil"
	"log"
	"sync"

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
	var wg sync.WaitGroup

	//------------------------
	// Configure the provider
	//------------------------

	p := aws.Provider()
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

	//-------------------------
	// Create a nice S3 bucket
	//-------------------------

	myNiceBucket := &resource.Handler{
		ResourceType: "aws_s3_bucket",
		ResourceConfig: map[string]interface{}{
			"bucket": "my-nice-bucket",
		},
		InstanceState: &terraform.InstanceState{
			ID: "my-nice-bucket",
			Attributes: map[string]string{
				"acl":           "private",
				"force_destroy": "false",
			},
		},
	}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		if err := myNiceBucket.Reconcile(ctx, p); err != nil {
			logrus.Fatal(err)
		}
	}(&wg)

	//--------------------------
	// Create an ugly S3 bucket
	//--------------------------

	myUglyBucket := &resource.Handler{
		ResourceType: "aws_s3_bucket",
		ResourceConfig: map[string]interface{}{
			"bucket": "my-ugly-bucket",
		},
		InstanceState: &terraform.InstanceState{
			ID: "my-ugly-bucket",
			Attributes: map[string]string{
				"acl":           "private",
				"force_destroy": "false",
			},
		},
	}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		if err := myUglyBucket.Reconcile(ctx, p); err != nil {
			logrus.Fatal(err)
		}
	}(&wg)

	//------------------
	// Block until done
	//------------------

	wg.Wait()
}
