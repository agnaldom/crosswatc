package forms

import (
	"testing"

	"github.com/agnaldom/crosswatc/internal/kube"
)

func xrd(annotation string) kube.XRD {
	return kube.XRD{
		Kind: "XVirtualMachine", ClaimKind: "VirtualMachine",
		Annotations: map[string]string{MetadataAnnotation: annotation},
		Versions: []kube.XRDVersion{{Name: "v1", Served: true, Referenceable: true, Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{"spec": map[string]any{"type": "object", "properties": map[string]any{
				"parameters": map[string]any{
					"type":     "object",
					"required": []any{"vmSize", "location"},
					"properties": map[string]any{
						"vmSize": map[string]any{"type": "string", "enum": []any{"Standard_D2as_v5", "Standard_D4as_v5"}, "description": "VM size"},
						"location": map[string]any{"type": "string",
							"x-kubernetes-validations": []any{map[string]any{"rule": "self == oldSelf"}}},
						"diskGb": map[string]any{"type": "integer", "minimum": int64(32), "maximum": int64(1024),
							"x-crosswatc": map[string]any{"editable": true, "order": 30, "displayName": "Disk (GB)", "safeInPlace": false}},
						"network": map[string]any{"type": "object", "properties": map[string]any{
							"subnet": map[string]any{"type": "string", "pattern": "^snet-"},
						}},
					},
				},
			}}},
		}}},
	}
}

const meta = `
spec.parameters.vmSize:
  displayName: VM Size
  editable: true
  order: 20
  safeInPlace: true
  approvalRequired: {dev: false, hlg: true, prd: true}
location: {editable: true, order: 10}
`

func TestBuild(t *testing.T) {
	f, err := Build(xrd(meta))
	if err != nil {
		t.Fatal(err)
	}
	if f.Kind != "VirtualMachine" || f.Version != "v1" || len(f.Fields) != 4 {
		t.Fatalf("%+v", f)
	}
	keys := []string{}
	for _, fl := range f.Fields {
		keys = append(keys, fl.Key)
	}
	if keys[0] != "location" || keys[1] != "vmSize" || keys[2] != "diskGb" || keys[3] != "network.subnet" {
		t.Fatalf("order %v", keys)
	}

	vm, _ := f.FieldByKey("vmSize")
	if vm.Title != "VM Size" || !vm.Editable || !vm.Required || len(vm.Enum) != 2 || vm.Path != "spec.parameters.vmSize" || !vm.SafeInPlace {
		t.Fatalf("vmSize %+v", vm)
	}
	if vm.ApprovalRequiredIn("dev") || !vm.ApprovalRequiredIn("prd") || !vm.ApprovalRequiredIn("other") {
		t.Fatal("approval")
	}

	loc, _ := f.FieldByKey("location")
	if !loc.Immutable || loc.Editable || loc.SafeInPlace {
		t.Fatalf("immutable fields are never editable: %+v", loc)
	}

	disk, _ := f.FieldByKey("diskGb")
	if !disk.Editable || disk.Title != "Disk (GB)" || disk.SafeInPlace || *disk.Minimum != 32 || *disk.Maximum != 1024 {
		t.Fatalf("x-crosswatc extension: %+v", disk)
	}

	sub, _ := f.FieldByKey("network.subnet")
	if sub.Editable || sub.Pattern != "^snet-" || sub.ApprovalRequiredIn("dev") != false {
		t.Fatalf("no metadata → read-only: %+v", sub)
	}
}

func TestBuildEdgeCases(t *testing.T) {
	if _, err := Build(xrd("{not yaml: [")); err == nil {
		t.Fatal("bad annotation must error")
	}
	f, err := Build(kube.XRD{Kind: "X", Versions: []kube.XRDVersion{{Name: "v1", Served: true}}})
	if err != nil || f.Kind != "X" || len(f.Fields) != 0 {
		t.Fatalf("no schema: %+v %v", f, err)
	}
}

func TestValues(t *testing.T) {
	f, _ := Build(xrd(meta))
	spec := map[string]any{"parameters": map[string]any{"vmSize": "Standard_D2as_v5", "network": map[string]any{"subnet": "snet-a"}}}
	v := Values(f, spec)
	if v["vmSize"] != "Standard_D2as_v5" || v["network.subnet"] != "snet-a" || len(v) != 2 {
		t.Fatalf("%v", v)
	}
	if _, ok := Lookup(spec, "parameters.nope"); ok {
		t.Fatal("lookup")
	}
}
