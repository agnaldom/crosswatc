package contract

import "testing"

func valid() (map[string]string, map[string]string) {
	return map[string]string{
		LabelManaged:      "true",
		LabelOwner:        "payments",
		LabelEnvironment:  "dev",
		LabelProvider:     "azure",
		LabelResourceType: "postgresql",
	}, map[string]string{
		AnnotationRequestID:       "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T",
		AnnotationRevision:        "4",
		AnnotationRepository:      "crosswatc-org/argocd",
		AnnotationPath:            "claims/dev/pgsql-payments.yaml",
		AnnotationBackstageTaskID: "6f1c",
	}
}

func TestParseValid(t *testing.T) {
	l, a := valid()
	md, errs := Parse(l, a)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if md.Owner != "payments" || md.Revision != 4 || md.BackstageTaskID != "6f1c" || md.ArgoApp != "" {
		t.Fatalf("unexpected metadata: %+v", md)
	}
}

func TestParseViolations(t *testing.T) {
	cases := map[string]struct {
		mutate func(l, a map[string]string)
		key    string
	}{
		"not managed":       {func(l, _ map[string]string) { l[LabelManaged] = "false" }, LabelManaged},
		"missing owner":     {func(l, _ map[string]string) { delete(l, LabelOwner) }, LabelOwner},
		"bad provider":      {func(l, _ map[string]string) { l[LabelProvider] = "gcp" }, LabelProvider},
		"bad request id":    {func(_, a map[string]string) { a[AnnotationRequestID] = "req-1" }, AnnotationRequestID},
		"missing path":      {func(_, a map[string]string) { delete(a, AnnotationPath) }, AnnotationPath},
		"bad revision":      {func(_, a map[string]string) { a[AnnotationRevision] = "x" }, AnnotationRevision},
		"negative revision": {func(_, a map[string]string) { a[AnnotationRevision] = "-1" }, AnnotationRevision},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			l, a := valid()
			tc.mutate(l, a)
			_, errs := Parse(l, a)
			if len(errs) != 1 || errs[0].Key != tc.key {
				t.Fatalf("want single error on %s, got %v", tc.key, errs)
			}
		})
	}
}

func TestParseEmpty(t *testing.T) {
	_, errs := Parse(nil, nil)
	if len(errs) != 8 {
		t.Fatalf("want 8 errors (managed + 4 labels + 3 annotations), got %d: %v", len(errs), errs)
	}
}
