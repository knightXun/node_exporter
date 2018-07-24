package docker

import (
	dockerType "github.com/docker/engine-api/types"
	dClient "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types/filters"
	"golang.org/x/net/context"
	"sync"
	"fmt"
	"time"
)

var defaultDockerSock = "unix:///var/run/docker.sock"

var dockerClient *dClient.Client
var once = sync.Once{}
var dclientErr error

type ContainerStatus struct {
	Status 			string
	State 			string
	RestartCount 	int
	ImageName		string
}

func NewDockerClient() (*dClient.Client, error) {
	once.Do(func() {
		dockerClient, dclientErr = dClient.NewClient(defaultDockerSock, "" , nil, nil)
	})
	return dockerClient, dclientErr
}

func GetContainerStatus(imageName string) (status ContainerStatus, err error){
	opts := dockerType.ContainerListOptions{
		Filter: filters.NewArgs(),
		All: true,
	}
	opts.Filter.Add("ancestor", imageName)

	ctx, _ := context.WithTimeout(context.Background(), 1 * time.Second)
	containers, err := dockerClient.ContainerList(ctx , opts)
	if err != nil {
		return ContainerStatus{}, err
	}
	if len(containers) != 1 {
		return ContainerStatus{}, fmt.Errorf("more then one containers: %v", imageName)
	}

	fmt.Println(containers[0].Status, containers[0].State)
	containerID := containers[0].ID

	containerStats, err := dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return ContainerStatus{}, err
	}
	res := ContainerStatus{
		RestartCount: containerStats.RestartCount,
		Status:  containerStats.State.Status,
		ImageName: containerStats.Image,
	}
	return res, nil
}