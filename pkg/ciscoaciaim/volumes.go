package ciscoaciaim

import (
    corev1 "k8s.io/api/core/v1"
    "k8s.io/utils/pointer"
)

// Get a common set of VolumeMounts for AIM containers
func GetVolumeMounts() []corev1.VolumeMount {
    return []corev1.VolumeMount{
        {
            Name:      "config-volume",
            MountPath: "/var/lib/kolla/config_files/src/etc/aim",
            ReadOnly:  true,
        },
        {
            Name:      "config-kolla",
            MountPath: "/var/lib/kolla/config_files/",
            ReadOnly:  true,
        },
        {
            Name:      "rabbitmq-ca",
            MountPath: "/etc/pki/ca-trust/extracted/pem/",
            ReadOnly:  true,
        },
        {
            Name:      "aim-logs",
            MountPath: "/var/log/aim",
            ReadOnly:  false, // Needs write access for logs
        },
        {
            Name:      "init-script-volume",
            MountPath: "/etc/aim/scripts",
            ReadOnly:  true,
        },
    }
}

// Get a common set of Volumes for AIM pods
// configMapName: The name of the ConfigMap that holds the configuration files.
// pvcName: The name of the PersistentVolumeClaim for logs.
func GetVolumes(configMapName string, pvcName string) []corev1.Volume {
    return []corev1.Volume{
        {
            Name: "config-volume",
            VolumeSource: corev1.VolumeSource{
                ConfigMap: &corev1.ConfigMapVolumeSource{
                    LocalObjectReference: corev1.LocalObjectReference{
                        Name: configMapName,
                    },
                    Items: []corev1.KeyToPath{
                        {Key: "aim.conf", Path: "aim.conf"},
                        {Key: "aim_supervisord.conf", Path: "aim_supervisord.conf"},
                        {Key: "aim_healthcheck.sh", Path: "aim_healthcheck"},
                        {Key: "aimctl.conf", Path: "aimctl.conf"},
                    },
                    DefaultMode: pointer.Int32(0755),
                },
            },
        },
        {
            Name: "init-script-volume",
            VolumeSource: corev1.VolumeSource{
                ConfigMap: &corev1.ConfigMapVolumeSource{
                    LocalObjectReference: corev1.LocalObjectReference{
                        Name: configMapName,
                    },
                    Items: []corev1.KeyToPath{
                        {Key: "init.sh", Path: "init.sh"},
                    },
                    DefaultMode: pointer.Int32(0755),
                },
            },
        },
        {
            Name: "config-kolla",
            VolumeSource: corev1.VolumeSource{
                ConfigMap: &corev1.ConfigMapVolumeSource{
                    LocalObjectReference: corev1.LocalObjectReference{
                        Name: configMapName,
                    },
                    Items: []corev1.KeyToPath{
                        {Key: "kolla_config.json", Path: "config.json"},
                    },
                },
            },
        },
        {
            Name: "rabbitmq-ca",
            VolumeSource: corev1.VolumeSource{
                Secret: &corev1.SecretVolumeSource{
                    SecretName: "rabbitmq-ca-secret",
                },
            },
        },
        {
            Name: "aim-logs",
            VolumeSource: corev1.VolumeSource{
                PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
                    ClaimName: pvcName,
                },
            },
        },
    }
}
