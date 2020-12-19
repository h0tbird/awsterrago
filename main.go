package main

//-----------------------------------------------------------------------------
// Imports
//-----------------------------------------------------------------------------

import (

	// stdlib
	"context"
	"fmt"
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

//-----------------------------------------------------------------------------
// Constants
//-----------------------------------------------------------------------------

const (
	nodesPolicy = `{
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
}`

	assumeRolePolicy = `{
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
}`
)

//-----------------------------------------------------------------------------
// State implementation
//-----------------------------------------------------------------------------

type state struct{}

func (s *state) Read(logicalID string) (*terraform.InstanceState, error) {
	// TODO: Implement this function
	return nil, nil
}

func (s *state) Write(logicalID string, state *terraform.InstanceState) error {
	// TODO: Implement this function
	fmt.Printf("\n%v\n", state)
	return nil
}

//-----------------------------------------------------------------------------
// Init
//-----------------------------------------------------------------------------

func init() {
	// TODO: replace logrus with zap logger
	log.SetOutput(ioutil.Discard)
}

//-----------------------------------------------------------------------------
// Main
//-----------------------------------------------------------------------------

func main() {

	ctx := context.Background()
	s := &state{}

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

	//---------------------------------------------------------------
	// AWS::IAM::Policy | nodes.cluster-api-provider-aws.sigs.k8s.io
	//---------------------------------------------------------------

	nodesPolicy := &resource.Handler{
		ResourcePhysicalID: "arn:aws:iam::729179300383:policy/nodes.cluster-api-provider-aws.sigs.k8s.io",
		ResourceLogicalID:  "NodesPolicy",
		ResourceType:       "aws_iam_policy",
		ResourceConfig: map[string]interface{}{
			"name":        "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"description": "For the Kubernetes Cloud Provider AWS nodes",
			"policy":      nodesPolicy,
		},
	}

	if err := nodesPolicy.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	//-------------------------------------------------------------
	// AWS::IAM::Role | nodes.cluster-api-provider-aws.sigs.k8s.io
	//-------------------------------------------------------------

	nodesRole := &resource.Handler{
		ResourcePhysicalID: "nodes.cluster-api-provider-aws.sigs.k8s.io",
		ResourceLogicalID:  "NodesRole",
		ResourceType:       "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name":               "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": assumeRolePolicy,
		},
	}

	if err := nodesRole.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	//-----------------------------------------------------------------------------
	// AWS::IAM::RolePolicyAttachment | nodes.cluster-api-provider-aws.sigs.k8s.io
	//-----------------------------------------------------------------------------

	nodesRolePolicyAttachment := &resource.Handler{
		//ResourcePhysicalID: "nodes.cluster-api-provider-aws.sigs.k8s.io-20201219183256855300000001",
		ResourceLogicalID: "NodesRolePolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       nodesRole.ResourceConfig["name"],
			"policy_arn": nodesPolicy.ResourcePhysicalID,
		},
	}

	if err := nodesRolePolicyAttachment.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	//------------------------------------------------------------------------
	// AWS::IAM::InstanceProfile | nodes.cluster-api-provider-aws.sigs.k8s.io
	//------------------------------------------------------------------------

	nodesInstanceProfile := &resource.Handler{
		ResourcePhysicalID: "nodes.cluster-api-provider-aws.sigs.k8s.io",
		ResourceLogicalID:  "NodesInstanceProfile",
		ResourceType:       "aws_iam_instance_profile",
		ResourceConfig: map[string]interface{}{
			"name": "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"role": nodesRole.ResourceConfig["name"],
		},
	}

	if err := nodesInstanceProfile.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::ManagedPolicy   | control-plane.cluster-api-provider-aws.sigs.k8s.io
	// AWS::IAM::Role            | control-plane.cluster-api-provider-aws.sigs.k8s.io
	// AWS::IAM::InstanceProfile | control-plane.cluster-api-provider-aws.sigs.k8s.io

	// AWS::IAM::ManagedPolicy   | controllers.cluster-api-provider-aws.sigs.k8s.io
	// AWS::IAM::Role            | controllers.cluster-api-provider-aws.sigs.k8s.io
	// AWS::IAM::InstanceProfile | controllers.cluster-api-provider-aws.sigs.k8s.io
}
