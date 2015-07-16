package dockerclient

import (
	"io"
)

type Callback func(*Event, ...interface{})

type Client interface {
	Info() (*Info, error)
	ListContainers(all, size bool, filters string) ([]Container, error)
	InspectContainer(id string) (*ContainerInfo, error)
	CreateContainer(config *ContainerConfig, name string) (string, error)
	ContainerLogs(id string, options *LogOptions) (io.ReadCloser, error)
	Exec(config *ExecConfig) (string, error)
	StartContainer(id string, config *HostConfig) error
	StopContainer(id string, timeout int) error
	RestartContainer(id string, timeout int) error
	KillContainer(id, signal string) error
	StartMonitorEvents(cb Callback, args ...interface{})
	StopAllMonitorEvents()
	Version() (*Version, error)
	PullImage(name string, auth *AuthConfig) error
	RemoveContainer(id string, force bool) error
	ListImages(all bool, dangling bool) ([]*Image, error)
	ImageHistory(id string) ([]ImageHistory, error)
	InspectImage(id string) (*ImageInfo, error)
	RemoveImage(name string) error
	PauseContainer(name string) error
	UnpauseContainer(name string) error
}
