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

	// AWS::IAM::ManagedPolicy | nodes.cluster-api-provider-aws.sigs.k8s.io
	nodesPolicy := &resource.Handler{
		ResourceID:   "arn:aws:iam::729179300383:policy/nodes.cluster-api-provider-aws.sigs.k8s.io",
		ResourceType: "aws_iam_policy",
		ResourceConfig: map[string]interface{}{
			"name":        "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"description": "For the Kubernetes Cloud Provider AWS nodes",
			"policy": `{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Action": [
							"ec2:DescribeInstances",
							"ec2:DescribeRegions",
							"ecr:GetAuthorizationToken",
							"ecr:BatchCheckLayerAvailability",
							"ecr:GetDownloadUrlForLayer",
							"ecr:GetRepositoryPolicy",
							"ecr:DescribeRepositories",
							"ecr:ListImages",
							"ecr:BatchGetImage"
						],
						"Resource": [
							"*"
						],
						"Effect": "Allow"
					},
					{
						"Action": [
							"secretsmanager:DeleteSecret",
							"secretsmanager:GetSecretValue"
						],
						"Resource": [
							"arn:*:secretsmanager:*:*:secret:aws.cluster.x-k8s.io/*"
						],
						"Effect": "Allow"
					},
					{
						"Action": [
							"ssm:UpdateInstanceInformation",
							"ssmmessages:CreateControlChannel",
							"ssmmessages:CreateDataChannel",
							"ssmmessages:OpenControlChannel",
							"ssmmessages:OpenDataChannel",
							"s3:GetEncryptionConfiguration"
						],
						"Resource": [
							"*"
						],
						"Effect": "Allow"
					}
				]
			}`,
		},
	}

	if err := nodesPolicy.Reconcile(ctx, p); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::Role | nodes.cluster-api-provider-aws.sigs.k8s.io
	nodesRole := &resource.Handler{
		ResourceID:   "nodes.cluster-api-provider-aws.sigs.k8s.io",
		ResourceType: "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name": "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": `{
				"Version": "2012-10-17",
				"Statement": [
				  {
					"Effect": "Allow",
					"Principal": {
					  "Service": "ec2.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				  }
				]
			  }`,
		},
		InstanceState: &terraform.InstanceState{
			ID: "nodes.cluster-api-provider-aws.sigs.k8s.io",
			Attributes: map[string]string{
				"force_detach_policies": "false",
			},
		},
	}

	if err := nodesRole.Reconcile(ctx, p); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::InstanceProfile | nodes.cluster-api-provider-aws.sigs.k8s.io

	// AWS::IAM::ManagedPolicy   | control-plane.cluster-api-provider-aws.sigs.k8s.io
	// AWS::IAM::Role            | control-plane.cluster-api-provider-aws.sigs.k8s.io
	// AWS::IAM::InstanceProfile | control-plane.cluster-api-provider-aws.sigs.k8s.io

	// AWS::IAM::ManagedPolicy   | controllers.cluster-api-provider-aws.sigs.k8s.io
	// AWS::IAM::Role            | controllers.cluster-api-provider-aws.sigs.k8s.io
	// AWS::IAM::InstanceProfile | controllers.cluster-api-provider-aws.sigs.k8s.io

	//-------------------------
	// Create a nice S3 bucket
	//-------------------------

	myNiceBucket := &resource.Handler{
		ResourceID:   "my-nice-bucket",
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
		ResourceID:   "my-ugly-bucket",
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
