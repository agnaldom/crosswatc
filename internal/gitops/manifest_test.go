package gitops

import (
	"errors"
	"strings"
	"testing"
)

const claim = `# Managed by Backstage
apiVersion: compute.crosswatc.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm-payments
  namespace: team-payments
  labels:
    crosswatc.io/managed: "true"
spec:
  parameters:
    vmSize: Standard_D2as_v5 # sized for dev
    diskGb: 64
    location: brazilsouth
`

func TestUpdateManifestPreservesFormatting(t *testing.T) {
	out, err := UpdateManifest([]byte(claim), "vm-payments",
		[]Edit{{"spec.parameters.vmSize", "Standard_D4as_v5"}, {"spec.parameters.diskGb", int64(128)}, {"spec.parameters.network.subnet", "snet-a"}},
		map[string]string{"crosswatc.io/revision": "8", "crosswatc.io/request-id": "crosswatc-req-X"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"# Managed by Backstage",
		"vmSize: Standard_D4as_v5 # sized for dev",
		"diskGb: 128",
		"location: brazilsouth",
		"    network:\n      subnet: snet-a",
		`crosswatc.io/revision: "8"`,
		"crosswatc.io/request-id: crosswatc-req-X",
		`crosswatc.io/managed: "true"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	if strings.Index(s, "vmSize") > strings.Index(s, "location") {
		t.Error("key order must be preserved")
	}
}

func TestUpdateManifestMultiDoc(t *testing.T) {
	src := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: other\n---\n" + claim
	out, err := UpdateManifest([]byte(src), "vm-payments", []Edit{{"spec.parameters.diskGb", int64(256)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "name: other") || !strings.Contains(string(out), "diskGb: 256") {
		t.Fatalf("%s", out)
	}
	if _, err := UpdateManifest([]byte(src), "missing", nil, nil); !errors.Is(err, ErrClaimNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateManifestErrors(t *testing.T) {
	if _, err := UpdateManifest([]byte("a: [1"), "x", nil, nil); err == nil {
		t.Fatal("invalid yaml")
	}
	if _, err := UpdateManifest([]byte("spec: 1\nmetadata: {name: x}\n"), "x", []Edit{{"spec.parameters.a", "b"}}, nil); err == nil {
		t.Fatal("non-mapping intermediate")
	}
}
