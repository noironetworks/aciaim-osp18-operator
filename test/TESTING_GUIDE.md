# ACI AIM Operator Testing Guide

## 🚀 Quick Start

**Run tests locally (no cluster):**
```bash
make test
```

**Run integration tests (OpenStack cluster):**
```bash
# Deploy operator first, then:
make test-kuttl
```

---

## 🔧 Test Types

### 1. **Functional Tests** - No Cluster Needed ⚡
Tests your operator's controller logic locally without a real cluster.

**Run:**
```bash
make test
```

**What it tests:**
- CR creation, updates, deletion
- Spec validation
- Controller reconciliation logic

**Time:** ~30 seconds

---

### 2. **KUTTL Tests** - OpenStack Cluster Required 🔧
End-to-end integration tests on a real OpenStack cluster.

**Run:**
```bash
make test-kuttl
```

**What it tests:**
1. Setup namespaces
2. Create CiscoAciAim CR
3. StatefulSet creation
4. ConfigMap creation
5. Secret creation
6. PVC creation
7. Replica updates
8. Cleanup & finalizers

**Time:** ~5 minutes

---

### 3. **E2E Tests** - Kind Cluster Required
Full deployment testing.

**Run:**
```bash
make test-e2e
```

---

## 🚀 Running Tests

### Functional Tests (No Cluster)

**Step 1: Run the tests**
```bash
make test
```

That's it! The test will:
- Download envtest binaries automatically
- Start a fake Kubernetes API
- Run all controller tests
- Generate coverage report (`cover.out`)

**View coverage:**
```bash
go tool cover -html=cover.out
```

---

### KUTTL Tests (OpenStack Cluster Required)

**Prerequisites:**
- OpenStack on OpenShift cluster
- `kubectl-kuttl` installed: `brew install kudobuilder/tap/kuttl`

**Step 1: Connect to cluster**
```bash
oc login https://your-openstack-cluster:6443
```

**Step 2: Deploy operator**
```bash
oc apply -f config/crd/bases/api.cisco.com_ciscoaciaims.yaml
oc apply -f config/aim_configs/ciscoaciaim-rbac.yaml
oc apply -f config/aim_configs/aim_operator_deployment.yaml
```

**Step 3: Run tests**
```bash
make test-kuttl
```

**Verbose output:**
```bash
kubectl-kuttl test --config test/kuttl/kuttl-test.yaml --verbose
```

**Run specific test:**
```bash
kubectl-kuttl test --config test/kuttl/kuttl-test.yaml --test 01-create-ciscoaciaim-cr
```

**Skip cleanup (for debugging):**
```bash
kubectl-kuttl test --config test/kuttl/kuttl-test.yaml --skip-delete
```

---

## 🔧 Customizing Tests

### Update KUTTL Test Configuration
Edit `test/kuttl/e2e/01-create-ciscoaciaim-cr/00-cr.yaml` with your ACI settings:

```yaml
spec:
  aciConnection:
    ACIApicHosts: "YOUR_ACTUAL_APIC_IP"
    ACIApicPassword: "YOUR_PASSWORD"
    ACIApicSystemId: "YOUR_SYSTEM_ID"
  aciFabric:
    ACIApicEntityProfile: "YOUR_ENTITY_PROFILE"
    ACIVpcPairs: ["101:102"]
```

---

## 🐛 Troubleshooting

### Functional Tests

**Tests fail to find CRDs:**
```bash
# Regenerate CRDs
make manifests
make test
```

**envtest fails to download:**
```bash
# Manual download
make envtest
./bin/setup-envtest use $(ENVTEST_K8S_VERSION)
```

---

### KUTTL Tests

**Test timeout:**
Edit `test/kuttl/kuttl-test.yaml` and increase timeout:
```yaml
timeout: 600  # 10 minutes instead of 5
```

**Operator not creating resources:**
```bash
# Check operator is running
oc get pods -n openstack-operators

# Check operator logs
oc logs -n openstack-operators deployment/ciscoaciaim-operator -f

# Check RBAC permissions
oc auth can-i create statefulsets --as=system:serviceaccount:openstack-operators:ciscoaciaim-operator-controller-manager
```

**Check test resources:**
```bash
# View CR
oc get ciscoaciaim -n openstack
oc describe ciscoaciaim ciscoaci-aim-test -n openstack

# View created resources
oc get all -n openstack
oc get statefulset,configmap,secret,pvc -n openstack -l app=ciscoaci-aim
```

**Clean up after failed test:**
```bash
oc delete ciscoaciaim ciscoaci-aim-test -n openstack
oc delete namespace openstack openstack-operators
```

---

## 📖 Additional Resources

- **KUTTL Documentation**: https://kuttl.dev/
- **envtest Guide**: https://book.kubebuilder.io/reference/envtest.html
