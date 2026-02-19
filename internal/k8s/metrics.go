package k8s

import (
	"context"
	"encoding/json"

	"k8s.io/client-go/kubernetes"
)

// nodeMetrics holds raw CPU (millicores) and memory (MiB) usage for one node.
type nodeMetrics struct {
	cpuM  int64
	memMi int64
}

// nodeMetricsResp is the minimal JSON shape of the metrics-server response.
type nodeMetricsResp struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Usage struct {
			CPU    string `json:"cpu"`
			Memory string `json:"memory"`
		} `json:"usage"`
	} `json:"items"`
}

// FetchNodeMetrics queries /apis/metrics.k8s.io/v1beta1/nodes and returns a
// map of node-name → usage. Returns nil (no error) when metrics-server is
// unavailable — callers should treat nil as "metrics not available".
func FetchNodeMetrics(ctx context.Context, cs *kubernetes.Clientset) (map[string]nodeMetrics, error) {
	data, err := cs.RESTClient().Get().
		AbsPath("/apis/metrics.k8s.io/v1beta1/nodes").
		DoRaw(ctx)
	if err != nil {
		return nil, nil // metrics-server not installed or unavailable
	}

	var resp nodeMetricsResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil
	}

	result := make(map[string]nodeMetrics, len(resp.Items))
	for _, item := range resp.Items {
		result[item.Metadata.Name] = nodeMetrics{
			cpuM:  parseMilliCPU(item.Usage.CPU),
			memMi: parseMemMiB(item.Usage.Memory),
		}
	}
	return result, nil
}

// parseMilliCPU converts a CPU quantity string to millicores.
// Handles: "123456789n" (nanocores), "234m" (millicores), "2" (whole cores).
func parseMilliCPU(s string) int64 {
	if len(s) == 0 {
		return 0
	}
	switch s[len(s)-1] {
	case 'n': // nanocores → millicores (divide by 1,000,000)
		return parseInt(s[:len(s)-1]) / 1_000_000
	case 'm': // already millicores
		return parseInt(s[:len(s)-1])
	}
	// no suffix = whole cores
	return parseInt(s) * 1000
}

// parseMemMiB converts a memory quantity string (e.g. "1234Ki", "2Gi") to MiB.
func parseMemMiB(s string) int64 {
	if len(s) < 2 {
		return 0
	}
	// Detect suffix
	switch {
	case len(s) >= 2 && s[len(s)-2:] == "Ki":
		n := parseInt(s[:len(s)-2])
		return n / 1024
	case len(s) >= 2 && s[len(s)-2:] == "Mi":
		return parseInt(s[:len(s)-2])
	case len(s) >= 2 && s[len(s)-2:] == "Gi":
		return parseInt(s[:len(s)-2]) * 1024
	case len(s) >= 2 && s[len(s)-2:] == "Ti":
		return parseInt(s[:len(s)-2]) * 1024 * 1024
	case s[len(s)-1] == 'k' || s[len(s)-1] == 'K':
		return parseInt(s[:len(s)-1]) / 1024
	case s[len(s)-1] == 'M':
		return parseInt(s[:len(s)-1])
	case s[len(s)-1] == 'G':
		return parseInt(s[:len(s)-1]) * 1024
	}
	// plain bytes
	return parseInt(s) / (1024 * 1024)
}

func parseInt(s string) int64 {
	n := int64(0)
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int64(c-'0')
	}
	return n
}
