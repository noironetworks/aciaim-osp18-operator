# Optional AIM Log Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow a CiscoAciAim resource to omit `logPersistence` and run with container-native stdout/stderr logging without creating or mounting a log PVC.

**Architecture:** Represent persistence with an optional pointer in the CR API. Resource builders and reconciliation branch on pointer presence. Render Supervisor, Kolla, and initialization templates from one `LogToDisk` value so the pod configuration and process logging mode cannot diverge.

**Tech Stack:** Go 1.24.1, Kubebuilder/controller-runtime, Kubernetes StatefulSet and PVC APIs, Go text templates, Ginkgo/Gomega envtest.

---

### Task 1: Make the API field optional

**Files:**
- Modify: `api/v1alpha1/ciscoaciaim_types.go`
- Test: `test/functional/ciscoaciaim_controller_test.go`

- [ ] **Step 1: Add a functional admission test**

Create a CiscoAciAim object without `LogPersistence`, submit it through the
envtest API server, and assert creation succeeds. Retrieve it and assert
`Spec.LogPersistence` is nil.

- [ ] **Step 2: Run the focused functional test and verify RED**

Run the functional package with the current generated CRD. Expect admission to
reject the object because `spec.logPersistence` is required.

- [ ] **Step 3: Change the source API type**

Change the field to:

```go
// +kubebuilder:validation:Optional
LogPersistence *LogPersistenceSpec `json:"logPersistence,omitempty"`
```

Change `LogPersistenceSpec.Size` to a required, non-omitempty string while
leaving `StorageClassName` optional.

### Task 2: Skip PVC and volume wiring when persistence is absent

**Files:**
- Modify: `internal/controller/ciscoaciaim_controller.go`
- Modify: `pkg/ciscoaciaim/pvc.go`
- Modify: `pkg/ciscoaciaim/volumes.go`
- Test: `internal/controller/ciscoaciaim_controller_test.go`
- Test: `pkg/ciscoaciaim/statefulset_test.go`

- [ ] **Step 1: Add failing controller and builder tests**

Add a fake-client controller test asserting `ensureLogPVC` succeeds without
creating a PVC when `LogPersistence` is nil. Add StatefulSet tests asserting
the `aim-logs` mount and volume are absent in nonpersistent mode and present in
persistent mode.

- [ ] **Step 2: Run the focused packages and verify RED**

Run:

```sh
go test ./internal/controller ./pkg/ciscoaciaim
```

Expect the nil-persistence paths to fail or panic before the implementation.

- [ ] **Step 3: Implement the minimal conditional behavior**

Return immediately from `ensureLogPVC` when persistence is nil. Dereference the
pointer only in `LogPVC`. Build the common volumes and mounts first, then append
`aim-logs` only when persistence is configured.

- [ ] **Step 4: Run the focused packages and verify GREEN**

Run the same focused test command and require exit status zero.

### Task 3: Render stdout and persistent logging modes

**Files:**
- Modify: `internal/controller/ciscoaciaim_controller.go`
- Modify: `templates/aim_supervisord.conf`
- Modify: `templates/kolla_config.json`
- Modify: `templates/init.sh`
- Test: `internal/controller/config_test.go`

- [ ] **Step 1: Add failing template tests**

Parse the real template files with `LogToDisk` false and true. Assert that
nonpersistent Supervisor output uses `/dev/stdout`, `/dev/fd/1`, and
`/dev/fd/2`, contains no AIM `--log-file`, and that Kolla contains no
`/var/log/aim` permission. Assert persistent output retains the current
per-ordinal filenames and Kolla permission. Assert nonpersistent initialization
selects ordinal zero and does not reference the persistent done marker.

- [ ] **Step 2: Run the controller tests and verify RED**

Run `go test ./internal/controller` and expect the nonpersistent assertions to
fail against the static templates.

- [ ] **Step 3: Implement conditional template rendering**

Add:

```go
type LogConfigData struct {
    LogToDisk bool
}
```

Render the Supervisor, Kolla, and init files through the existing template
executor using `LogToDisk: instance.Spec.LogPersistence != nil`. Preserve the
existing persistent branches byte-for-byte where practical.

- [ ] **Step 4: Run controller and builder tests and verify GREEN**

Run `go test ./internal/controller ./pkg/ciscoaciaim` and require exit status
zero.

### Task 4: Regenerate and synchronize API artifacts

**Files:**
- Modify generated: `api/v1alpha1/zz_generated.deepcopy.go`
- Modify generated: `config/crd/bases/api.cisco.com_ciscoaciaims.yaml`
- Modify generated: `bundle/manifests/api.cisco.com_ciscoaciaims.yaml`
- Modify generated: `config/aim_configs/api.cisco.com_ciscoaciaims.yaml`

- [ ] **Step 1: Run generators with Go 1.24.1**

Run `make generate manifests`, then synchronize the generated CRD into the
