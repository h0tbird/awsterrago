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

// Handler ...
type Handler struct {
	ResourceID        string
	ResourceType      string
	ImportStateIgnore []string
	ResourceConfig    map[string]interface{}
	InstanceState     *terraform.InstanceState
}

//-----------------------------------------------------------------------------
// Methods
//-----------------------------------------------------------------------------

// Reconcile ...
func (h *Handler) Reconcile(ctx context.Context, p *schema.Provider) error {

	// Fixed log fields
	logFields := logrus.Fields{
		"id":   h.ResourceID,
		"type": h.ResourceType,
	}

	// Resource pointer and config
	rp := p.ResourcesMap[h.ResourceType]
	rc := &terraform.ResourceConfig{
		Config: h.ResourceConfig,
	}

	// Instance state
	if h.InstanceState == nil {
		h.InstanceState = &terraform.InstanceState{
			ID: h.ResourceID,
		}
	}

	// Refresh
	logrus.WithFields(logFields).Info("Refreshing the state")
	state1, diags := rp.RefreshWithoutUpgrade(ctx, h.InstanceState, p.Meta())
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

	if diff == nil {
		logrus.WithFields(logFields).Info("All good")
		return nil
	}

	// Remove the fields we are ignoring
	for _, v := range importStateIgnore[h.ResourceType] {
		for k := range diff.Attributes {
			if strings.HasPrefix(k, v) {
				delete(diff.Attributes, k)
			}
		}
	}

	if len(diff.Attributes) == 0 {
		logrus.WithFields(logFields).Info("All good")
		return nil
	}

	// Apply
	fooFields := logFields
	fooFields["diff"] = diff.Attributes
	logrus.WithFields(logFields).Info("Applying changes")
	state2, diags := rp.Apply(ctx, state1, diff, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				return fmt.Errorf("error configuring resource: %s", d.Summary)
			}
		}
	}

	// Diff
	logrus.WithFields(logFields).Info("Diffing state and config")
	diff, err = rp.Diff(ctx, state2, rc, p.Meta())
	if err != nil {
		return err
	}

	if diff == nil {
		logrus.WithFields(logFields).Info("All good")
		return nil
	}

	// Remove the fields we are ignoring
	for _, v := range importStateIgnore[h.ResourceType] {
		for k := range diff.Attributes {
			if strings.HasPrefix(k, v) {
				delete(diff.Attributes, k)
			}
		}
	}

	if len(diff.Attributes) == 0 {
		logrus.WithFields(logFields).Info("All good")
		return nil
	}

	// Return
	return fmt.Errorf("error state is divergent")
}
