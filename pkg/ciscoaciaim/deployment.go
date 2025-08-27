package ciscoaciaim

import (
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/utils/pointer"

    ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1" // **IMPORTANT: Adjust this import path**
    "github.com/openstack-k8s-operators/lib-common/modules/common/env"
)

// Constants (from pkg/ciscoaciaim/constants.go) would be implicitly available here
// if this file is in the same package.

// Deployment creates an appsv1.Deployment object for the CiscoAciAim service.
func Deployment(
    instance *ciscoaciaimv1.CiscoAciAim,
    configMapName string,
    pvcName string,
) *appsv1.Deployment {
    replicas := int32(2)
    if instance.Spec.Replicas != nil {
        replicas = *instance.Spec.Replicas
    }

    // Get VolumeMounts and Volumes from the dedicated volumes.go file
    volumeMounts := GetVolumeMounts()
    volumes := GetVolumes(configMapName, pvcName)

    envVars := map[string]env.Setter{}
    envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
    args := []string{"-c", ServiceCommand} // ServiceCommand is from constants.go
    trueVal := true

    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      instance.Name,
            Namespace: instance.Namespace,
            Labels:    map[string]string{"app": instance.Name},
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": instance.Name},
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": instance.Name},
                },
                Spec: corev1.PodSpec{
                    SecurityContext: &corev1.PodSecurityContext{
                        FSGroup: pointer.Int64(NeutronUID), // NeutronUID is from constants.go
                    },
                    ServiceAccountName: "neutron-neutron",
                    Containers: []corev1.Container{
                        {
                            Name:  "aciaim",
                            Image: instance.Spec.ContainerImage,
                            VolumeMounts: volumeMounts,
                            Command:      []string{"/bin/bash"},
                            Args:         args,
                            Env:          env.MergeEnvs([]corev1.EnvVar{}, envVars),
                            ImagePullPolicy: "Always",
                            SecurityContext: &corev1.SecurityContext{
                                RunAsUser:    pointer.Int64(NeutronUID),
                                RunAsGroup:   pointer.Int64(NeutronGID), // NeutronGID is from constants.go
                                RunAsNonRoot: &trueVal,
                            },
                            Lifecycle: &corev1.Lifecycle{
                                PostStart: &corev1.LifecycleHandler{
                                    Exec: &corev1.ExecAction{
                                        Command: []string{
                                            "/bin/sh",
                                            "-c",
                                            "/etc/aim/scripts/init.sh",
                                        },
                                    },
                                },
                            },
                            LivenessProbe: &corev1.Probe{
                                ProbeHandler: corev1.ProbeHandler{
                                    Exec: &corev1.ExecAction{
                                        Command: []string{"/etc/aim/aim_healthcheck"},
                                    },
                                },
                                InitialDelaySeconds: 30,
                                PeriodSeconds:       10,
                            },
                        },
                    },
                    Volumes: volumes,
                },
            },
        },
    }

    return deployment
}
