package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/89luca89/distrobox/pkg/containermanager"
)

type Docker struct {
	verbose bool
}

var _ containermanager.ContainerManager = &Docker{}

func NewDocker(verbose bool) *Docker {
	return &Docker{
		verbose: verbose,
	}
}

func (d *Docker) Name() string {
	return "docker"
}

// dockerContainer represents the JSON output from `docker ps --format json`.
type dockerContainer struct {
	ID     string `json:"ID"`
	Image  string `json:"Image"`
	Names  string `json:"Names"`
	Status string `json:"Status"`
	Labels string `json:"Labels"`
}

func (d *Docker) ListContainers(ctx context.Context) ([]containermanager.Container, error) {
	args := []string{"ps", "-a", "--no-trunc", "--format", "json"}
	out, err := d.run(ctx, args)
	if err != nil {
		return nil, err
	}
	return parseContainerList(out)
}

func (d *Docker) run(ctx context.Context, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		captured := strings.TrimSpace(stderr.String())
		if captured != "" {
			return "", fmt.Errorf("command execution failed: %s", captured)
		}
		return "", fmt.Errorf("command execution failed: %w", err)
	}
	return stdout.String(), nil
}

func parseContainerList(output string) ([]containermanager.Container, error) {
	var containers []containermanager.Container

	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}

		var dc dockerContainer
		if err := json.Unmarshal([]byte(line), &dc); err != nil {
			return nil, fmt.Errorf("failed to parse container JSON: %w", err)
		}

		id := dc.ID
		if len(id) > 12 {
			id = id[:12]
		}

		containers = append(containers, containermanager.Container{
			ID:     id,
			Image:  dc.Image,
			Name:   dc.Names,
			Status: dc.Status,
			Labels: parseLabels(dc.Labels),
		})
	}

	return containers, nil
}

func parseLabels(labels string) map[string]string {
	result := make(map[string]string)
	if labels == "" {
		return result
	}

	for label := range strings.SplitSeq(labels, ",") {
		key, value, found := strings.Cut(label, "=")
		if found {
			result[key] = value
		}
	}
	return result
}
