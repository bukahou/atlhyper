// =======================================================================================
// 📄 deployment_util.go
//
// ✨ Description:
//     1️⃣ GetDeploymentNameFromPod(): Trace the Deployment name a Pod belongs to via ReplicaSet ownerRef.
//     2️⃣ CheckDeploymentReplicaStatusByName(): Fetch and verify replica state for a specific Deployment.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package utils

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
)

// GetDeploymentNameFromPod attempts to extract the Deployment name associated with a given Pod.
// 🧠 Logic: Pod ➜ ReplicaSet ➜ Deployment (via ownerReference chain)
//
// 📍 Use case:
//   - When a Pod is abnormal, determine which Deployment it belongs to,
//     to aggregate issues, send alerts, or trigger scaling.
func GetDeploymentNameFromPod(ctx context.Context, pod *corev1.Pod) (string, error) {
	cli := GetClient() // ✅ Use global controller-runtime client

	// 🔁 Iterate through the Pod's ownerReferences
	for _, owner := range pod.OwnerReferences {
		// 🔍 If the owner is a ReplicaSet, continue tracing
		if owner.Kind == "ReplicaSet" {

			// ✅ 1️⃣ Retrieve the corresponding ReplicaSet object
			rs := &appsv1.ReplicaSet{}
			err := cli.Get(ctx, client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      owner.Name, // 🧭 Extracted from Pod owner
			}, rs)
			if err != nil {
				// ❌ Failed to fetch ReplicaSet (possibly deleted)
				Error(ctx, "❌ Failed to retrieve ReplicaSet", zap.String("replicaSet", owner.Name), zap.Error(err))
				return "", fmt.Errorf("failed to get replicaset %s: %w", owner.Name, err)
			}

			// 🔁 Continue tracing: check if this ReplicaSet is owned by a Deployment
			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind == "Deployment" {
					deployName := rsOwner.Name

					// 🟢 Log success: traced to Deployment
					Info(ctx, "✅ Successfully resolved Deployment owner",
						zap.String("pod", pod.Name),
						zap.String("deployment", deployName),
					)

					// 🔁 Optionally check replica status immediately
					CheckDeploymentReplicaStatusByName(ctx, pod.Namespace, deployName)

					return deployName, nil
				}
			}

			// ❌ No Deployment owner found in ReplicaSet
			return "", errors.New("replicaSet has no Deployment ownerRef")
		}
	}

	// ❌ No valid ReplicaSet owner found for the Pod
	return "", errors.New("pod has no valid ReplicaSet ownerRef")
}

// CheckDeploymentReplicaStatusByName checks the replica status of the given Deployment.
//
// 📍 Use case:
//   - After identifying the Deployment a failing Pod belongs to,
//     this function verifies whether there are missing or unavailable replicas.
func CheckDeploymentReplicaStatusByName(ctx context.Context, namespace string, name string) {
	cli := GetClient()

	var deployment appsv1.Deployment
	// 🔍 Retrieve the Deployment object
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &deployment); err != nil {
		// ❌ Failed to fetch (may be deleted or not yet available)
		Error(ctx, "❌ Failed to get Deployment status",
			zap.String("deployment", name),
			zap.Error(err),
		)
		return
	}

	// ✅ Extract replica status information
	desired := *deployment.Spec.Replicas                 // Desired replicas
	ready := deployment.Status.ReadyReplicas             // Ready replicas
	unavailable := deployment.Status.UnavailableReplicas // Currently unavailable pods

	// 🚨 Case 1: Fewer ready replicas than desired
	if ready < desired {
		Warn(ctx, "🚨 Deployment has insufficient ready replicas",
			zap.String("deployment", name),
			zap.Int32("desired", desired),
			zap.Int32("ready", ready),
		)
	}

	// ⚠️ Case 2: There are unavailable replicas
	if unavailable > 0 {
		Warn(ctx, "⚠️ Deployment contains unavailable replicas",
			zap.String("deployment", name),
			zap.Int32("unavailable", unavailable),
		)
	}
}
