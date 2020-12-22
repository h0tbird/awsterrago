package main

//-----------------------------------------------------------------------------
// Imports
//-----------------------------------------------------------------------------

import (

	// stdlib
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"

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

	controllersPolicy = `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Action": [
					"ec2:AllocateAddress",
					"ec2:AssociateRouteTable",
					"ec2:AttachInternetGateway",
					"ec2:AuthorizeSecurityGroupIngress",
					"ec2:CreateInternetGateway",
					"ec2:CreateNatGateway",
					"ec2:CreateRoute",
					"ec2:CreateRouteTable",
					"ec2:CreateSecurityGroup",
					"ec2:CreateSubnet",
					"ec2:CreateTags",
					"ec2:CreateVpc",
					"ec2:ModifyVpcAttribute",
					"ec2:DeleteInternetGateway",
					"ec2:DeleteNatGateway",
					"ec2:DeleteRouteTable",
					"ec2:DeleteSecurityGroup",
					"ec2:DeleteSubnet",
					"ec2:DeleteTags",
					"ec2:DeleteVpc",
					"ec2:DescribeAccountAttributes",
					"ec2:DescribeAddresses",
					"ec2:DescribeAvailabilityZones",
					"ec2:DescribeInstances",
					"ec2:DescribeInternetGateways",
					"ec2:DescribeImages",
					"ec2:DescribeNatGateways",
					"ec2:DescribeNetworkInterfaces",
					"ec2:DescribeNetworkInterfaceAttribute",
					"ec2:DescribeRouteTables",
					"ec2:DescribeSecurityGroups",
					"ec2:DescribeSubnets",
					"ec2:DescribeVpcs",
					"ec2:DescribeVpcAttribute",
					"ec2:DescribeVolumes",
					"ec2:DetachInternetGateway",
					"ec2:DisassociateRouteTable",
					"ec2:DisassociateAddress",
					"ec2:ModifyInstanceAttribute",
					"ec2:ModifyNetworkInterfaceAttribute",
					"ec2:ModifySubnetAttribute",
					"ec2:ReleaseAddress",
					"ec2:RevokeSecurityGroupIngress",
					"ec2:RunInstances",
					"ec2:TerminateInstances",
					"tag:GetResources",
					"elasticloadbalancing:AddTags",
					"elasticloadbalancing:CreateLoadBalancer",
					"elasticloadbalancing:ConfigureHealthCheck",
					"elasticloadbalancing:DeleteLoadBalancer",
					"elasticloadbalancing:DescribeLoadBalancers",
					"elasticloadbalancing:DescribeLoadBalancerAttributes",
					"elasticloadbalancing:DescribeTags",
					"elasticloadbalancing:ModifyLoadBalancerAttributes",
					"elasticloadbalancing:RegisterInstancesWithLoadBalancer",
					"elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
					"elasticloadbalancing:RemoveTags"
				],
				"Resource": [
					"*"
				],
				"Effect": "Allow"
			},
			{
				"Condition": {
					"StringLike": {
						"iam:AWSServiceName": "elasticloadbalancing.amazonaws.com"
					}
				},
				"Action": [
					"iam:CreateServiceLinkedRole"
				],
				"Resource": [
					"arn:*:iam::*:role/aws-service-role/elasticloadbalancing.amazonaws.com/AWSServiceRoleForElasticLoadBalancing"
				],
				"Effect": "Allow"
			},
			{
				"Condition": {
					"StringLike": {
						"iam:AWSServiceName": "spot.amazonaws.com"
					}
				},
				"Action": [
					"iam:CreateServiceLinkedRole"
				],
				"Resource": [
					"arn:*:iam::*:role/aws-service-role/spot.amazonaws.com/AWSServiceRoleForEC2Spot"
				],
				"Effect": "Allow"
			},
			{
				"Action": [
					"iam:PassRole"
				],
				"Resource": [
					"arn:*:iam::*:role/*.cluster-api-provider-aws.sigs.k8s.io"
				],
				"Effect": "Allow"
			},
			{
				"Action": [
					"secretsmanager:CreateSecret",
					"secretsmanager:DeleteSecret",
					"secretsmanager:TagResource"
				],
				"Resource": [
					"arn:*:secretsmanager:*:*:secret:aws.cluster.x-k8s.io/*"
				],
				"Effect": "Allow"
			}
		]
	}`

	controlPlanePolicy = `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Action": [
					"autoscaling:DescribeAutoScalingGroups",
					"autoscaling:DescribeLaunchConfigurations",
					"autoscaling:DescribeTags",
					"ec2:DescribeInstances",
					"ec2:DescribeImages",
					"ec2:DescribeRegions",
					"ec2:DescribeRouteTables",
					"ec2:DescribeSecurityGroups",
					"ec2:DescribeSubnets",
					"ec2:DescribeVolumes",
					"ec2:CreateSecurityGroup",
					"ec2:CreateTags",
					"ec2:CreateVolume",
					"ec2:ModifyInstanceAttribute",
					"ec2:ModifyVolume",
					"ec2:AttachVolume",
					"ec2:AuthorizeSecurityGroupIngress",
					"ec2:CreateRoute",
					"ec2:DeleteRoute",
					"ec2:DeleteSecurityGroup",
					"ec2:DeleteVolume",
					"ec2:DetachVolume",
					"ec2:RevokeSecurityGroupIngress",
					"ec2:DescribeVpcs",
					"elasticloadbalancing:AddTags",
					"elasticloadbalancing:AttachLoadBalancerToSubnets",
					"elasticloadbalancing:ApplySecurityGroupsToLoadBalancer",
					"elasticloadbalancing:CreateLoadBalancer",
					"elasticloadbalancing:CreateLoadBalancerPolicy",
					"elasticloadbalancing:CreateLoadBalancerListeners",
					"elasticloadbalancing:ConfigureHealthCheck",
					"elasticloadbalancing:DeleteLoadBalancer",
					"elasticloadbalancing:DeleteLoadBalancerListeners",
					"elasticloadbalancing:DescribeLoadBalancers",
					"elasticloadbalancing:DescribeLoadBalancerAttributes",
					"elasticloadbalancing:DetachLoadBalancerFromSubnets",
					"elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
					"elasticloadbalancing:ModifyLoadBalancerAttributes",
					"elasticloadbalancing:RegisterInstancesWithLoadBalancer",
					"elasticloadbalancing:SetLoadBalancerPoliciesForBackendServer",
					"elasticloadbalancing:AddTags",
					"elasticloadbalancing:CreateListener",
					"elasticloadbalancing:CreateTargetGroup",
					"elasticloadbalancing:DeleteListener",
					"elasticloadbalancing:DeleteTargetGroup",
					"elasticloadbalancing:DescribeListeners",
					"elasticloadbalancing:DescribeLoadBalancerPolicies",
					"elasticloadbalancing:DescribeTargetGroups",
					"elasticloadbalancing:DescribeTargetHealth",
					"elasticloadbalancing:ModifyListener",
					"elasticloadbalancing:ModifyTargetGroup",
					"elasticloadbalancing:RegisterTargets",
					"elasticloadbalancing:SetLoadBalancerPoliciesOfListener",
					"iam:CreateServiceLinkedRole",
					"kms:DescribeKey"
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

func (s *state) Read(logicalID string, state interface{}) error {

	// Open a file handler
	f, err := os.Open(os.Getenv("HOME") + "/.terramorph/" + logicalID + ".json")
	if err != nil {

		// No file means no state
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	// Unmarshal json
	return json.NewDecoder(f).Decode(state)
}

func (s *state) Write(logicalID string, state interface{}) error {

	// Open a file handler
	f, err := os.Create(os.Getenv("HOME") + "/.terramorph/" + logicalID + ".json")
	if err != nil {
		return err
	}
	defer f.Close()

	// Marshal json
	jsonBytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	// Write to disk
	_, err = io.Copy(f, bytes.NewReader(jsonBytes))
	return err
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

	//--------------------------------------------
	// nodes.cluster-api-provider-aws.sigs.k8s.io
	//--------------------------------------------

	// AWS::IAM::Policy
	nodesPolicy := &resource.Handler{
		ResourceLogicalID: "NodesPolicy",
		ResourceType:      "aws_iam_policy",
		ResourceConfig: map[string]interface{}{
			"name":        "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"description": "For the Kubernetes Cloud Provider AWS nodes",
			"policy":      nodesPolicy,
		},
	}

	if err := nodesPolicy.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::Role
	nodesRole := &resource.Handler{
		ResourceLogicalID: "NodesRole",
		ResourceType:      "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name":               "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": assumeRolePolicy,
		},
	}

	if err := nodesRole.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::RolePolicyAttachment
	nodesRoleToNodesPolicyAttachment := &resource.Handler{
		ResourceLogicalID: "NodesRoleToNodesPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       nodesRole.ResourceConfig["name"],
			"policy_arn": nodesPolicy.ResourceState.ID,
		},
	}

	if err := nodesRoleToNodesPolicyAttachment.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::InstanceProfile
	nodesInstanceProfile := &resource.Handler{
		ResourceLogicalID: "NodesInstanceProfile",
		ResourceType:      "aws_iam_instance_profile",
		ResourceConfig: map[string]interface{}{
			"name": "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"role": nodesRole.ResourceConfig["name"],
		},
	}

	if err := nodesInstanceProfile.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	//-----------------------------------------------------------------------------------
	// controllers.cluster-api-provider-aws.sigs.k8s.io
	//-----------------------------------------------------------------------------------

	// AWS::IAM::Policy
	controllersPolicy := &resource.Handler{
		ResourceLogicalID: "ControllersPolicy",
		ResourceType:      "aws_iam_policy",
		ResourceConfig: map[string]interface{}{
			"name":        "controllers.cluster-api-provider-aws.sigs.k8s.io",
			"description": "For the Kubernetes Cluster API Provider AWS Controllers",
			"policy":      controllersPolicy,
		},
	}

	if err := controllersPolicy.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::Role
	controllersRole := &resource.Handler{
		ResourceLogicalID: "ControllersRole",
		ResourceType:      "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name":               "controllers.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": assumeRolePolicy,
		},
	}

	if err := controllersRole.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::RolePolicyAttachment
	controllersRoleToControllersPolicyAttachment := &resource.Handler{
		ResourceLogicalID: "ControllersRoleToControllersPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       controllersRole.ResourceConfig["name"],
			"policy_arn": controllersPolicy.ResourceState.ID,
		},
	}

	if err := controllersRoleToControllersPolicyAttachment.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::InstanceProfile
	controllersInstanceProfile := &resource.Handler{
		ResourceLogicalID: "ControllersInstanceProfile",
		ResourceType:      "aws_iam_instance_profile",
		ResourceConfig: map[string]interface{}{
			"name": "controllers.cluster-api-provider-aws.sigs.k8s.io",
			"role": controllersRole.ResourceConfig["name"],
		},
	}

	if err := controllersInstanceProfile.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	//----------------------------------------------------
	// control-plane.cluster-api-provider-aws.sigs.k8s.io
	//----------------------------------------------------

	// AWS::IAM::Policy
	controlPlanePolicy := &resource.Handler{
		ResourceLogicalID: "ControlPlanePolicy",
		ResourceType:      "aws_iam_policy",
		ResourceConfig: map[string]interface{}{
			"name":        "control-plane.cluster-api-provider-aws.sigs.k8s.io",
			"description": "For the Kubernetes Cloud Provider AWS Control Plane",
			"policy":      controlPlanePolicy,
		},
	}

	if err := controlPlanePolicy.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::Role
	controlPlaneRole := &resource.Handler{
		ResourceLogicalID: "ControlPlaneRole",
		ResourceType:      "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name":               "control-plane.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": assumeRolePolicy,
		},
	}

	if err := controlPlaneRole.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::RolePolicyAttachment
	controlPlaneRoleToControlPlanePolicyAttachment := &resource.Handler{
		ResourceLogicalID: "ControlPlaneRoleToControlPlanePolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       controlPlaneRole.ResourceConfig["name"],
			"policy_arn": controlPlanePolicy.ResourceState.ID,
		},
	}

	if err := controlPlaneRoleToControlPlanePolicyAttachment.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::RolePolicyAttachment
	controlPlaneRoleToNodesPolicyAttachment := &resource.Handler{
		ResourceLogicalID: "ControlPlaneRoleToNodesPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       controlPlaneRole.ResourceConfig["name"],
			"policy_arn": nodesPolicy.ResourceState.ID,
		},
	}

	if err := controlPlaneRoleToNodesPolicyAttachment.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::RolePolicyAttachment
	controlPlaneRoleToControllersPolicyAttachment := &resource.Handler{
		ResourceLogicalID: "ControlPlaneRoleToControllersPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       controlPlaneRole.ResourceConfig["name"],
			"policy_arn": controllersPolicy.ResourceState.ID,
		},
	}

	if err := controlPlaneRoleToControllersPolicyAttachment.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::InstanceProfile
	controlPlaneInstanceProfile := &resource.Handler{
		ResourceLogicalID: "ControlPlaneInstanceProfile",
		ResourceType:      "aws_iam_instance_profile",
		ResourceConfig: map[string]interface{}{
			"name": "control-plane.cluster-api-provider-aws.sigs.k8s.io",
			"role": controlPlaneRole.ResourceConfig["name"],
		},
	}

	if err := controlPlaneInstanceProfile.Reconcile(ctx, p, s); err != nil {
		logrus.Fatal(err)
	}
}
