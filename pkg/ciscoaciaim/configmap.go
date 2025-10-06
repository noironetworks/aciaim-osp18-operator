package ciscoaciaim

import (
	"crypto/sha256"
	"encoding/hex"
	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
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

// This ensures a unique identifier for the ConfigMap's content,
// which can be used to trigger Deployment rollouts.
func GenerateConfigMapChecksum(configData map[string]string) string {
	// Sort keys to ensure consistent hash generation regardless of map iteration order
	keys := make([]string, 0, len(configData))
	for k := range configData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	hasher := sha256.New()
	for _, k := range keys {
		hasher.Write([]byte(k))
		hasher.Write([]byte(configData[k]))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}
