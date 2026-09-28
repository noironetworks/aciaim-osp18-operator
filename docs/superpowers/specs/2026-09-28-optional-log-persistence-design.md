# Optional AIM Log Persistence

## Problem

`CiscoAciAim.spec.logPersistence` is required and reconciliation always creates
one RWO PVC that every AIM replica mounts. Deployments cannot opt out of this
claim, so an RWO-only storage class constrains all replicas to one worker.

This first patch provides a supported nonpersistent mode. Per-replica persistent
claims and pod anti-affinity are intentionally reserved for a follow-up patch.

## API

`spec.logPersistence` becomes an optional pointer. Its meanings are:

- Absent: AIM writes to container stdout and stderr and has no log volume.
- Present: AIM retains the existing shared PVC and file logging behavior.

When `logPersistence` is present, `size` is required. `storageClassName` remains
optional. Regenerated CRD schemas must describe these fields structurally so
the API server rejects values with the wrong type before the informer decodes
them.

Existing custom resources remain compatible because their object-valued
`logPersistence` fields decode into the pointer without a manifest change.

## Reconciliation

The controller calls `ensureLogPVC` only when persistence is configured.
Removing `logPersistence` updates the pod template and configuration without
deleting the existing PVC. Keeping the claim protects existing log data and
avoids making a configuration update destructive.

The StatefulSet builder includes the `aim-logs` volume and mount only in
persistent mode. The follow-up patch will replace the shared claim with a
StatefulSet `volumeClaimTemplate`.

## Process logging

Supervisor configuration becomes conditional:

- Nonpersistent mode sends the Supervisor log to stdout, removes each AIM
  process's `--log-file` argument, and forwards child stdout and stderr to the
  container streams without rotation.
- Persistent mode retains the existing per-ordinal file names and rotation.

The Kolla configuration manages `/var/log/aim` permissions only in persistent
mode.

## Initialization

The current post-start script keeps its completion marker on the log PVC. With
no persistent volume, only StatefulSet ordinal `0` performs AIM initialization.
It performs the existing idempotent configuration commands whenever ordinal
`0` is replaced. Other replicas skip initialization. Persistent mode keeps the
existing shared completion marker behavior.

## Testing

Tests cover:

- CR admission without `logPersistence`.
- No PVC creation when persistence is absent.
- Conditional StatefulSet volume and mount construction.
- stdout/stderr Supervisor and Kolla rendering in nonpersistent mode.
- unchanged file logging in persistent mode.
