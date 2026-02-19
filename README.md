# kubecurses

A terminal dashboard for Kubernetes. See what's running on each node — without leaving your terminal.

```
● worker-1          Ready      v1.29.0
    nginx-web        default    Running    2/2    0    3d
    coredns          kube-system Running   2/2    0    15d
    metrics-server   kube-system Running   1/1    0    15d

● worker-2          Ready      v1.29.0
    api-server       default    Running    3/3    0    1d
    redis            default    Pending    0/1    5    3h
    broken-job       default    Failed     0/1    12   45m
```

## Install

Download the latest binary for your platform from [Releases](https://github.com/1homsi/kubecurses/releases):

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
| `q` / `Ctrl+C` | Quit |

## Views

### Overview (default)
Nodes are shown as section headers. Each node lists the pods scheduled on it, coloured by status. Restart counts are highlighted amber (≥3) or red (≥10) so problem pods stand out immediately.

### Pods
Flat list of all pods across all nodes, sortable by navigation. Respects the `--namespace` flag and the active search filter.

### Deployments / Namespaces
Standard tabular views with ready/available counts and ages.

## Search

Press `/` to open the search bar. Start typing — rows are filtered live across all views by pod name, namespace, or status. Press `Enter` to keep the filter active while you navigate, or `Esc` to clear it.

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

## License

MIT
