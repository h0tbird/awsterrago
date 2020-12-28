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
	r := map[string]*resource.Handler{}
	g := dag.AcyclicGraph{}
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

	g.Add(r["nodesPolicy"])
	g.Connect(dag.BasicEdge(0, r["nodesPolicy"]))

	// AWS::IAM::Role
	r["nodesRole"] = &resource.Handler{
		ResourceLogicalID: "NodesRole",
		ResourceType:      "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name":               "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": assumeRolePolicy,
		},
	}

	g.Add(r["nodesRole"])
	g.Connect(dag.BasicEdge(0, r["nodesRole"]))

	// AWS::IAM::RolePolicyAttachment
	r["nodesRoleToNodesPolicyAttachment"] = &resource.Handler{
		ResourceLogicalID: "NodesRoleToNodesPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       r["nodesRole"].ResourceConfig["name"],
			"policy_arn": "nodesPolicy.ResourceState.ID",
		},
	}

	g.Add(r["nodesRoleToNodesPolicyAttachment"])
	g.Connect(dag.BasicEdge(r["nodesPolicy"], r["nodesRoleToNodesPolicyAttachment"]))
	g.Connect(dag.BasicEdge(r["nodesRole"], r["nodesRoleToNodesPolicyAttachment"]))

	// AWS::IAM::InstanceProfile
	r["nodesInstanceProfile"] = &resource.Handler{
		ResourceLogicalID: "NodesInstanceProfile",
		ResourceType:      "aws_iam_instance_profile",
		ResourceConfig: map[string]interface{}{
			"name": "nodes.cluster-api-provider-aws.sigs.k8s.io",
			"role": r["nodesRole"].ResourceConfig["name"],
		},
	}

	g.Add(r["nodesInstanceProfile"])
	g.Connect(dag.BasicEdge(r["nodesRole"], r["nodesInstanceProfile"]))

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

	g.Add(r["controllersPolicy"])
	g.Connect(dag.BasicEdge(0, r["controllersPolicy"]))

	w := &dag.Walker{Callback: resource.Walk(ctx, p, s, r)}
	w.Update(&g)

	if err := w.Wait(); err != nil {
		logrus.Fatalf("err: %s", err)
	}

	os.Exit(0)

	// AWS::IAM::Role
	controllersRole := &resource.Handler{
		ResourceLogicalID: "ControllersRole",
		ResourceType:      "aws_iam_role",
		ResourceConfig: map[string]interface{}{
			"name":               "controllers.cluster-api-provider-aws.sigs.k8s.io",
			"assume_role_policy": assumeRolePolicy,
		},
	}

	//g.Add(&controllersRole)

	if err := controllersRole.Reconcile(ctx, p, s, r); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::RolePolicyAttachment
	controllersRoleToControllersPolicyAttachment := &resource.Handler{
		ResourceLogicalID: "ControllersRoleToControllersPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       controllersRole.ResourceConfig["name"],
			"policy_arn": "controllersPolicy.ResourceState.ID",
		},
	}

	//g.Add(&controllersRoleToControllersPolicyAttachment)

	if err := controllersRoleToControllersPolicyAttachment.Reconcile(ctx, p, s, r); err != nil {
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

	//g.Add(&controllersInstanceProfile)

	if err := controllersInstanceProfile.Reconcile(ctx, p, s, r); err != nil {
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

	//g.Add(&controlPlanePolicy)

	if err := controlPlanePolicy.Reconcile(ctx, p, s, r); err != nil {
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

	//g.Add(&controlPlaneRole)

	if err := controlPlaneRole.Reconcile(ctx, p, s, r); err != nil {
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

	//g.Add(&controlPlaneRoleToControlPlanePolicyAttachment)

	if err := controlPlaneRoleToControlPlanePolicyAttachment.Reconcile(ctx, p, s, r); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::RolePolicyAttachment
	controlPlaneRoleToNodesPolicyAttachment := &resource.Handler{
		ResourceLogicalID: "ControlPlaneRoleToNodesPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       controlPlaneRole.ResourceConfig["name"],
			"policy_arn": r["nodesPolicy"].ResourceState.ID,
		},
	}

	//g.Add(&controlPlaneRoleToNodesPolicyAttachment)

	if err := controlPlaneRoleToNodesPolicyAttachment.Reconcile(ctx, p, s, r); err != nil {
		logrus.Fatal(err)
	}

	// AWS::IAM::RolePolicyAttachment
	controlPlaneRoleToControllersPolicyAttachment := &resource.Handler{
		ResourceLogicalID: "ControlPlaneRoleToControllersPolicyAttachment",
		ResourceType:      "aws_iam_role_policy_attachment",
		ResourceConfig: map[string]interface{}{
			"role":       controlPlaneRole.ResourceConfig["name"],
			"policy_arn": "controllersPolicy.ResourceState.ID",
		},
	}

	//g.Add(&controlPlaneRoleToControllersPolicyAttachment)

	if err := controlPlaneRoleToControllersPolicyAttachment.Reconcile(ctx, p, s, r); err != nil {
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

	//g.Add(&controlPlaneInstanceProfile)

	if err := controlPlaneInstanceProfile.Reconcile(ctx, p, s, r); err != nil {
		logrus.Fatal(err)
	}
}
