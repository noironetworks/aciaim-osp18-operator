/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package functional_test

import (
	"math/rand"
	"time"

	aciaimv1alpha1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	. "github.com/onsi/gomega" //revive:disable:dot-imports
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 30
	interval = timeout / 100
)

// RandStringBytes generates random string for test namespace
func RandStringBytes(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// GetDefaultCiscoAciAimSpec returns a default spec for testing
func GetDefaultCiscoAciAimSpec(namespace string) map[string]any {
	return map[string]any{
		"containerImage": "test-registry/openstack-ciscoaci-aim:latest",
		"replicas":       int64(1),
		"ACIAimDebug":    true,
		"logPersistence": map[string]any{
			"size": "1Gi",
		},
		"aciConnection": map[string]any{
			"ACIApicHosts":    "10.0.0.1",
			"ACIApicUsername": "admin",
			"ACIApicPassword": "password",
			"ACIApicSystemId": "test-system",
		},
	}
}

// CreateCiscoAciAim creates a CiscoAciAim CR
func CreateCiscoAciAim(namespace string, name string, spec map[string]any) *aciaimv1alpha1.CiscoAciAim {
	cr := &aciaimv1alpha1.CiscoAciAim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "api.cisco.com/v1alpha1",
			Kind:       "CiscoAciAim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
	return cr
}

// GetCiscoAciAim retrieves a CiscoAciAim CR
func GetCiscoAciAim(name types.NamespacedName) *aciaimv1alpha1.CiscoAciAim {
	instance := &aciaimv1alpha1.CiscoAciAim{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

// DeleteCiscoAciAim deletes a CiscoAciAim CR
func DeleteCiscoAciAim(name types.NamespacedName) {
	Eventually(func(g Gomega) {
		cr := &aciaimv1alpha1.CiscoAciAim{}
		err := k8sClient.Get(ctx, name, cr)
		if err == nil {
			g.Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
		}
	}, timeout, interval).Should(Succeed())
}

// GetStatefulSet retrieves a StatefulSet
func GetStatefulSet(name types.NamespacedName) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, sts)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return sts
}

// GetConfigMap retrieves a ConfigMap
func GetConfigMap(name types.NamespacedName) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, cm)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return cm
}

// GetSecret retrieves a Secret
func GetSecret(name types.NamespacedName) *corev1.Secret {
	secret := &corev1.Secret{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, secret)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return secret
}

// StatefulSetExists checks if a StatefulSet exists
func StatefulSetExists(name types.NamespacedName) bool {
	sts := &appsv1.StatefulSet{}
	err := k8sClient.Get(ctx, name, sts)
	return err == nil
}

// SimulateStatefulSetReady simulates a StatefulSet becoming ready
func SimulateStatefulSetReady(name types.NamespacedName) {
	Eventually(func(g Gomega) {
		sts := GetStatefulSet(name)
		sts.Status.ReadyReplicas = *sts.Spec.Replicas
		sts.Status.Replicas = *sts.Spec.Replicas
		g.Expect(k8sClient.Status().Update(ctx, sts)).To(Succeed())
	}, timeout, interval).Should(Succeed())
}

// CreateSecret creates a test secret
func CreateSecret(namespace string, name string, data map[string][]byte) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
	return secret
}

// ListStatefulSets lists all StatefulSets with given labels
func ListStatefulSets(namespace string, labels map[string]string) *appsv1.StatefulSetList {
	stsList := &appsv1.StatefulSetList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}
	if labels != nil {
		listOpts = append(listOpts, client.MatchingLabels(labels))
	}
	Expect(k8sClient.List(ctx, stsList, listOpts...)).Should(Succeed())
	return stsList
}
