package kube

import apierrors "k8s.io/apimachinery/pkg/api/errors"

// IsNotFound reports a Kubernetes 404.
func IsNotFound(err error) bool { return apierrors.IsNotFound(err) }
