package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *CiscoAciAimReconciler) ensureLogPVCStorage(
	ctx context.Context,
	instance *ciscoaciaimv1.CiscoAciAim,
) error {
	if instance.Spec.LogPersistence == nil {
		return nil
	}

	desiredSize, err := resource.ParseQuantity(instance.Spec.LogPersistence.Size)
	if err != nil {
		return fmt.Errorf("invalid log persistence size %q: %w", instance.Spec.LogPersistence.Size, err)
	}

	claims := &corev1.PersistentVolumeClaimList{}
	if err := r.List(
		ctx,
		claims,
		client.InNamespace(instance.Namespace),
		client.MatchingLabels{"app": instance.Name},
	); err != nil {
		return fmt.Errorf("list log persistent volume claims: %w", err)
	}

	claimPrefix := fmt.Sprintf("aim-logs-%s-", instance.Name)
	for i := range claims.Items {
		claim := &claims.Items[i]
		if !strings.HasPrefix(claim.Name, claimPrefix) {
			continue
		}

		changed, err := prepareLogPVCResize(
			claim,
			desiredSize,
			instance.Spec.LogPersistence.StorageClassName,
		)
		if err != nil {
			return fmt.Errorf("prepare log persistent volume claim %s: %w", claim.Name, err)
		}
		if !changed {
			continue
		}

		if err := r.Update(ctx, claim); err != nil {
			return fmt.Errorf("resize log persistent volume claim %s: %w", claim.Name, err)
		}
	}

	return nil
}

func prepareLogPVCResize(
	claim *corev1.PersistentVolumeClaim,
	desiredSize resource.Quantity,
	desiredStorageClass string,
) (bool, error) {
	if desiredStorageClass != "" &&
		(claim.Spec.StorageClassName == nil || *claim.Spec.StorageClassName != desiredStorageClass) {
		return false, fmt.Errorf("storage class cannot be changed to %q", desiredStorageClass)
	}

	currentSize := claim.Spec.Resources.Requests.Storage()
	if currentSize != nil {
		switch currentSize.Cmp(desiredSize) {
		case 1:
			return false, fmt.Errorf(
				"storage cannot be reduced from %s to %s",
				currentSize.String(),
				desiredSize.String(),
			)
		case 0:
			return false, nil
		}
	}

	if claim.Spec.Resources.Requests == nil {
		claim.Spec.Resources.Requests = corev1.ResourceList{}
	}
	claim.Spec.Resources.Requests[corev1.ResourceStorage] = desiredSize

	return true, nil
}
