package main

//-----------------------------------------------------------------------------
// Imports
//-----------------------------------------------------------------------------

import (

	// stdlib
	"context"
	"io/ioutil"
	"log"

	// community
	"github.com/sirupsen/logrus"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/terraform-providers/terraform-provider-aws/aws"

	// local
	"github.com/h0tbird/awsterrago/pkg/dag"
	"github.com/h0tbird/awsterrago/pkg/resource"
)

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
	r := map[string]*resource.Handler{}
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
	r["nodesPolicy"] = &resource.Handler{
		ResourceLogicalID: "NodesPolicy",
		ResourceType:      "aws_iam_policy",
		ResourceConfig: map[string]interface{}{
			"name":        "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"description": "For the Kubernetes Cloud Provider AWS nodes",
			"policy":      nodesPolicy,
		},
	}

	// AWS::IAM::Role
	r["nodesRole"] = &resource.Handler{
		ResourceLogicalID: "NodesRole",
		ResourceType:      "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name":               "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": assumeRolePolicy,
		},
	}

	// AWS::IAM::RolePolicyAttachment
	r["nodesRoleToNodesPolicyAttachment"] = &resource.Handler{
		ResourceLogicalID: "NodesRoleToNodesPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       r["nodesRole"].ResourceConfig["name"],
			"policy_arn": "nodesPolicy.ResourceState.ID",
		},
	}

	// AWS::IAM::InstanceProfile
	r["nodesInstanceProfile"] = &resource.Handler{
		ResourceLogicalID: "NodesInstanceProfile",
		ResourceType:      "aws_iam_instance_profile",
		ResourceConfig: map[string]interface{}{
			"name": "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"role": r["nodesRole"].ResourceConfig["name"],
		},
	}

	//-----------------------------------------------------------------------------------
	// controllers.cluster-api-provider-aws.sigs.k8s.io
	//-----------------------------------------------------------------------------------

	// AWS::IAM::Policy
	r["controllersPolicy"] = &resource.Handler{
		ResourceLogicalID: "ControllersPolicy",
		ResourceType:      "aws_iam_policy",
		ResourceConfig: map[string]interface{}{
			"name":        "controllers.cluster-api-provider-aws.sigs.k8s.io",
			"description": "For the Kubernetes Cluster API Provider AWS Controllers",
			"policy":      controllersPolicy,
		},
	}

	// AWS::IAM::Role
	r["controllersRole"] = &resource.Handler{
		ResourceLogicalID: "ControllersRole",
		ResourceType:      "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name":               "controllers.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": assumeRolePolicy,
		},
	}

	// AWS::IAM::RolePolicyAttachment
	r["controllersRoleToControllersPolicyAttachment"] = &resource.Handler{
		ResourceLogicalID: "ControllersRoleToControllersPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       r["controllersRole"].ResourceConfig["name"],
			"policy_arn": "controllersPolicy.ResourceState.ID",
		},
	}

	// AWS::IAM::InstanceProfile
	r["controllersInstanceProfile"] = &resource.Handler{
		ResourceLogicalID: "ControllersInstanceProfile",
		ResourceType:      "aws_iam_instance_profile",
		ResourceConfig: map[string]interface{}{
			"name": "controllers.cluster-api-provider-aws.sigs.k8s.io",
			"role": r["controllersRole"].ResourceConfig["name"],
		},
	}

	//----------------------------------------------------
	// control-plane.cluster-api-provider-aws.sigs.k8s.io
	//----------------------------------------------------

	// AWS::IAM::Policy
	r["controlPlanePolicy"] = &resource.Handler{
		ResourceLogicalID: "ControlPlanePolicy",
		ResourceType:      "aws_iam_policy",
		ResourceConfig: map[string]interface{}{
			"name":        "control-plane.cluster-api-provider-aws.sigs.k8s.io",
			"description": "For the Kubernetes Cloud Provider AWS Control Plane",
			"policy":      controlPlanePolicy,
		},
	}

	// AWS::IAM::Role
	r["controlPlaneRole"] = &resource.Handler{
		ResourceLogicalID: "ControlPlaneRole",
		ResourceType:      "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name":               "control-plane.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": assumeRolePolicy,
		},
	}

	// AWS::IAM::RolePolicyAttachment
	r["controlPlaneRoleToControlPlanePolicyAttachment"] = &resource.Handler{
		ResourceLogicalID: "ControlPlaneRoleToControlPlanePolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       r["controlPlaneRole"].ResourceConfig["name"],
			"policy_arn": "controlPlanePolicy.ResourceState.ID",
		},
	}

	// AWS::IAM::RolePolicyAttachment
	r["controlPlaneRoleToNodesPolicyAttachment"] = &resource.Handler{
		ResourceLogicalID: "ControlPlaneRoleToNodesPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       r["controlPlaneRole"].ResourceConfig["name"],
			"policy_arn": "nodesPolicy.ResourceState.ID",
		},
	}

	// AWS::IAM::RolePolicyAttachment
	r["controlPlaneRoleToControllersPolicyAttachment"] = &resource.Handler{
		ResourceLogicalID: "ControlPlaneRoleToControllersPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       r["controlPlaneRole"].ResourceConfig["name"],
			"policy_arn": "controllersPolicy.ResourceState.ID",
		},
	}

	// AWS::IAM::InstanceProfile
	r["controlPlaneInstanceProfile"] = &resource.Handler{
		ResourceLogicalID: "ControlPlaneInstanceProfile",
		ResourceType:      "aws_iam_instance_profile",
		ResourceConfig: map[string]interface{}{
			"name": "control-plane.cluster-api-provider-aws.sigs.k8s.io",
			"role": r["controlPlaneRole"].ResourceConfig["name"],
		},
	}

	//---------------
	// Setup the DAG
	//---------------

	g := dag.AcyclicGraph{}

	for k := range r {
		g.Add(r[k])
	}

	// TODO: Automagic connections
	g.Connect(dag.BasicEdge(0, r["nodesPolicy"]))
	g.Connect(dag.BasicEdge(0, r["nodesRole"]))
	g.Connect(dag.BasicEdge(0, r["controllersPolicy"]))
	g.Connect(dag.BasicEdge(0, r["controllersRole"]))
	g.Connect(dag.BasicEdge(0, r["controlPlanePolicy"]))
	g.Connect(dag.BasicEdge(0, r["controlPlaneRole"]))
	g.Connect(dag.BasicEdge(r["nodesPolicy"], r["nodesRoleToNodesPolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["nodesRole"], r["nodesRoleToNodesPolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["nodesRole"], r["nodesInstanceProfile"]))
	g.Connect(dag.BasicEdge(r["controllersPolicy"], r["controllersRoleToControllersPolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["controllersRole"], r["controllersRoleToControllersPolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["controllersRole"], r["controllersInstanceProfile"]))
	g.Connect(dag.BasicEdge(r["controlPlanePolicy"], r["controlPlaneRoleToControlPlanePolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["controlPlaneRole"], r["controlPlaneRoleToControlPlanePolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["nodesPolicy"], r["controlPlaneRoleToNodesPolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["controlPlaneRole"], r["controlPlaneRoleToNodesPolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["controllersPolicy"], r["controlPlaneRoleToControllersPolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["controlPlaneRole"], r["controlPlaneRoleToControllersPolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["controlPlaneRole"], r["controlPlaneInstanceProfile"]))

	//--------------
	// Walk the DAG
	//--------------

	w := &dag.Walker{Callback: resource.Walk(ctx, p, s, r)}
	w.Update(&g)

	if err := w.Wait(); err != nil {
		logrus.Fatalf("err: %s", err)
	}
}
