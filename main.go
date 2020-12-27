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
	"sync"

	// community
	"github.com/sirupsen/logrus"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/terraform-providers/terraform-provider-aws/aws"

	// local
	"github.com/h0tbird/awsterrago/pkg/dag"
	"github.com/h0tbird/awsterrago/pkg/resource"
	"github.com/h0tbird/awsterrago/pkg/tfd"
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

	//----------
	// DAG test
	//----------

	g := dag.AcyclicGraph{}

	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Add(5)
	g.Add(6)
	g.Connect(dag.BasicEdge(1, 2))
	g.Connect(dag.BasicEdge(1, 3))
	g.Connect(dag.BasicEdge(1, 4))
	g.Connect(dag.BasicEdge(2, 5))
	g.Connect(dag.BasicEdge(3, 5))
	g.Connect(dag.BasicEdge(5, 6))
	g.Connect(dag.BasicEdge(4, 6))

	var order []interface{}
	w := &dag.Walker{Callback: walkCbRecord(&order)}
	w.Update(&g)

	if err := w.Wait(); err != nil {
		logrus.Fatalf("err: %s", err)
	}

	logrus.Println(order)

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

//-----------------------------------------------------------------------------
// Test helper callback that just records the order called.
//-----------------------------------------------------------------------------

func walkCbRecord(order *[]interface{}) dag.WalkFunc {
	var l sync.Mutex
	return func(v dag.Vertex) tfd.Diagnostics {
		l.Lock()
		defer l.Unlock()
		*order = append(*order, v)
		return nil
	}
}
