// =======================================================================================
// 📄 service_util.go
//
// ✨ Description:
//     1️⃣ GetServiceNameFromPod(): Match a Service based on a Pod's label selector.
//     2️⃣ CheckServiceEndpointStatus(): Check whether a Service has ready Endpoints.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package utils

import (
	"context"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetServiceNameFromPod attempts to find the Service associated with the given Pod
// by matching its labels with Service selectors.
//
// 🔹 Label selectors are critical for Service-to-Pod association.
func GetServiceNameFromPod(ctx context.Context, pod *corev1.Pod) (string, error) {
	cli := GetClient()

	var serviceList corev1.ServiceList
	if err := cli.List(ctx, &serviceList, client.InNamespace(pod.Namespace)); err != nil {
		Error(ctx, "❌ Failed to list Services",
			zap.String("namespace", pod.Namespace),
			zap.Error(err),
		)
		return "", err
	}

	// 🔀 Match Pod labels against each Service's selector
	for _, svc := range serviceList.Items {
		match := true
		for key, val := range svc.Spec.Selector {
			if podVal, ok := pod.Labels[key]; !ok || podVal != val {
				match = false
				break
			}
		}
		if match {
			Info(ctx, "✅ Matched Service found",
				zap.String("service", svc.Name),
				zap.String("pod", pod.Name),
			)

			CheckServiceEndpointStatus(ctx, pod.Namespace, svc.Name)
			return svc.Name, nil
		}
	}

	return "", nil // No matching Service found
}

// CheckServiceEndpointStatus verifies whether the specified Service has any ready Endpoints.
func CheckServiceEndpointStatus(ctx context.Context, namespace, name string) {
	cli := GetClient()

	var endpoints corev1.Endpoints
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &endpoints); err != nil {
		Warn(ctx, "⚠️ Failed to retrieve Endpoints",
			zap.String("service", name),
			zap.Error(err),
		)
		return
	}

	readyCount := 0
	for _, subset := range endpoints.Subsets {
		readyCount += len(subset.Addresses)
	}

	if readyCount == 0 {
		Warn(ctx, " No ready Pods found in Endpoints",
			zap.String("service", name),
			zap.String("namespace", namespace),
		)
	} else {
		Info(ctx, "✅ Endpoints are healthy",
			zap.String("service", name),
			zap.Int("ready", readyCount),
		)
	}
}
