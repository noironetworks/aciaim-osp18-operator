package ciscoaciaim

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
)

// LogPVC creates a corev1.PersistentVolumeClaim object for the CiscoAciAim logs.
// instance: The CiscoAciAim CR instance.
func LogPVC(
	instance *ciscoaciaimv1.CiscoAciAim,
) (*corev1.PersistentVolumeClaim, error) {
	pvcName := instance.Name + "-log-pvc"

	storageSize, err := resource.ParseQuantity(instance.Spec.LogPersistence.Size)
	if err != nil {
		return nil, err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: instance.Namespace,
			Labels:    map[string]string{"app": instance.Name},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageSize,
				},
			},
		},
	}

	if instance.Spec.LogPersistence.StorageClassName != "" {
		pvc.Spec.StorageClassName = &instance.Spec.LogPersistence.StorageClassName
	}

	return pvc, nil
}
