package resource

//----------------------------------------------------------------
// Imports
//----------------------------------------------------------------

import (

	// stdlib
	"context"
	"fmt"

	// community
	"github.com/sirupsen/logrus"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

//----------------------------------------------------------------
// Types
//----------------------------------------------------------------

// Handler ...
type Handler struct {
	ResourceID     string
	ResourceType   string
	ResourceConfig map[string]interface{}
	InstanceState  *terraform.InstanceState
}

//----------------------------------------------------------------
// Methods
//----------------------------------------------------------------

// Reconcile ...
func (h *Handler) Reconcile(ctx context.Context, p *schema.Provider) error {

	// Fixed log fields
	logFields := logrus.Fields{
		"type": h.ResourceType,
		"id":   h.InstanceState.ID,
	}

	// Resource pointer and config
	rp := p.ResourcesMap[h.ResourceType]
	rc := &terraform.ResourceConfig{
		Config: h.ResourceConfig,
	}

	// Set ID if provided
	if h.ResourceID != "" {
		h.InstanceState.ID = h.ResourceID
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
	diff1, err := rp.Diff(ctx, state1, rc, p.Meta())
	if err != nil {
		return err
	}

	if diff1 == nil {
		logrus.WithFields(logFields).Info("All good")
		return nil
	}

	// Apply
	logrus.WithFields(logFields).Info("Applying changes")
	state2, diags := rp.Apply(ctx, state1, diff1, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				return fmt.Errorf("error configuring resource: %s", d.Summary)
			}
		}
	}

	// Diff
	logrus.WithFields(logFields).Info("Diffing state and config")
	diff2, err := rp.Diff(ctx, state2, rc, p.Meta())
	if err != nil {
		return err
	}

	if diff2 == nil {
		logrus.WithFields(logFields).Info("All good")
		return nil
	}

	// Return
	return fmt.Errorf("error state is divergent")
}
