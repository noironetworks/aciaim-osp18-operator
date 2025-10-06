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
	aciaimv1alpha1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

var _ = Describe("CiscoAciAim controller", func() {
	var crName string
	var ciscoAciAimName types.NamespacedName

	BeforeEach(func() {
		crName = "test-aciaim"
		ciscoAciAimName = types.NamespacedName{
			Name:      crName,
			Namespace: namespace,
		}
	})

	AfterEach(func() {
		// Cleanup
		DeleteCiscoAciAim(ciscoAciAimName)
	})

	When("A CiscoAciAim CR is created", func() {
		BeforeEach(func() {
			cr := &aciaimv1alpha1.CiscoAciAim{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "api.cisco.com/v1alpha1",
					Kind:       "CiscoAciAim",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: namespace,
				},
				Spec: aciaimv1alpha1.CiscoAciAimSpec{
					ContainerImage: "test-registry/openstack-ciscoaci-aim:latest",
					Replicas:       ptr.To(int32(1)),
					ACIAimDebug:    true,
					LogPersistence: aciaimv1alpha1.LogPersistenceSpec{
						Size: "1Gi",
					},
					AciConnection: aciaimv1alpha1.AciConnectionSpec{
						ACIApicHosts:    "10.0.0.1",
						ACIApicUsername: "admin",
						ACIApicPassword: "password",
						ACIApicSystemId: "test-system",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
		})

		It("should create the CR successfully", func() {
			cr := GetCiscoAciAim(ciscoAciAimName)
			Expect(cr).NotTo(BeNil())
			Expect(*cr.Spec.Replicas).To(Equal(int32(1)))
			Expect(cr.Spec.ACIAimDebug).To(BeTrue())
		})

		It("should have the correct spec values", func() {
			cr := GetCiscoAciAim(ciscoAciAimName)
			Expect(cr.Spec.ContainerImage).To(Equal("test-registry/openstack-ciscoaci-aim:latest"))
			Expect(cr.Spec.AciConnection.ACIApicHosts).To(Equal("10.0.0.1"))
			Expect(cr.Spec.AciConnection.ACIApicUsername).To(Equal("admin"))
			Expect(cr.Spec.LogPersistence.Size).To(Equal("1Gi"))
		})
	})

	When("A CiscoAciAim CR with custom configuration is created", func() {
		BeforeEach(func() {
			cr := &aciaimv1alpha1.CiscoAciAim{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "api.cisco.com/v1alpha1",
					Kind:       "CiscoAciAim",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: namespace,
				},
				Spec: aciaimv1alpha1.CiscoAciAimSpec{
					ContainerImage:             "test-registry/openstack-ciscoaci-aim:custom",
					Replicas:                   ptr.To(int32(2)),
					ACIAimDebug:                false,
					ACIOptimizedMetadata:       true,
					ACIScopeNames:              true,
					ACIScopeInfra:              true,
					AciGen1HwGratArps:          false,
					AciEnableFaultSubscription: true,
					LogPersistence: aciaimv1alpha1.LogPersistenceSpec{
						Size: "5Gi",
					},
					AciConnection: aciaimv1alpha1.AciConnectionSpec{
						ACIApicHosts:             "10.0.0.1,10.0.0.2",
						ACIApicUsername:          "admin",
						ACIApicPassword:          "password",
						ACIApicSystemId:          "prod-system",
						ACIApicSystemIdMaxLength: 16,
					},
					AciFabric: &aciaimv1alpha1.AciFabricSpec{
						ACIApicEntityProfile: "prod-entity-profile",
						ACIVpcPairs:          []string{"101:102", "103:104"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
		})

		It("should create the CR with custom values", func() {
			cr := GetCiscoAciAim(ciscoAciAimName)
			Expect(*cr.Spec.Replicas).To(Equal(int32(2)))
			Expect(cr.Spec.ACIOptimizedMetadata).To(BeTrue())
			Expect(cr.Spec.AciConnection.ACIApicSystemId).To(Equal("prod-system"))
		})

		It("should have fabric configuration", func() {
			cr := GetCiscoAciAim(ciscoAciAimName)
			Expect(cr.Spec.AciFabric).NotTo(BeNil())
			Expect(cr.Spec.AciFabric.ACIApicEntityProfile).To(Equal("prod-entity-profile"))
			Expect(cr.Spec.AciFabric.ACIVpcPairs).To(HaveLen(2))
		})
	})

	When("A CiscoAciAim CR is updated", func() {
		BeforeEach(func() {
			cr := &aciaimv1alpha1.CiscoAciAim{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "api.cisco.com/v1alpha1",
					Kind:       "CiscoAciAim",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: namespace,
				},
				Spec: aciaimv1alpha1.CiscoAciAimSpec{
					ContainerImage: "test-registry/openstack-ciscoaci-aim:v1",
					Replicas:       ptr.To(int32(1)),
					LogPersistence: aciaimv1alpha1.LogPersistenceSpec{
						Size: "1Gi",
					},
					AciConnection: aciaimv1alpha1.AciConnectionSpec{
						ACIApicHosts:    "10.0.0.1",
						ACIApicUsername: "admin",
						ACIApicPassword: "password",
						ACIApicSystemId: "test-system",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
		})

		It("should allow replica count updates", func() {
			Eventually(func(g Gomega) {
				cr := GetCiscoAciAim(ciscoAciAimName)
				cr.Spec.Replicas = ptr.To(int32(3))
				g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			cr := GetCiscoAciAim(ciscoAciAimName)
			Expect(*cr.Spec.Replicas).To(Equal(int32(3)))
		})

		It("should allow debug flag updates", func() {
			Eventually(func(g Gomega) {
				cr := GetCiscoAciAim(ciscoAciAimName)
				cr.Spec.ACIAimDebug = true
				g.Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			cr := GetCiscoAciAim(ciscoAciAimName)
			Expect(cr.Spec.ACIAimDebug).To(BeTrue())
		})
	})

	When("A CiscoAciAim CR is deleted", func() {
		BeforeEach(func() {
			cr := &aciaimv1alpha1.CiscoAciAim{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "api.cisco.com/v1alpha1",
					Kind:       "CiscoAciAim",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      crName,
					Namespace: namespace,
				},
				Spec: aciaimv1alpha1.CiscoAciAimSpec{
					ContainerImage: "test-registry/openstack-ciscoaci-aim:latest",
					Replicas:       ptr.To(int32(1)),
					LogPersistence: aciaimv1alpha1.LogPersistenceSpec{
						Size: "1Gi",
					},
					AciConnection: aciaimv1alpha1.AciConnectionSpec{
						ACIApicHosts:    "10.0.0.1",
						ACIApicUsername: "admin",
						ACIApicPassword: "password",
						ACIApicSystemId: "test-system",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
		})

		It("should be removed from the cluster", func() {
			DeleteCiscoAciAim(ciscoAciAimName)
			Eventually(func() bool {
				cr := &aciaimv1alpha1.CiscoAciAim{}
				err := k8sClient.Get(ctx, ciscoAciAimName, cr)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
