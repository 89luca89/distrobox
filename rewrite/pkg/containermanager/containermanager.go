package containermanager

import (
	"context"
	"strings"
)

type Container struct {
	ID     string
	Image  string
	Name   string
	Status string
	Labels map[string]string
}

type CreateOptions struct {
	ContainerName           string
	ContainerImage          string
	ContainerClone          string
	ContainerUserCustomHome string
	ContainerHostname       string
	ContainerPlatform       string
	ContainerUserHome       string
	Nopasswd                bool
	UnshareDevsys           bool
	UnshareGroups           bool
	UnshareIPC              bool
	UnshareNetNS            bool
	UnshareProcess          bool
	AdditionalFlags         []string
	AdditionalVolumes       []string
	AdditionalPackages      []string
	ContainerPreInitHook    string
	ContainerInitHook       string
	Init                    bool
	Nvidia                  bool
	DryRun                  bool
}

type EnterOptions struct {
	ContainerName   string
	AdditionalFlags string
	NoTTY           bool
	NoWorkDir       bool
	CleanPath       bool
	Verbose         bool
}

func (c Container) IsDistrobox() bool {
	return c.Labels["manager"] == "distrobox"
}

func (c Container) IsRunning() bool {
	s := strings.ToLower(c.Status)
	return strings.Contains(s, "up") || strings.Contains(s, "running")
}

type ContanerManagerType string

type ContainerManager interface {
	Name() string
	Enter(ctx context.Context, options EnterOptions) error
	ListContainers(ctx context.Context) ([]Container, error)
	Create(ctx context.Context, opts CreateOptions) error
}
