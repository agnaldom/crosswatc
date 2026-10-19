package httpapi

import (
	"context"
	"testing"

	"github.com/agnaldom/crosswatc/internal/auth"
	"github.com/agnaldom/crosswatc/internal/forms"
	"github.com/agnaldom/crosswatc/internal/kube"
)

type formCatalog struct{}

func pgXRDWithForm() kube.XRD {
	return kube.XRD{Group: "database.crosswatc.io", Kind: "XPostgreSQLInstance", ClaimKind: "PostgreSQLInstance", Scope: "Cluster",
		Annotations: map[string]string{forms.MetadataAnnotation: `{"vmSize": {"editable": true, "displayName": "VM Size"}}`},
		Versions: []kube.XRDVersion{{Name: "v1", Served: true, Referenceable: true, Schema: map[string]any{
			"type": "object", "properties": map[string]any{"spec": map[string]any{"type": "object", "properties": map[string]any{
				"parameters": map[string]any{"type": "object", "properties": map[string]any{
					"vmSize":  map[string]any{"type": "string"},
					"storage": map[string]any{"type": "integer"},
				}},
			}}},
		}}}}
}

func (formCatalog) DiscoverXRDs(context.Context) ([]kube.XRD, error) {
	return []kube.XRD{pgXRDWithForm()}, nil
}

func TestForms(t *testing.T) {
	fs, _, _ := setup()
	p := fs.projection[payID]
	p.AppliedState = []byte(`{"spec":{"parameters":{"vmSize":"Standard_D2as_v5","storage":64}}}`)
	fs.projection[payID] = p
	h := New(Deps{Store: fs, Catalog: formCatalog{}, Auth: auth.Dev{Mapping: auth.GroupMapping{SquadPrefix: "squad-"}}})

	w := do(h, "/api/v1/catalog/resource-types/PostgreSQLInstance/form", payments)
	var f forms.Form
	decode(t, w, &f)
	if w.Code != 200 || len(f.Fields) != 2 || f.Fields[0].Key != "storage" {
		t.Fatalf("%d %+v", w.Code, f)
	}

	w = do(h, "/api/v1/resources/"+payID.String()+"/form", payments)
	var rf resourceFormDTO
	decode(t, w, &rf)
	if !rf.Editable || rf.Values["vmSize"] != "Standard_D2as_v5" || rf.Values["storage"] != float64(64) {
		t.Fatalf("%+v", rf)
	}
	vm, _ := rf.Form.FieldByKey("vmSize")
	if !vm.Editable || vm.Title != "VM Size" {
		t.Fatalf("%+v", vm)
	}

	r := fs.resources[payID]
	r.ResourceState = "UPDATING"
	fs.resources[payID] = r
	decode(t, do(h, "/api/v1/resources/"+payID.String()+"/form", payments), &rf)
	if rf.Editable || rf.Reason == "" {
		t.Fatalf("non-operational must not be editable: %+v", rf)
	}

	if w := do(h, "/api/v1/catalog/resource-types/Nope/form", payments); w.Code != 404 {
		t.Fatalf("%d", w.Code)
	}
}
