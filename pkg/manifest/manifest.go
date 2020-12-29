package manifest

//-----------------------------------------------------------------------------
// Imports
//-----------------------------------------------------------------------------

import (

	// stdlib
	"context"

	// community
	"github.com/sirupsen/logrus"

	// terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	// terramorph
	"github.com/h0tbird/terramorph/pkg/dag"
	"github.com/h0tbird/terramorph/pkg/resource"
)

//-----------------------------------------------------------------------------
// Types
//-----------------------------------------------------------------------------

// Handler ...
type Handler struct {
	Resources map[string]*resource.Handler
	Dag       dag.AcyclicGraph
}

//-----------------------------------------------------------------------------
// Methods
//-----------------------------------------------------------------------------

// New ...
func New() *Handler {
	return &Handler{
		Resources: map[string]*resource.Handler{},
		Dag:       dag.AcyclicGraph{},
	}
}

// Apply ...
func (h *Handler) Apply(ctx context.Context, p *schema.Provider, s resource.State) {

	// Setup the DAG
	for resKey, resVal := range h.Resources {

		// All vertices
		h.Dag.Add(resVal)
		match := false

		// Dependent edges
		for _, fieldVal := range resVal.ResourceConfig {
			submatch := resource.Reg.FindStringSubmatch(fieldVal.(string))
			if submatch != nil {
				h.Dag.Connect(dag.BasicEdge(h.Resources[submatch[1]], h.Resources[resKey]))
				match = true
			}
		}

		// Non-dependent edges
		if !match {
			h.Dag.Connect(dag.BasicEdge(0, h.Resources[resKey]))
		}
	}

	// Walk the DAG
	w := &dag.Walker{Callback: resource.Walk(ctx, p, s, h.Resources)}
	w.Update(&h.Dag)

	// TODO: err is Diagnostics (return it)
	if err := w.Wait(); err != nil {
		logrus.Fatalf("err: %s", err)
	}
}
