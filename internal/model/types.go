// Package model defines canonical data types with no external dependencies.
// All k8s-specific types are converted to these before reaching the UI.
package model

import "time"

// Pod represents a Kubernetes pod in normalized form.
type Pod struct {
	Namespace     string
	Name          string
	Ready         string // e.g. "2/3"
	Status        string
	Restarts      int32
	Age           time.Duration
	Node          string
	PendingReason string // last FailedScheduling event message, if any
}

// Node represents a Kubernetes node in normalized form.
type Node struct {
	Name    string
	Status  string
	Roles   string
	Age     time.Duration
	Version string
	// Allocatable resources — always populated from node spec.
	AllocCPUm  int64 // allocatable CPU in millicores
	AllocMemMi int64 // allocatable memory in MiB
	AllocPods  int   // max allocatable pods
	// Current usage — populated only when metrics-server is available.
	UsedCPUm  int64
	UsedMemMi int64
	MetricsOK bool
}

// Namespace represents a Kubernetes namespace in normalized form.
type Namespace struct {
	Name   string
	Status string
	Age    time.Duration
}

// Deployment represents a Kubernetes deployment in normalized form.
type Deployment struct {
	Namespace string
	Name      string
	Ready     string // e.g. "1/3"
	UpToDate  int32
	Available int32
	Age       time.Duration
}

// UpdateKind identifies which resource type an Update carries.
type UpdateKind int

const (
	UpdatePods UpdateKind = iota
	UpdateNodes
	UpdateNamespaces
	UpdateDeployments
)

// Update is sent from watcher goroutines to the main goroutine.
type Update struct {
	Kind        UpdateKind
	Pods        []Pod
	Nodes       []Node
	Namespaces  []Namespace
	Deployments []Deployment
	Err         error
}
