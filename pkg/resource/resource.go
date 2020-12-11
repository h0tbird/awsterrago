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
	Provider       *schema.Provider
	ResourceType   string
	ResourceConfig *terraform.ResourceConfig
	InstanceState  *terraform.InstanceState
}

//----------------------------------------------------------------
// Methods
//----------------------------------------------------------------

// Reconcile ...
func (h *Handler) Reconcile(ctx context.Context) error {

	// Resource pointer
	resource := h.Provider.ResourcesMap[h.ResourceType]

	// Refresh
	logrus.Info("Refreshing the state...")
	state1, diags := resource.RefreshWithoutUpgrade(ctx, h.InstanceState, h.Provider.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				return fmt.Errorf("error reading the instance state: %s", d.Summary)
			}
		}
	}

	// Diff
	logrus.Info("Diffing state and config...")
	diff1, err := resource.Diff(ctx, state1, h.ResourceConfig, h.Provider.Meta())
	if err != nil {
		return err
	}

	if diff1 == nil {
		return nil
	}

	// Apply
	logrus.Info("Applying changes...")
	state2, diags := resource.Apply(ctx, state1, diff1, h.Provider.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				return fmt.Errorf("error configuring resource: %s", d.Summary)
			}
		}
	}

	// Diff
	logrus.Info("Diffing state and config...")
	diff2, err := resource.Diff(ctx, state2, h.ResourceConfig, h.Provider.Meta())
	if err != nil {
		return err
	}

	if diff2 != nil {
		return fmt.Errorf("error state is divergent")
	}

	return nil
}
