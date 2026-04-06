# Helm Kubeconfig Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure all Helm operations invoked by repo Make targets use `.cache/kubeconfig`, so installs and upgrades always target the same kind cluster as `kubectl` and `kind`.

**Architecture:** Reuse the existing `KUBE_ENV` in `Makefile` and prepend it to every Helm-based target. Then update docs to describe that repo commands already pin kubeconfig, and re-run the `dequeue` flow to confirm the live KEDA resources in `kind-keda-kind` now match the rendered develop manifests.

**Tech Stack:** Make, Helm, kubectl, kind, KEDA

---

### Task 1: Align Helm Make Targets With Repo Kubeconfig

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Write the failing context check**

Run:

```bash
kubectl config current-context
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl config current-context
```

Expected: The two contexts differ, showing why Helm must not rely on the shell default context.

- [ ] **Step 2: Update Helm targets to include `KUBE_ENV`**

Update the Helm commands in `Makefile` from:

```make
env $(HELM_ENV) helm ...
```

to:

```make
env $(KUBE_ENV) $(HELM_ENV) helm ...
```

Apply that change to all of these targets:

```make
helm-deps
install-elasticmq
install-postgresql
install-keda
install-keda-prod
install-enqueue
install-dequeue
```

- [ ] **Step 3: Verify the Makefile wiring**

Run:

```bash
rg -n "env \\$\\(KUBE_ENV\\) \\$\\(HELM_ENV\\) helm" Makefile
```

Expected: Every Helm target listed above now uses both `KUBE_ENV` and `HELM_ENV`.

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "fix: pin helm commands to repo kubeconfig"
```

### Task 2: Document The Kubeconfig Pinning

**Files:**
- Modify: `README.md`
- Modify: `TODO.md`

- [ ] **Step 1: Write the failing docs check**

Run:

```bash
rg -n "\\.cache/kubeconfig|Make target|secretTargetRef|export KUBECONFIG" README.md TODO.md
```

Expected: Docs still rely on manual `export KUBECONFIG=...` framing and do not clearly say that Makefile pins Helm to `.cache/kubeconfig`.

- [ ] **Step 2: Update README**

Add wording equivalent to:

```md
repo の `make` target は `KUBECONFIG=$(CURDIR)/.cache/kubeconfig` を使う前提で Helm / kubectl / kind を実行します。default context が別クラスタを向いていても、repo の導線は `kind-keda-kind` を対象にします。
```

Place it near the kind deployment instructions so the execution model is obvious before install steps.

- [ ] **Step 3: Update TODO**

Adjust `TODO.md` so:

- the `export KUBECONFIG=...` note is treated as an explicit shell convenience, not a requirement for Makefile correctness
- the verification list for the KEDA/dequeue flow adds a live resource check for `secretTargetRef`

Add wording equivalent to:

```md
- [ ] `kubectl get triggerauthentication dequeue -o yaml`
  - [ ] `secretTargetRef` が live resource に入っている
```

- [ ] **Step 4: Re-run the docs check**

Run:

```bash
rg -n "\\.cache/kubeconfig|secretTargetRef|export KUBECONFIG" README.md TODO.md
```

Expected: Matches now describe both the repo-pinned kubeconfig behavior and the live-resource check.

- [ ] **Step 5: Commit**

```bash
git add README.md TODO.md
git commit -m "docs: clarify helm kubeconfig behavior"
```

### Task 3: Verify Live Resources Match The Intended Cluster

**Files:**
- Modify: none

- [ ] **Step 1: Reinstall onto the pinned cluster**

Run:

```bash
make install-dequeue
```

Expected: Helm upgrade succeeds using the repo kubeconfig.

- [ ] **Step 2: Verify Helm and kubectl now agree on the same cluster**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig helm list -A
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get triggerauthentication dequeue -o yaml
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get scaledjob dequeue -o yaml
```

Expected:
- `helm list -A` shows the updated release on `kind-keda-kind`
- `TriggerAuthentication` contains `secretTargetRef`, not `podIdentity`
- `ScaledJob` no longer contains `identityOwner: operator` in the trigger metadata

- [ ] **Step 3: Verify KEDA starts creating jobs**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get jobs --watch
```

Expected: A `dequeue-*` Job appears after the scaler polls the queue. Stop the watch after confirming at least one job.

- [ ] **Step 4: Verify PostgreSQL persistence**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl port-forward svc/postgresql 5432:5432
psql 'postgres://app:app@127.0.0.1:5432/app?sslmode=disable' -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'
```

Expected: Query returns at least one row produced by `dequeue`.

- [ ] **Step 5: Commit**

```bash
git commit --allow-empty -m "test: verify helm kubeconfig alignment"
```
