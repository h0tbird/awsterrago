package resource

//-----------------------------------------------------------------------------
// Imports
//-----------------------------------------------------------------------------

import (

	// stdlib
	"context"
	"fmt"
	"strings"

	// community
	"github.com/sirupsen/logrus"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

//-----------------------------------------------------------------------------
// Fields ignored by resource type
//-----------------------------------------------------------------------------

var importStateIgnore = map[string][]string{
	"aws_s3_bucket": []string{"force_destroy", "acl"},
	"aws_iam_role":  []string{"force_detach_policies"},
}

//-----------------------------------------------------------------------------
// Types
//-----------------------------------------------------------------------------

// State ...
type State interface {
	Read(string) (*terraform.InstanceState, error)
	Write(string, *terraform.InstanceState) error
}

// Handler ...
type Handler struct {
	ResourcePhysicalID string
	ResourceLogicalID  string
	ResourceType       string
	ImportStateIgnore  []string
	ResourceConfig     map[string]interface{}
}

//-----------------------------------------------------------------------------
// Methods
//-----------------------------------------------------------------------------

// Reconcile ...
func (h *Handler) Reconcile(ctx context.Context, p *schema.Provider, s State) error {

	// Fixed log fields
	logFields := logrus.Fields{
		"id":   h.ResourceLogicalID,
		"type": h.ResourceType,
	}

	// Resource pointer and config
	rp := p.ResourcesMap[h.ResourceType]
	rc := &terraform.ResourceConfig{
		Config: h.ResourceConfig,
	}

	// Read the stored state
	state0, err := s.Read(h.ResourceLogicalID)
	if err != nil {
		return err
	}

	// Default to empty state
	if state0 == nil {
		state0 = &terraform.InstanceState{
			ID: h.ResourcePhysicalID,
		}
	}

	// Refresh the state
	logrus.WithFields(logFields).Info("Refreshing the state")
	state1, diags := rp.RefreshWithoutUpgrade(ctx, state0, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				return fmt.Errorf("error reading the instance state: %s", d.Summary)
			}
		}
	}

	// Diff
	logrus.WithFields(logFields).Info("Diffing state and config")
	diff, err := rp.Diff(ctx, state1, rc, p.Meta())
	if err != nil {
		return err
	}

	// Return if there is nothing to sync
	if diff == nil {
		logrus.WithFields(logFields).Info("All good")
		return nil
	}

	// Remove all ignored attributes
	for _, v := range importStateIgnore[h.ResourceType] {
		for k := range diff.Attributes {
			if strings.HasPrefix(k, v) {
				delete(diff.Attributes, k)
			}
		}
	}

	// Return if there is nothing to sync
	if len(diff.Attributes) == 0 {
		logrus.WithFields(logFields).Info("All good")
		return nil
	}

	// Add out-of-sync attributes to the log
	logFields["diff"] = []string{}
	for k := range diff.Attributes {
		logFields["diff"] = append(logFields["diff"].([]string), k)
	}

	// Apply the changes
	logrus.WithFields(logFields).Info("Applying changes")
	state2, diags := rp.Apply(ctx, state1, diff, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				return fmt.Errorf("error configuring resource: %s", d.Summary)
			}
		}
	}

	// Write the state
	if err := s.Write(h.ResourceLogicalID, state2); err != nil {
		return err
	}

	return nil
}
