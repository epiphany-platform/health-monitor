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
	}
	if len(out.String()) > 0 {
		logger.Info(out.String())
	}
	metric.IncrementRestartCount()
	timer.Launch(
		timer.Name(conf.Env.Name),
		timer.Timeout(conf.Env.Interval),
		timer.Type(DockerTimerType),
		timer.SubType(dockerTimerWait),
		timer.User(conf),
	)
}

// Run launch specified client
func Run() {
	conf.Run(
		dockerPackage,
		DockerTimerType,
		dockerTimerSubtype,
	)
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

// probeDocker running containers
func probeDocker(tle *timer.TLE) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	_, err = cli.ContainerList(context.Background(), types.ContainerListOptions{})
	return err

}

// resetCounter retry counter 
func resetCounter(conf *conf.Conf) {
	conf.RetryCounter = 0
}

// incCounter increment retry counter 
func incCounter(conf *conf.Conf) {
	conf.RetryCounter++
}

// Probe specified docker HTTP endpoint
func Probe(tle *timer.TLE) {
	conf, _ := tle.User.(*conf.Conf)
	if err := probeDocker(tle); err == nil {
		metric.SetDockerMetric(1)
		resetCounter(conf)
		armTimer(tle)
	} else {
		metric.SetDockerMetric(0)
		if !strings.Contains(
			strings.ToLower(err.Error()), "cannot connect to the docker daemon") {
			logger.Warning(err.Error())
			restartService(conf)
		} else {
			armTimer(tle)
		}
	}
}
