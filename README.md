# kubecurses

**Node-first Kubernetes terminal viewer focused on spotting scheduling issues and failing workloads instantly.**

Not a kubectl replacement — a fast, opinionated dashboard for when you need to know *which node a pod landed on, why it's failing, and how long it's been that way* without context-switching to a browser.

```
1:Overview  2:Pods  3:Deployments  4:Namespaces    ctx:prod-33 | 3 nodes | 47 pods

⚠  Scheduling imbalance: ip-172-16-217-131… has 74% of pods (26/35 total)

  NAME                              NAMESPACE              STATUS     READY  REST  AGE
● ip-172-16-217-131.eu-west-1…     cpu:62% mem:71% 26/…   Ready               8d
  ✔ nginx-web                       default                Running    2/2    0     3d
  ✔ coredns                         kube-system            Running    2/2    0     15d
  ✔ metrics-server                  kube-system            Running    1/1    0     15d

● ip-172-16-219-253.eu-west-1…     cpu:8%  mem:44% 8/58   NotReady            5h
  ↻ redis                           default                Pending    0/1    0     3h
    → 0/3 nodes available: Insufficient memory on 2 nodes, 1 node has unmet affinity
  ✖ broken-job                      default                CrashLoop  0/1    12    45m
```

## Install

Download the latest binary from [Releases](https://github.com/1homsi/kubecurses/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/1homsi/kubecurses/releases/latest/download/kubecurses_darwin_arm64.tar.gz | tar xz
sudo mv kubecurses /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/1homsi/kubecurses/releases/latest/download/kubecurses_darwin_amd64.tar.gz | tar xz
sudo mv kubecurses /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/1homsi/kubecurses/releases/latest/download/kubecurses_linux_amd64.tar.gz | tar xz
sudo mv kubecurses /usr/local/bin/
```

Or build from source (requires Go 1.22+):

```bash
go install github.com/1homsi/kubecurses@latest
```

## Usage

```bash
# Uses your current kubeconfig context, all namespaces
kubecurses

# Filter to a specific namespace
kubecurses --namespace default

# Use a specific context
kubecurses --context my-cluster

# Combine
kubecurses --context production --namespace kube-system

# Faster polling
kubecurses --interval 5s

# Version info
kubecurses --version
```

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `1` | Overview — nodes with pods |
| `2` | Pods — flat list |
| `3` | Deployments |
| `4` | Namespaces |
| `Tab` / `Shift+Tab` | Next / previous tab |
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `PgDn` / `PgUp` | Page scroll |
| `/` | Search — filter rows live |
| `Esc` | Clear search |
| `r` | Manual refresh |
| `?` | Help overlay |
| `q` / `Ctrl+C` | Quit |

## Views

### Overview (default)

Node section headers are sorted by health — **NotReady nodes surface first**, then by pod count descending.

**Node rows** show cpu/mem usage and pod capacity when metrics-server is available (`cpu:62% mem:71% 26/58 pods`), colour-coded green → amber → red. Falls back to the k8s version string when metrics-server is absent.

**Pod rows** show a status icon, name, namespace, status, ready count, restart count, and age. Restart counts are highlighted amber (≥3) or red (≥10). Detected statuses beyond plain `kubectl` phase: `CrashLoopBackOff`, `Terminating`, `OOMKilled`, `ImagePullBackOff`, and more.

**Pending pod explainer** — Pending pods show a sub-row with the scheduler's reason:
```
  ↻ redis   default   Pending   0/1   0   3h
    → 0/3 nodes available: Insufficient memory on 2 nodes
```

**Scheduling imbalance banner** — shown when any node carries 2× the average pod count.

**Column widths** scale with your terminal — names expand on wider displays.

### Pods
Flat list of all pods across all nodes. Respects the `--namespace` flag and the active search filter.

### Deployments / Namespaces
Standard tabular views with ready/available counts and ages.

## Search

Press `/` to open the search bar. Start typing — rows are filtered live by pod name, namespace, or status. Press `Enter` to keep the filter while you navigate, or `Esc` to clear it.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `~/.kube/config` | Path to kubeconfig file |
| `--context` | current context | Kubernetes context to use |
| `--namespace` | all namespaces | Namespace filter |
| `--interval` | `10s` | Polling interval |
| `--version` | — | Print version and exit |

## Stack

- **Go 1.22**
- **[tcell/v2](https://github.com/gdamore/tcell)** — terminal rendering
- **[client-go](https://github.com/kubernetes/client-go)** — Kubernetes API
- No metrics-server dependency — gracefully degrades when absent

## Roadmap

- [x] Node CPU/mem/pod capacity via metrics-server
- [x] Pending pod scheduler explainer
- [x] Scheduling imbalance detection
- [ ] Informer-based live updates (no polling)
- [ ] Logs view (`l` to stream pod logs)
- [ ] Exec shell (`e` to open a shell in a container)
- [ ] Workload grouping — pods grouped by Deployment/StatefulSet/DaemonSet
- [ ] Persisted settings (`~/.config/kubecurses/config.yaml`)
- [ ] Brew tap

## License

MIT
