package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/89luca89/distrobox/pkg/commands"
	"github.com/89luca89/distrobox/pkg/config"
	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/internal/testutil"
)

// mockListContainerManager wraps MockContainerManager and returns a fixed container list.
type mockListContainerManager struct {
	*testutil.MockContainerManager
	containers []containermanager.Container
}

func (m *mockListContainerManager) ListContainers(_ context.Context) ([]containermanager.Container, error) {
	m.MockContainerManager.Spy.ListContainers = append(m.MockContainerManager.Spy.ListContainers, []any{})
	return m.containers, nil
}

func newTestListCommand(containers []containermanager.Container) *commands.ListCommand {
	mock := &mockListContainerManager{
		MockContainerManager: &testutil.MockContainerManager{},
		containers:           containers,
	}
	return commands.NewListCommand(&config.Values{}, mock)
}

func distroboxContainer(name string) containermanager.Container {
	return containermanager.Container{
		ID:     name + "-id",
		Name:   name,
		Image:  "ubuntu:latest",
		Status: "Up",
		Labels: map[string]string{"manager": "distrobox"},
	}
}

func TestListCommand_NoOpts_ReturnsAllDistroboxes(t *testing.T) {
	containers := []containermanager.Container{
		distroboxContainer("box-a"),
		distroboxContainer("box-b"),
		{Name: "not-a-distrobox", Labels: map[string]string{}},
	}

	result, err := newTestListCommand(containers).Execute(context.Background(), nil)

	require.NoError(t, err)
	require.Len(t, result.Containers, 2)
	assert.Equal(t, "box-a", result.Containers[0].Name)
	assert.Equal(t, "box-b", result.Containers[1].Name)
}

func TestListCommand_MatchingName_ReturnsExactlyOne(t *testing.T) {
	containers := []containermanager.Container{
		distroboxContainer("box-a"),
		distroboxContainer("box-b"),
	}

	result, err := newTestListCommand(containers).Execute(context.Background(), &commands.ListOptions{
		ContainerName: "box-a",
	})

	require.NoError(t, err)
	require.Len(t, result.Containers, 1)
	assert.Equal(t, "box-a", result.Containers[0].Name)
}

func TestListCommand_NonMatchingName_ReturnsErrListContainerNotFound(t *testing.T) {
	containers := []containermanager.Container{
		distroboxContainer("box-a"),
	}

	_, err := newTestListCommand(containers).Execute(context.Background(), &commands.ListOptions{
		ContainerName: "does-not-exist",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, commands.ErrListContainerNotFound))
}
