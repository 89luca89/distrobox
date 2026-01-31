package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/89luca89/distrobox/pkg/containermanager"
	"github.com/89luca89/distrobox/pkg/ui"
)

const (
	// DefaultCreateContainerImage Fedora toolbox is a sensitive default
	// FIXME: the default image should be determined based on the configuration
	DefaultCreateContainerImage = "registry.fedoraproject.org/fedora-toolbox:latest"
	DefaultCreateContainerName  = "my-distrobox"
	maxHostnameLength           = 64
)

var ErrHostnameTooLong = fmt.Errorf("hostname too long, must be less than %d characters", maxHostnameLength)

type ContainerAlreadyExistsError struct {
	ContainerName string
}

func (e *ContainerAlreadyExistsError) Error() string {
	return fmt.Sprintf("container named '%s' already exists", e.ContainerName)
}

type CreateCommand struct {
	containerManager containermanager.ContainerManager
	generateEntryCmd *GenerateEntryCommand
	progress         *ui.Progress
}

type CreateOptions struct {
	// ContainerClone name of the distrobox container to use as base for a new container
	ContainerClone string
	// ContainerImage image to use for the container
	ContainerImage string
	// ContainerName name of the distrobox
	ContainerName string
	// ContainerHostname hostname to set inside the container
	ContainerHostname string
	// ContainerPlatform platform to use for the container (e.g., linux/amd64, linux/arm64)
	ContainerPlatform string
	Nopasswd          bool

	// UnshareNetNs if true, do not share host network namespace
	UnshareNetNs bool
	// UnshareDevsys if true, do not share host devices and sysfs dirs from host
	UnshareDevsys bool
	// UnshareGroups if true, do not forward user's additional groups into the container
	UnshareGroups bool
	// UnshareIpc if true, do not share host IPC namespace
	UnshareIpc bool
	// UnshareProcess if true, do not share host process namespace
	UnshareProcess bool

	AdditionalFlags      []string
	AdditionalVolumes    []string
	AdditionalPackages   []string
	ContainerPreInitHook string
	ContainerInitHook    string

	ContainerUserCustomHome string
	ContainerHomePrefix     string
	Init                    bool

	Nvidia bool
	DryRun bool

	GenerateEntry bool
	Rootful       bool
}

func NewCreateCommand(cm containermanager.ContainerManager, progress *ui.Progress) *CreateCommand {
	return &CreateCommand{
		containerManager: cm,
		generateEntryCmd: NewGenerateEntryCommand(NewListCommand(cm)),
		progress:         progress,
	}
}

func (c *CreateCommand) Execute(ctx context.Context, opts CreateOptions) error {
	// Determine right containerImage to use
	//
	// If no clone option and no container image, let's choose a default image to use.
	//
	// If no name is specified and we're using the default container_image, then let's
	// set a default name for the container, that is distinguishable from the default
	// toolbx one. This will avoid problems when using both toolbx and distrobox on
	// the same system.
	containerImage := opts.ContainerImage
	if opts.ContainerClone == "" && containerImage == "" {
		containerImage = DefaultCreateContainerImage
	}
	if opts.DryRun && opts.ContainerClone != "" {
		containerImage = opts.ContainerClone
	}

	// Determine right containerName to use
	//
	// If no name is specified and no image is specified, then let's
	// set a default name for the container, that is distinguishable from the default
	// toolbx one. This will avoid problems when using both toolbx and distrobox on
	// the same system.
	//
	// If no container_name is declared, we build our container name starting from the
	// container image specified.
	//
	// Examples:
	//	alpine -> alpine
	//	ubuntu:20.04 -> ubuntu-20.04
	//	registry.fedoraproject.org/fedora-toolbox:39 -> fedora-toolbox-39
	//	ghcr.io/void-linux/void-linux:latest-full-x86_64 -> void-linux-latest-full-x86_64
	containerName := opts.ContainerName
	if opts.ContainerImage == "" {
		containerName = DefaultCreateContainerName
	}
	if containerName == "" {
		base := path.Base(containerImage)
		base = strings.ReplaceAll(base, ":", "-")
		base = strings.ReplaceAll(base, ".", "-")
		containerName = base
	}

	// Determine right containerHostname to use
	//
	containerHostname := opts.ContainerHostname
	if containerHostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("unable to get hostname: %w", err)
		}
		containerHostname = hostname
		if opts.UnshareNetNs {
			containerHostname = fmt.Sprintf("%s.%s", containerHostname, hostname)
		}
	}
	if len(containerHostname) > maxHostnameLength {
		return ErrHostnameTooLong
	}

	// Determine right containerUserCustomHome to use
	//
	// We check if the user has a custom home prefix to use for the container home.
	// If we have a home prefix to use, and no custom home set, then we set
	// the custom home to be PREFIX/CONTAINER_NAME
	containerUserCustomHome := opts.ContainerUserCustomHome
	if opts.ContainerHomePrefix != "" && containerUserCustomHome == "" {
		containerUserCustomHome = filepath.Join(opts.ContainerHomePrefix, containerName)
	}

	if c.containerManager.Exists(ctx, containerName) {
		return &ContainerAlreadyExistsError{ContainerName: containerName}
	}

	if opts.ContainerClone != "" && !opts.DryRun {
		cloneImage, err := c.clone(ctx, opts.ContainerClone)
		if err != nil {
			return fmt.Errorf("failed to clone container %s: %w", opts.ContainerClone, err)
		}
		containerImage = cloneImage
	}

	// TODO: pull image if needed
	// https://github.com/89luca89/distrobox/blob/main/distrobox-create#L1016

	c.progress.Next("Creating '%s' using image %s", containerName, containerImage)

	err := c.containerManager.Create(
		ctx,
		containermanager.CreateOptions{
			ContainerName:           containerName,
			ContainerImage:          containerImage,
			ContainerClone:          opts.ContainerClone,
			ContainerUserCustomHome: containerUserCustomHome,
			ContainerHostname:       containerHostname,
			ContainerPlatform:       opts.ContainerPlatform,
			Nopasswd:                opts.Nopasswd,
			UnshareDevsys:           opts.UnshareDevsys,
			UnshareGroups:           opts.UnshareGroups,
			UnshareIPC:              opts.UnshareIpc,
			UnshareNetNS:            opts.UnshareNetNs,
			UnshareProcess:          opts.UnshareProcess,
			AdditionalFlags:         opts.AdditionalFlags,
			AdditionalVolumes:       opts.AdditionalVolumes,
			AdditionalPackages:      opts.AdditionalPackages,
			ContainerPreInitHook:    opts.ContainerPreInitHook,
			ContainerInitHook:       opts.ContainerInitHook,
			Init:                    opts.Init,
			Nvidia:                  opts.Nvidia,
			DryRun:                  opts.DryRun,
		},
	)

	if err != nil {
		c.progress.Fail()
		return fmt.Errorf("failed to create container: %w", err)
	}

	c.progress.Done()

	if opts.GenerateEntry && !opts.DryRun && !opts.Rootful {
		err := c.generateEntryCmd.Execute(
			ctx,
			&GenerateEntryOptions{
				ContainerName: containerName,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to generate entry for container %s: %w", containerName, err)
		}
	}

	return nil
}

func (c *CreateCommand) clone(ctx context.Context, containerName string) (string, error) {
	i, err := c.containerManager.InspectContainer(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container status: %w", err)
	}

	if i.ContainerStatus == "running" {
		return "", errors.New("cannot clone running container, name: " + containerName)
	}

	commitTag := fmt.Sprintf("%s:%s", strings.ToLower(containerName), time.Now().Format("2006-01-02"))

	err = c.containerManager.Commit(ctx, i.ContanerID, commitTag)
	if err != nil {
		return "", fmt.Errorf("Failed to commit container: %w", err)
	}

	return commit_tag, nil
}
