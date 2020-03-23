package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/healthd/conf"
	"github.com/healthd/logger"
	"github.com/healthd/metric"
	"github.com/healthd/timer"
)

const (
	dockerPackage = "docker"
	// DockerTimerType must be unique across probes
	DockerTimerType = 2001
	// dockerTimerSubtype normal processing probes
	dockerTimerSubtype = 2002
	// DockerTimerRetry Retry logic enabled
	dockerTimerRetry = 2003
	// DockerTimerWait Wail docker daemon recovers
	dockerTimerWait = 2004
)

// restartService running docker daemon
func restartService(conf *conf.Conf) {
	cmd := exec.Command("pkill", "-SIGUSR1", "dockerd")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logger.Err(err.Error())
		panic(err)
	} else {
		logger.Info(out.String())
		metric.IncrementRestartCount()
		timer.Launch(
			timer.Name(conf.Env.Name),
			timer.Timeout(conf.Env.Interval),
			timer.Type(DockerTimerType),
			timer.SubType(dockerTimerWait),
			timer.User(conf),
		)
	}
}

// Run launch specified client
func Run() {
	for _, conf := range conf.Confs {
		if strings.EqualFold(conf.Env.Package, dockerPackage) {
			timer.Launch(
				timer.Name(conf.Env.Name),
				timer.Timeout(conf.Env.Interval),
				timer.Type(DockerTimerType),
				timer.SubType(dockerTimerSubtype),
				timer.User(conf),
			)
		}
	}
}

func retryOperation(tle *timer.TLE) {
	conf, _ := tle.User.(*conf.Conf)
	conf.RetryCounter++
	if conf.RetryCounter > conf.Env.Retries {
		conf.RetryCounter = 0
		logger.Warning(fmt.Sprintf(
			"%s Retries Exceeded Max: %d Curr: %d",
			conf.Env.Name,
			conf.Env.Retries,
			conf.RetryCounter))
		restartService(conf)
	} else {
		timer.Launch(
			timer.Name(conf.Env.Name),
			timer.Timeout(conf.Env.Interval),
			timer.Type(tle.Type),
			timer.SubType(dockerTimerRetry),
			timer.User(conf),
			timer.Key(tle.Key),
		)
	}
}

func armTimer(tle *timer.TLE) {
	if conf, ok := tle.User.(*conf.Conf); ok {
		timer.Launch(
			timer.Name(tle.Name),
			timer.Timeout(conf.Env.Interval),
			timer.Type(tle.Type),
			timer.SubType(tle.SubType),
			timer.User(conf),
			timer.Key(tle.Key),
		)
	}
}

// Probe running containers
func Probe(tle *timer.TLE) {
	if conf, ok := tle.User.(*conf.Conf); ok {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			logger.Err(err.Error())
			metric.SetDockerMetric(1)
			retryOperation(tle)
			return
		}
		_, err = cli.ContainerList(
			context.Background(),
			types.ContainerListOptions{},
		)
		if err != nil {
			logger.Err(err.Error())
			metric.SetDockerMetric(0)
			retryOperation(tle)
			return
		}
		metric.SetDockerMetric(1)
		conf.RetryCounter = 0
		armTimer(tle)
	}
}
