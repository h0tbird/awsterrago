package resource

//-----------------------------------------------------------------------------
// Imports
//-----------------------------------------------------------------------------

import (

	// stdlib
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	// community
	"github.com/h0tbird/terramorph/pkg/dag"
	"github.com/h0tbird/terramorph/pkg/tfd"
	"github.com/sirupsen/logrus"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

//-----------------------------------------------------------------------------
// Globals
//-----------------------------------------------------------------------------

// Fields ignored by resource type
var importStateIgnore = map[string][]string{
	"aws_s3_bucket": []string{"force_destroy", "acl"},
	"aws_iam_role":  []string{"force_detach_policies"},
}

// <resource>.<ResourceConfig|ResourceState>.<field>
var reg = regexp.MustCompile("(\\w+)\\.(ResourceConfig|ResourceState)\\.(\\w+)")

//-----------------------------------------------------------------------------
// Types
//-----------------------------------------------------------------------------

// State ...
type State interface {
	Read(string, interface{}) error
	Write(string, interface{}) error
}

// Handler ...
type Handler struct {
	ResourceLogicalID string
	ResourceType      string
	ResourceConfig    map[string]interface{}
	ResourceState     *terraform.InstanceState
}

//-----------------------------------------------------------------------------
// Methods
//-----------------------------------------------------------------------------

// Reconcile ...
func (h *Handler) Reconcile(ctx context.Context, p *schema.Provider, s State, r map[string]*Handler) error {

	// Fixed log fields
	logFields := logrus.Fields{
		"id":   h.ResourceLogicalID,
		"type": h.ResourceType,
	}

	// Update the ResourceConfig
	for k, v := range h.ResourceConfig {
		submatch := reg.FindStringSubmatch(v.(string))
		if submatch != nil {
			switch submatch[2] {
			case "ResourceConfig":
				h.ResourceConfig[k] = r[submatch[1]].ResourceConfig[submatch[3]]
			case "ResourceState":
				h.ResourceConfig[k] = reflect.ValueOf(r[submatch[1]].ResourceState).Elem().FieldByName(submatch[3]).String()
			}
		}
	}

	// Resource pointer and config
	rp := p.ResourcesMap[h.ResourceType]
	rc := &terraform.ResourceConfig{
		Config: h.ResourceConfig,
	}

	// Read the stored state
	h.ResourceState = &terraform.InstanceState{}
	if err := s.Read(h.ResourceLogicalID, h.ResourceState); err != nil {
		return err
	}

	// Refresh the state
	logrus.WithFields(logFields).Info("Refreshing the state")
	state, diags := rp.RefreshWithoutUpgrade(ctx, h.ResourceState, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				return fmt.Errorf("error reading the instance state: %s", d.Summary)
			}
		}
	}

	// Diff
	logrus.WithFields(logFields).Info("Diffing state and config")
	diff, err := rp.Diff(ctx, state, rc, p.Meta())
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
	state, diags = rp.Apply(ctx, state, diff, p.Meta())
	if diags != nil && diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				return fmt.Errorf("error configuring resource: %s", d.Summary)
			}
		}
	}

	// Write the state
	h.ResourceState = state
	if err := s.Write(h.ResourceLogicalID, state); err != nil {
		return err
	}

	return nil
}

// Walk ...
// TODO: Move this to the 'Manifest' package?
func Walk(ctx context.Context, p *schema.Provider, s State, r map[string]*Handler) dag.WalkFunc {
	var l sync.Mutex
	return func(v dag.Vertex) tfd.Diagnostics {
		l.Lock()
		defer l.Unlock()

		rh := v.(*Handler)
		if err := rh.Reconcile(ctx, p, s, r); err != nil {
			// TODO: Return diagnostics
			logrus.Fatal(err)
		}

		return nil
	}
}
