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
                                    
