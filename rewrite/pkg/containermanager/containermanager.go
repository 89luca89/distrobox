package containermanager

import (
	"context"
	"strings"

	"github.com/89luca89/distrobox/pkg/ui"
)

type Container struct {
	ID     string
	Image  string
	Name   string
	Status string
	Labels map[string]string
}

type InspectResult struct {
	ContanerID      string
	ContainerStatus string
	ContainerHome   string
	ContainerPath   string
	UnshareGroups   bool
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
	CustomCommand   string
	DryRun          bool
	NoTTY           bool
	NoWorkDir       bool
	CleanPath       bool
	Verbose         bool
}

type RmOptions struct {
	Force         bool
	RemoveHome    bool
	ContainerHome string
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
	Enter(ctx context.Context, options EnterOptions, progress *ui.Progress, printer *ui.Printer) error
	ListContainers(ctx context.Context) ([]Container, error)
	Create(ctx context.Context, opts CreateOptions) error
	Remove(ctx context.Context, containerName string, opts RmOptions) error
	Exists(ctx context.Context, containerName string) bool
	ImageExists(ctx context.Context, imageName string) bool
	Stop(ctx context.Context, containerNames []string) error
	InspectContainer(ctx context.Context, containerName string) (*InspectResult, error)
	PullImage(ctx context.Context, imageName string, platform string) error
	Commit(ctx context.Context, containerID string, imageTag string) error
}
