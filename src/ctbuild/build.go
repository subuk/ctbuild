package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
)

type mainCmd struct {
	App *kingpin.Application

	Specfile string

	BuildArgs  []string
	ConfigPath string
	SourceDir  string
	ResultDir  string
	CacheDir   string
	NoCleanup  bool

	docker *dockerclient.Client
}

func AddMainCmd(app *kingpin.Application) {
	cmd := &mainCmd{App: app}

	app.Arg("args", "Build arguments").Required().StringsVar(&cmd.BuildArgs)
	app.Flag("config", "Build env configuration").Short('c').StringVar(&cmd.ConfigPath)
	app.Flag("no-cleanup", "Do not remove container after build").Short('n').BoolVar(&cmd.NoCleanup)
	app.Flag("source-dir", "Source directory").Default(".").Short('s').StringVar(&cmd.SourceDir)
	app.Flag("result-dir", "Results output directory").Short('d').Default("./result").StringVar(&cmd.ResultDir)
	app.Flag("cache-dir", "Host directory for cache").Short('e').Default("/tmp/ctbuild-cache").StringVar(&cmd.CacheDir)

	app.Action(cmd.Run)
	app.PreAction(func(*kingpin.ParseContext) error {
		docker, err := dockerclient.NewEnvClient()
		if err != nil {
			return Error{err, "failed to create docker client"}
		}
		cmd.docker = docker
		return nil
	})
}

func (cmd *mainCmd) removeContainer(containerId string) {
	if cmd.NoCleanup {
		cmd.docker.ContainerKill(context.Background(), containerId, "KILL")
		return
	}
	opts := types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}
	if err := cmd.docker.ContainerRemove(context.Background(), containerId, opts); err != nil {
		log.Printf("failed to remove container: %s", err)
	}
}

func (cmd *mainCmd) Run(*kingpin.ParseContext) error {
	sourceDir, err := filepath.Abs(cmd.SourceDir)
	if err != nil {
		return Error{err, "failed to resolve sources directory"}
	}
	resultDir, err := filepath.Abs(cmd.ResultDir)
	if err != nil {
		return Error{err, "failed to resolve result directory"}
	}
	cacheDir, err := filepath.Abs(cmd.CacheDir)
	if err != nil {
		return Error{err, "failed to resolve cache directory"}
	}

	if err := os.MkdirAll(resultDir, 0755); err != nil {
		return Error{err, "failed to create results directory"}
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return Error{err, "failed to create cache directory"}
	}

	ctx := context.Background()

	config, err := ParseBuildEnvConfig(cmd.ConfigPath)
	if err != nil {
		return Error{err, "failed to parse build env config file"}
	}
	envConfig := config

	createResponse, err := cmd.docker.ContainerCreate(ctx, &container.Config{
		Image:      envConfig.BaseImage,
		Tty:        true,
		Entrypoint: []string{envConfig.BuildCmd},
		Cmd:        cmd.BuildArgs,
	}, &container.HostConfig{
		Binds: []string{
			sourceDir + ":/source",
			resultDir + ":/result",
			cacheDir + ":/cache",
		},
	}, nil, "")
	if err != nil {
		return Error{err, "failed to create container"}
	}
	containerId := createResponse.ID

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func(id string) {
		for sig := range c {
			fmt.Println("\nsignal received:", sig)
			cmd.removeContainer(id)
		}
	}(containerId)

	defer signal.Stop(c)
	defer cmd.removeContainer(containerId)

	filesArchive, err := envConfig.FilesArchive()
	if err != nil {
		return Error{err, "failed to create files archive"}
	}

	if err := cmd.docker.CopyToContainer(ctx, containerId, "/", filesArchive, types.CopyToContainerOptions{}); err != nil {
		return Error{err, "failed to write files to container"}
	}

	if err := cmd.docker.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		return Error{err, "failed to start container"}
	}

	ctLogs, err := cmd.docker.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{
		Follow:     true,
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return Error{err, "failed to fetch container logs"}
	}
	if _, err := io.Copy(os.Stdout, ctLogs); err != nil && err != io.EOF {
		return Error{err, "failed to read container logs"}
	}
	ctInfo, err := cmd.docker.ContainerInspect(ctx, containerId)
	if err != nil {
		return Error{err, "failed to inspect container"}
	}
	if ctInfo.State.ExitCode != 0 {
		return Error{err, fmt.Sprintf("container exited with code %d", ctInfo.State.ExitCode)}
	}
	return nil

}
