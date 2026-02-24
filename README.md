# kubecurses

**Node-first Kubernetes terminal viewer focused on spotting scheduling issues and failing workloads instantly.**

Not a kubectl replacement — a fast, opinionated dashboard for when you need to know *which node a pod landed on, why it's failing, and how long it's been that way* without context-switching to a browser.

```
1:Heatmap  2:Overview  3:Xray  4:Deployments  5:Namespaces    ctx:prod-33 | 55 nodes | 962 pods

╱──────────────────────╲  ╱──────────────────────╲  ╱──────────────────────╲
│ ●● ip-172-16-220-138… │  │ ●● ip-172-16-222-174… │  │ ●● ip-172-16-228-107… │
│ ⬢ ⬢ ⬢ ⬢ ⬢ ⬢ ⬢        │  │ ⬢ ⬢ ⬢ ⬢ ⬢ ⬢ ⬢        │  │ ⬢ ⬢ ⬢ ⬢ ⬢ ⬢ ⬢        │
│  ⬢ ⬢ ⬢ ⬢ ⬢ ⬢         │  │  ⬢ ⬢ ⬢ ⬢ ⬢ ⬢         │  │  ⬢ ⬢ ⬢ ⬢ ⬢ ⬢         │
│ ⬢ ⬢ 🔵🔵              │  │ ⬢ ⬢ ⬢                 │  │ 🔴🔴 ⬢ ⬢ ⬢ ⬢ ⬢       │
╲──────────────────────╱  ╲──────────────────────╱  ╲──────────────────────╱
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

# Enable metrics-server integration (off by default)
kubecurses --enable-metrics

# Use informer-based live updates (default) or fall back to polling
kubecurses --watch=false --interval 5s

# Limit pods shown per node
kubecurses --max-pods 50

# Plain ASCII mode (no Unicode icons)
kubecurses --no-icons

# Version info
kubecurses --version
```

## Keyboard shortcuts

### Global

| Key | Action |
|-----|--------|
| `1` | Heatmap — per-node honeycomb grid (default) |
| `2` | Overview — nodes with pods |
| `3` | Xray — namespace→pod→container tree |
| `4` | Deployments |
| `5` | Namespaces |
| `Tab` / `Shift+Tab` | Next / previous tab |
| `c` | Switch cluster / context in-app |
| `/` | Search — filter rows live |
| `Esc` | Clear search / close overlay |
| `r` | Manual refresh |
| `?` | Help overlay |
| `q` / `Ctrl+C` | Quit |

### Heatmap tab

| Key | Action |
|-----|--------|
| `h` / `l` | Move left / right between nodes |
| `j` / `k` | Move down / up between node rows |
| `Enter` | Open node detail — pod table for that node |
| `Esc` | Back to heatmap from node detail |

### Node detail (inside heatmap)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate pods |
| `l` | Stream logs for selected pod |
| `e` | Open interactive shell in selected pod |
| `Esc` | Back to heatmap |

### Overview / Xray / other tabs

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `l` | Stream logs for selected pod / container (Xray) |
| `e` | Open interactive shell in selected pod / container (Xray) |

### Inside the logs / exec view

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll one line |
| `PgDn` / `PgUp` | Scroll one page |
| `s` | Toggle autoscroll (follow tail) |
| `Esc` | Close overlay |

## Views

### Heatmap (default — key `1`)

Each node gets a box with diagonal corners. Inside, every pod is a coloured `⬢` hexagon arranged in a staggered honeycomb grid:

| Colour | Status |
|--------|--------|
| 🟢 Green | Running |
| 🟡 Amber | Pending |
| 🔴 Red | Failed / CrashLoopBackOff / OOMKilled / ImagePullBackOff |
| ⚫ Gray | Terminating |
| 🔵 Blue | Unknown |

Pods are sorted within each node: failures first, then pending, running, terminating — so problems surface at the top-left of every cell.

Press `Enter` on any node to open the **node detail view** — a full pod table showing name, namespace, restart count, and age.

### Overview (key `2`)

Node section headers sorted by health — **NotReady nodes surface first**, then by pod count descending.

**Node rows** show cpu/mem usage and pod capacity when metrics-server is available (`cpu:62% mem:71% 26/58 pods`), colour-coded green → amber → red.

**Pod rows** show a status icon, name, namespace, status, ready count, restart count, and age. Restart counts are highlighted amber (≥3) or red (≥10).

**Pending pod explainer** — Pending pods show a sub-row with the scheduler's reason:
```
  ↻ redis   default   Pending   0/1   0   3h
    → 0/3 nodes available: Insufficient memory on 2 nodes
```

**Scheduling imbalance banner** — shown when any node carries 2× the average pod count.

### Xray (key `3`)

Namespace → pod → container tree. Shows every container inside each pod with its ready state, restart count, and status. Press `l` on any pod or container row to stream its logs live.

### Deployments / Namespaces

Standard tabular views with ready/available counts and ages.

## Logs

Navigate to the **Xray** tab (`3`), select a pod or container row, and press `l`. A bordered overlay streams logs live inside the dashboard.

- Lines longer than the box width are hard-wrapped.
- Autoscroll follows the tail by default. Press `s` to freeze and scroll freely; reaching the bottom re-engages autoscroll.
- Mouse wheel scrolls the log content.
- Press `Esc` to close.

## Exec

Navigate to the **Xray** tab (`3`) or open the **node detail view** (press `Enter` on a node in the Heatmap), select a pod or container row, and press `e`.

The TUI suspends and hands the terminal directly to `/bin/sh` inside the container — exactly like `kubectl exec -it`. The full terminal is available: colours, cursor movement, interactive editors (vim, nano), REPL tools, and any program that requires a real TTY all work as expected.

- **Terminal resize** (`SIGWINCH`) is forwarded to the remote PTY in real time, so the shell correctly tracks window resizes.
- Type `exit` or press `Ctrl+D` to end the session. The TUI resumes and redraws immediately.
- Transport and RBAC errors are surfaced in the status bar after the session ends.

## Cluster switching

Press `c` from any tab to open the in-app context picker. Select a context and press `Enter` — kubecurses reconnects without restarting and without touching your shell's current context.

## Search

Press `/` to open the search bar. Start typing — rows are filtered live by pod name, namespace, or status. Press `Enter` to keep the filter while navigating, or `Esc` to clear it.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `~/.kube/config` | Path to kubeconfig file |
| `--context` | current context | Kubernetes context to use |
| `--namespace` | all namespaces | Namespace filter |
| `--watch` | `true` | Use informer-based live updates (false = polling) |
| `--interval` | `10s` | Polling interval (polling mode only) |
| `--metrics-interval` | `30s` | How often to refresh node metrics |
| `--request-timeout` | `30s` | Kubernetes API request timeout |
| `--kube-api-qps` | `20` | Client-go QPS limit |
| `--kube-api-burst` | `40` | Client-go burst limit |
| `--enable-metrics` | `false` | Enable metrics-server calls |
| `--max-pods` | `0` (unlimited) | Cap pods shown per refresh |
| `--no-icons` | `false` | Plain ASCII fallbacks instead of Unicode icons |
| `--log-lines` | `200` | Log tail line count |
| `--version` | — | Print version and exit |

## Stack

- **Go 1.22**
- **[tcell/v2](https://github.com/gdamore/tcell)** — terminal rendering
- **[client-go](https://github.com/kubernetes/client-go)** — Kubernetes API, SharedInformerFactory
- No metrics-server dependency — gracefully degrades when absent

## Roadmap

- [x] Exec interactive shell (`e` — full PTY, raw key passthrough, SIGWINCH resize)
- [ ] Workload grouping — pods grouped by Deployment/StatefulSet/DaemonSet
- [ ] Persisted settings (`~/.config/kubecurses/config.yaml`)
- [ ] Brew tap

## License

MIT
