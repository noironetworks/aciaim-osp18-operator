package ciscoaciaim

import (
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/utils/pointer"

    ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
    "github.com/openstack-k8s-operators/lib-common/modules/common/env"
)

// StatefulSet creates an appsv1.StatefulSet object for the CiscoAciAim service.
// instance: The CiscoAciAim CR instance.
// configMapName: The name of the ConfigMap containing configuration files.
// pvcName: The name of the PersistentVolumeClaim for logs.
// configMapChecksum: The checksum of the ConfigMap content to trigger restarts.
func StatefulSet(
    instance *ciscoaciaimv1.CiscoAciAim,
    configMapName string,
    pvcName string,
    configMapChecksum string,
) *appsv1.StatefulSet {
    replicas := int32(2)
    if instance.Spec.Replicas != nil {
        replicas = *instance.Spec.Replicas
    }

    // Get VolumeMounts and Volumes from the dedicated volumes.go file
    volumeMounts := GetVolumeMounts()
    volumes := GetVolumes(configMapName, pvcName)

    envVars := map[string]env.Setter{}
    envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
    trueVal := true

    statefulSet := &appsv1.StatefulSet{
        ObjectMeta: metav1.ObjectMeta{
            Name:      instance.Name,
            Namespace: instance.Namespace,
            Labels:    map[string]string{"app": instance.Name},
        },
        Spec: appsv1.StatefulSetSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": instance.Name},
            },
            ServiceName: instance.Name,
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": instance.Name},
                    Annotations: map[string]string{
                        "configmap-checksum": configMapChecksum,
                    },
                },
                Spec: corev1.PodSpec{
                    SecurityContext: &corev1.PodSecurityContext{
                        FSGroup: pointer.Int64(NeutronUID),
                    },
                    ServiceAccountName: "neutron-neutron",
                    Containers: []corev1.Container{
                        {
                            Name:  "aciaim",
                            Image: instance.Spec.ContainerImage,
                            VolumeMounts: volumeMounts,
                            Command:      []string{"/bin/bash"},
                            Args:  []string{
                               "-c", // ADD THIS -c
                                "export POD_ORDINAL=$(echo $POD_NAME | rev | cut -d'-' -f1 | rev); " +
                                ServiceCommand,
                            },
                            Env: env.MergeEnvs(
                                []corev1.EnvVar{
                                    {
                                        Name: "POD_NAME",
                                        ValueFrom: &corev1.EnvVarSource{
                                            FieldRef: &corev1.ObjectFieldSelector{
                                                FieldPath: "metadata.name",
                                            },
                                        },
                                    },
                                },
                                envVars,
                            ),
                            ImagePullPolicy: "Always",
                            SecurityContext: &corev1.SecurityContext{
                                RunAsUser:    pointer.Int64(NeutronUID),
                                RunAsGroup:   pointer.Int64(NeutronGID),
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

    return statefulSet
}
