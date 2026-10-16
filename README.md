# Cisco ACI AIM OpenStack Platform 18 Operator

This repository contains the Kubernetes Operator for integrating Cisco Application Centric Infrastructure (ACI) with OpenStack Platform 18 (OSP18) deployments. This operator aims to automate the deployment and management of the ACI Integration Module (AIM) within an OpenStack environment running on Openshift clusters

### Deployment Steps
The files below are at: https://github.com/noironetworks/aciaim-osp18-operator/tree/main/config/aim_configs

1.  **Apply Custom Resource Definitions (CRDs)**:
    First, apply the Custom Resource Definition for the `CiscoACI` kind. This tells Kubernetes about the new resource type the operator will manage.

    ```bash
    oc apply -f api.cisco.com_ciscoaciaims.yaml
    ```

2.  **Apply RBAC Permissions**:
    Next, set up the necessary Role-Based Access Control (RBAC) permissions for the operator. This includes a ClusterRole and a RoleBinding.

    ```bash
    oc apply -f ciscoaciaim-rbac.yaml
    ```

3.  **Create the Custom Resource (CR)**:
    Once the operator is running, create an instance of the `CiscoACI` custom resource. This custom resource defines the desired state of your ACI AIM integration. You will need to edit `ciscoaciaim-cr.yaml` to include your specific ACI and OpenStack configuration details (e.g., APIC IP addresses, credentials).

    ```bash
    oc apply -n openstack -f ciscoaciaim-cr.yaml
    ```

4.  **Deploy the Operator Bundle**:                                                
    Deploy the operator's deployment manifest. This manifest typically includes the operator's Deployment, Service Account, and other necessary components. Ensure that the image specified within `aim_bundle_deployment.yaml` points to the correct pre-built operator image in your container registry. 
                                                                                 
    ```bash                                                                        
    oc apply -f aim_operator_deployment.yaml                                         
    ```

### Persistent Log Storage

When persistent log storage is enabled, each AIM StatefulSet pod uses its own log claim. Changes to immutable StatefulSet fields, such as claim-template fields other than requested storage size or pod-management-policy changes, can cause temporary AIM service downtime while the old StatefulSet and pods are removed and the replacement starts.

Increasing requested log storage expands existing claims in place; shrinking storage is rejected.

The previous shared claim named `<instance-name>-log-pvc` is retained after migration to per-pod claims. Its logs remain on the claim and it continues to consume storage, but the new pods do not mount it. To recover logs, mount the claim in a temporary pod in the same namespace and copy the files out (for example, with `oc cp`). Delete the claim manually only after confirming its data is no longer needed; the operator does not delete it automatically.
