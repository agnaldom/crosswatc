package kube

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	mdfake "k8s.io/client-go/metadata/fake"

	"github.com/agnaldom/crosswatc/internal/contract"
)

func u(apiVersion, kind, ns, name string, labels map[string]any, ann map[string]any, spec map[string]any, conds ...string) *unstructured.Unstructured {
	md := map[string]any{"name": name}
	if ns != "" {
		md["namespace"] = ns
	}
	if labels != nil {
		md["labels"] = labels
	}
	if ann != nil {
		md["annotations"] = ann
	}
	var cs []any
	for i := 0; i+1 < len(conds); i += 2 {
		cs = append(cs, map[string]any{"type": conds[i], "status": conds[i+1]})
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": apiVersion, "kind": kind, "metadata": md, "spec": spec,
		"status": map[string]any{"conditions": cs, "observedGeneration": int64(1)},
	}}
}

func ref(apiVersion, kind, name string) map[string]any {
	return map[string]any{"apiVersion": apiVersion, "kind": kind, "name": name}
}

var (
	pgXRD = XRD{Name: "xpostgresqlinstances.database.crosswatc.io", Group: "database.crosswatc.io", Kind: "XPostgreSQLInstance",
		Plural: "xpostgresqlinstances", ClaimKind: "PostgreSQLInstance", ClaimPlural: "postgresqlinstances", Scope: "Cluster",
		Versions: []XRDVersion{{Name: "v1alpha1", Served: true, Referenceable: true}}}
	netXRD = XRD{Name: "xnetworks.net.crosswatc.io", Group: "net.crosswatc.io", Kind: "XNetwork", Plural: "xnetworks", Scope: "Cluster",
		Versions: []XRDVersion{{Name: "v1alpha1", Served: true, Referenceable: true}}}
	bucketXRD = XRD{Name: "buckets.storage.crosswatc.io", Group: "storage.crosswatc.io", Kind: "Bucket", Plural: "buckets", Scope: "Namespaced",
		Versions: []XRDVersion{{Name: "v1", Served: true, Referenceable: true}}}
	mrIndex = MRIndex{
		{Group: "dbforpostgresql.azure.upbound.io", Kind: "FlexibleServer"}: {ResourceTarget: ResourceTarget{Group: "dbforpostgresql.azure.upbound.io", Version: "v1beta1", Kind: "FlexibleServer", Plural: "flexibleservers"}},
		{Group: "network.azure.upbound.io", Kind: "PrivateEndpoint"}:        {ResourceTarget: ResourceTarget{Group: "network.azure.upbound.io", Version: "v1beta1", Kind: "PrivateEndpoint", Plural: "privateendpoints"}},
		{Group: "network.azure.upbound.io", Kind: "VirtualNetwork"}:         {ResourceTarget: ResourceTarget{Group: "network.azure.upbound.io", Version: "v1beta1", Kind: "VirtualNetwork", Plural: "virtualnetworks"}},
		{Group: "storage.azure.m.upbound.io", Kind: "Account"}:              {ResourceTarget: ResourceTarget{Group: "storage.azure.m.upbound.io", Version: "v1beta1", Kind: "Account", Plural: "accounts", Namespaced: true}},
	}
)

func managedLabels() map[string]any {
	return map[string]any{
		contract.LabelManaged: "true", contract.LabelOwner: "payments", contract.LabelEnvironment: "dev",
		contract.LabelProvider: "azure", contract.LabelResourceType: "postgresql",
	}
}

func managedAnnotations() map[string]any {
	return map[string]any{
		contract.AnnotationRequestID:  "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T",
		contract.AnnotationRepository: "org/argocd", contract.AnnotationPath: "claims/dev/pg.yaml",
	}
}

func discoveryClient(t *testing.T, objs ...runtime.Object) *Client {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = metav1.AddMetaToScheme(scheme)
	secret := &metav1.PartialObjectMetadata{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-payments", Name: "pg-conn"}}
	return &Client{
		Dynamic: fakeDynamic(map[schema.GroupVersionResource]string{
			{Group: "database.crosswatc.io", Version: "v1alpha1", Resource: "postgresqlinstances"}: "PostgreSQLInstanceList",
			{Group: "storage.crosswatc.io", Version: "v1", Resource: "buckets"}:                    "BucketList",
		}, objs...),
		Metadata: mdfake.NewSimpleMetadataClient(scheme, secret),
	}
}

func TestDiscoverResourcesClaimXRMR(t *testing.T) {
	claim := u("database.crosswatc.io/v1alpha1", "PostgreSQLInstance", "team-payments", "pgsql-payments", managedLabels(), managedAnnotations(),
		map[string]any{
			"resourceRef":                ref("database.crosswatc.io/v1alpha1", "XPostgreSQLInstance", "xpostgresql-92a"),
			"writeConnectionSecretToRef": map[string]any{"name": "pg-conn"},
			"parameters":                 map[string]any{"vmSize": "Standard_D2as_v5"},
		}, "Ready", "True", "Synced", "True")
	unmanaged := u("database.crosswatc.io/v1alpha1", "PostgreSQLInstance", "team-x", "legacy", nil, nil, map[string]any{})
	xr := u("database.crosswatc.io/v1alpha1", "XPostgreSQLInstance", "", "xpostgresql-92a", nil, nil, map[string]any{
		"compositionRef":         map[string]any{"name": "postgresql-azure-v3"},
		"compositionRevisionRef": map[string]any{"name": "postgresql-azure-v3-abc12"},
		"resourceRefs": []any{
			ref("dbforpostgresql.azure.upbound.io/v1beta1", "FlexibleServer", "postgresql-server"),
			ref("net.crosswatc.io/v1alpha1", "XNetwork", "xnetwork-1"),
			ref("network.azure.upbound.io/v1beta1", "VirtualNetwork", "vnet-gone"),
			ref("unknown.io/v1", "Thing", "t"),
		},
	}, "Ready", "True", "Synced", "True")
	nested := u("net.crosswatc.io/v1alpha1", "XNetwork", "", "xnetwork-1", nil, nil, map[string]any{
		"resourceRefs": []any{ref("network.azure.upbound.io/v1beta1", "PrivateEndpoint", "pe-1")},
	}, "Ready", "True")
	server := u("dbforpostgresql.azure.upbound.io/v1beta1", "FlexibleServer", "", "postgresql-server", nil,
		map[string]any{"crossplane.io/composition-resource-name": "server", "crossplane.io/external-name": "pg-payments-dev"},
		map[string]any{}, "Ready", "True", "Synced", "True")
	pe := u("network.azure.upbound.io/v1beta1", "PrivateEndpoint", "", "pe-1", nil, nil, map[string]any{}, "Ready", "False")

	c := discoveryClient(t, claim, unmanaged, xr, nested, server, pe)
	res, err := c.DiscoverResources(context.Background(), []XRD{pgXRD, netXRD}, mrIndex)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("only managed claims must be discovered, got %d", len(res))
	}
	d := res[0]
	if len(d.MetadataErrors) != 0 || d.Metadata.Owner != "payments" {
		t.Fatalf("metadata: %+v %v", d.Metadata, d.MetadataErrors)
	}
	if d.XR == nil || d.XR.Name != "xpostgresql-92a" || d.XR.Parent != "PostgreSQLInstance/team-payments/pgsql-payments" {
		t.Fatalf("xr: %+v", d.XR)
	}
	if d.XR.CompositionName != "postgresql-azure-v3" || d.XR.CompositionRevision != "postgresql-azure-v3-abc12" {
		t.Fatalf("composition: %+v", d.XR)
	}
	if len(d.Composites) != 1 || d.Composites[0].Parent != "XPostgreSQLInstance/xpostgresql-92a" {
		t.Fatalf("composites: %+v", d.Composites)
	}
	if len(d.MRs) != 2 {
		t.Fatalf("mrs: %+v", d.MRs)
	}
	byName := map[string]Object{}
	for _, m := range d.MRs {
		byName[m.Name] = m
	}
	if s := byName["postgresql-server"]; !s.Ready() || !s.Synced() || s.ExternalName != "pg-payments-dev" || s.CompositionResourceName != "server" {
		t.Fatalf("server: %+v", s)
	}
	if p := byName["pe-1"]; p.Ready() || p.Parent != "XNetwork/xnetwork-1" {
		t.Fatalf("pe: %+v", p)
	}
	if len(d.Missing) != 1 || d.Missing[0].Name != "vnet-gone" || d.Missing[0].Parent != "XPostgreSQLInstance/xpostgresql-92a" {
		t.Fatalf("missing: %+v", d.Missing)
	}
	if len(d.Errors) != 1 {
		t.Fatalf("errors: %v", d.Errors)
	}
	if !d.SecretAvailable {
		t.Fatal("secret must be available")
	}
	if d.Claim.Spec["parameters"] == nil || d.Claim.ObservedGeneration != 1 {
		t.Fatalf("claim: %+v", d.Claim)
	}
}

func TestDiscoverResourcesPendingXRAndMissingSecret(t *testing.T) {
	claim := u("database.crosswatc.io/v1alpha1", "PostgreSQLInstance", "team-payments", "pg-new", managedLabels(), map[string]any{},
		map[string]any{"writeConnectionSecretToRef": map[string]any{"name": "absent"}})
	c := discoveryClient(t, claim)
	res, err := c.DiscoverResources(context.Background(), []XRD{pgXRD}, mrIndex)
	if err != nil || len(res) != 1 {
		t.Fatal(err, len(res))
	}
	d := res[0]
	if d.XR != nil || d.SecretAvailable || len(d.MetadataErrors) != 3 {
		t.Fatalf("got %+v", d)
	}
}

func TestDiscoverNamespacedXR(t *testing.T) {
	bucket := u("storage.crosswatc.io/v1", "Bucket", "team-payments", "logs", managedLabels(), managedAnnotations(), map[string]any{
		"crossplane": map[string]any{
			"compositionRef": map[string]any{"name": "bucket-azure"},
			"resourceRefs":   []any{ref("storage.azure.m.upbound.io/v1beta1", "Account", "logsacct")},
		},
	}, "Ready", "True", "Synced", "True")
	acct := u("storage.azure.m.upbound.io/v1beta1", "Account", "team-payments", "logsacct", nil, nil, map[string]any{}, "Ready", "True", "Synced", "True")
	c := discoveryClient(t, bucket, acct)
	res, err := c.DiscoverResources(context.Background(), []XRD{bucketXRD}, mrIndex)
	if err != nil || len(res) != 1 {
		t.Fatal(err, len(res))
	}
	d := res[0]
	if d.XR == nil || d.XR.Role != RoleComposite || d.XR.CompositionName != "bucket-azure" {
		t.Fatalf("xr: %+v", d.XR)
	}
	if len(d.MRs) != 1 || d.MRs[0].Namespace != "team-payments" {
		t.Fatalf("mrs: %+v", d.MRs)
	}

	one, err := c.DiscoverOne(context.Background(), bucketXRD, []XRD{bucketXRD}, mrIndex, "team-payments", "logs")
	if err != nil || len(one.MRs) != 1 {
		t.Fatalf("DiscoverOne: %v %+v", err, one)
	}
}

func TestProviderOf(t *testing.T) {
	cases := map[string]string{
		"dbforpostgresql.azure.upbound.io/v1beta1": "azure",
		"s3.aws.upbound.io/v1beta1":                "aws",
		"database.crosswatc.io/v1":                 "",
		"::":                                       "",
	}
	for in, want := range cases {
		if got := ProviderOf(in); got != want {
			t.Errorf("ProviderOf(%q) = %q, want %q", in, got, want)
		}
	}
}
