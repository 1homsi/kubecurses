# kubecurses

**Node-first Kubernetes terminal viewer focused on spotting scheduling issues and failing workloads instantly.**

Not a kubectl replacement — a fast, opinionated dashboard for when you need to know *which node a pod landed on, why it's failing, and how long it's been that way* without context-switching to a browser.

```
1:Overview  2:Pods  3:Deployments  4:Namespaces      ctx:prod-33 | 3 nodes | 47 pods

  NAME                            NAMESPACE         STATUS     READY  REST  AGE
● ip-172-16-217-131.eu-west-1…   v1.33.5-eks-11…   Ready               8d
  ✔ nginx-web                     default           Running    2/2    0     3d
  ✔ coredns                       kube-system       Running    2/2    0     15d
  ↻ redis                         default           Pending    0/1    5     3h
● ip-172-16-219-253.eu-west-1…   v1.33.5-eks-11…   NotReady            5h
  ✖ broken-job                    default           CrashLoop  0/1    12    45m
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
Nodes are section headers sorted by health — **NotReady nodes surface first**, then by pod count. Each node lists the pods scheduled on it. Pod rows show a status icon (✔ ↻ ✖ ⚠ ⊘), name, namespace, status, ready count, restart count, and age. Restart counts are highlighted amber (≥3) or red (≥10).

Detected statuses beyond plain `kubectl` phase: `CrashLoopBackOff`, `Terminating`, `OOMKilled`, `ImagePullBackOff`, and more.

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

## Roadmap

- [ ] Informer-based live updates (no polling)
- [ ] Node resource usage — CPU/mem requests vs capacity via metrics-server
- [ ] Logs view (`l` to stream pod logs)
- [ ] Exec shell (`e` to open a shell in a container)
- [ ] Workload grouping — pods grouped by Deployment/StatefulSet/DaemonSet
- [ ] Persisted settings (`~/.config/kubecurses/config.yaml`)
- [ ] Brew tap

## License

MIT
