package ciscoaciaim

import (
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1" // **IMPORTANT: Adjust this import path**
)

// ConfigMap creates a corev1.ConfigMap object for the CiscoAciAim service.
// instance: The CiscoAciAim CR instance.
// configData: A map of filename to content for the ConfigMap.
func ConfigMap(
    instance *ciscoaciaimv1.CiscoAciAim,
    configData map[string]string,
) *corev1.ConfigMap {
    configMapName := instance.Name + "-config"

    configMap := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      configMapName,
            Namespace: instance.Namespace,
            Labels:    map[string]string{"app": instance.Name},
        },
        Data: configData,
    }

    return configMap
}
