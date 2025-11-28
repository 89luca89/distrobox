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
	ListContainers(ctx context.Context) ([]Container, error)
}
