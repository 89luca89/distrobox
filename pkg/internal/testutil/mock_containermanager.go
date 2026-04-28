package testutil

import (
	"context"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

// ContainerManagerSpy records all calls made to each method.
// Each field is a slice of call argument lists.
type ContainerManagerSpy struct {
	Name             [][]any
	CloneAsRoot      [][]any
	Enter            [][]any
	ListContainers   [][]any
	Create           [][]any
	Remove           [][]any
	Exists           [][]any
	Stop             [][]any
	InspectContainer [][]any
	Commit           [][]any
	ImageExists      [][]any
	PullImage        [][]any
}

// MockContainerManager is a no-op container manager for testing.
// All method calls are recorded in the Spy.
//
// CloneAsRoot returns a distinct MockContainerManager so tests can
// distinguish calls made on the rootless vs root variant. The clone is
// cached on RootClone after first creation.
type MockContainerManager struct {
	Spy       ContainerManagerSpy
	Root      bool
	RootClone *MockContainerManager
}

func (m *MockContainerManager) Name() string {
	m.Spy.Name = append(m.Spy.Name, []any{})
	return "mock"
}

func (m *MockContainerManager) CloneAsRoot() containermanager.ContainerManager {
	m.Spy.CloneAsRoot = append(m.Spy.CloneAsRoot, []any{})
	if m.RootClone == nil {
		m.RootClone = &MockContainerManager{Root: true}
	}
	return m.RootClone
}

func (m *MockContainerManager) Enter(_ context.Context, options containermanager.EnterOptions, progress *ui.Progress, printer *ui.Printer) error {
	m.Spy.Enter = append(m.Spy.Enter, []any{options, progress, printer})
	return nil
}

func (m *MockContainerManager) ListContainers(_ context.Context) ([]containermanager.Container, error) {
	m.Spy.ListContainers = append(m.Spy.ListContainers, []any{})
	return []containermanager.Container{}, nil
}

func (m *MockContainerManager) Create(_ context.Context, opts containermanager.CreateOptions) error {
	m.Spy.Create = append(m.Spy.Create, []any{opts})
	return nil
}

func (m *MockContainerManager) Remove(_ context.Context, containerName string, opts containermanager.RmOptions) error {
	m.Spy.Remove = append(m.Spy.Remove, []any{containerName, opts})
	return nil
}

func (m *MockContainerManager) Exists(_ context.Context, containerName string) bool {
	m.Spy.Exists = append(m.Spy.Exists, []any{containerName})
	return false
}

func (m *MockContainerManager) Stop(_ context.Context, containerNames []string) error {
	m.Spy.Stop = append(m.Spy.Stop, []any{containerNames})
	return nil
}

func (m *MockContainerManager) InspectContainer(_ context.Context, containerName string) (*containermanager.InspectResult, error) {
	m.Spy.InspectContainer = append(m.Spy.InspectContainer, []any{containerName})
	return &containermanager.InspectResult{}, nil
}

func (m *MockContainerManager) Commit(_ context.Context, containerID string, tag string) error {
	m.Spy.Commit = append(m.Spy.Commit, []any{containerID, tag})
	return nil
}

func (m *MockContainerManager) ImageExists(_ context.Context, imageName string) bool {
	m.Spy.ImageExists = append(m.Spy.ImageExists, []any{imageName})
	return false
}

func (m *MockContainerManager) PullImage(_ context.Context, imageName string, platform string) error {
	m.Spy.PullImage = append(m.Spy.PullImage, []any{imageName, platform})
	return nil
}
