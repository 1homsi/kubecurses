// Package model defines canonical data types with no external dependencies.
// All k8s-specific types are converted to these before reaching the UI.
package model

import "time"

type Container struct {
	Name         string
	Ready        bool
	Restarts     int32
	Status       string
	CPURequestM  int64
	CPULimitM    int64
	MemRequestMi int64
	MemLimitMi   int64
	Image        string
	Message      string
}

type Pod struct {
	Namespace     string
	Name          string
	Ready         string
	Status        string
	Restarts      int32
	Age           time.Duration
	Node          string
	PendingReason string
	Containers    []Container
}

type Node struct {
	Name       string
	Status     string
	Roles      string
	Age        time.Duration
	Version    string
	AllocCPUm  int64
	AllocMemMi int64
	AllocPods  int
	UsedCPUm   int64
	UsedMemMi  int64
	MetricsOK  bool
	Taints     []string
}

type Namespace struct {
	Name   string
	Status string
	Age    time.Duration
}

type Deployment struct {
	Namespace string
	Name      string
	Ready     string
	UpToDate  int32
	Available int32
	Age       time.Duration
}

type Event struct {
	Namespace string
	Kind      string
	Name      string
	Reason    string
	Message   string
	Count     int32
	Age       time.Duration
	Type      string
}

// UpdateKind identifies which resource type an Update carries.
type UpdateKind int

const (
	UpdatePods UpdateKind = iota
	UpdateNodes
	UpdateNamespaces
	UpdateDeployments
	UpdateEvents
)

// Update is sent from watcher goroutines to the main goroutine.
type Update struct {
	Kind               UpdateKind
	Pods               []Pod
	Nodes              []Node
	Namespaces         []Namespace
	Deployments        []Deployment
	Events             []Event
	Err                error
	PodsTruncated      bool
	TotalPodsBeforeCap int
}
