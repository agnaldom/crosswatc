package kube

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
)

// IsNotFound reports a Kubernetes 404.
func IsNotFound(err error) bool { return apierrors.IsNotFound(err) }

// IsMissingAPI reports that the API (CRD) itself is not served, e.g. Crossplane v2 kinds on a v1 cluster:
// callers degrade to an empty result instead of failing.
func IsMissingAPI(err error) bool { return apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) }
