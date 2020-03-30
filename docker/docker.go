package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/epiphany-platform/health-monitor/conf"
	"github.com/epiphany-platform/health-monitor/logger"
	"github.com/epiphany-platform/health-monitor/metric"
	"github.com/epiphany-platform/health-monitor/timer"
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
func bounceService(conf *conf.Conf) {
	if !conf.Env.ActionFatal {
		resetCounter(conf)
		armTimer(conf)
	} else {
		cmd := exec.Command("systemctl", "kill", "--kill-who=main", "docker")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			logger.Err(err.Error())
		}
		if len(out.String()) > 0 {
			logger.Info(out.String())
		} else {
			logger.Info(fmt.Sprintf(
				"Restarted Name: %s Service: %s Completed",
				conf.Env.Name,
				conf.Env.Package,
			))
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
}

// Run launch specified client
func Run() {
	conf.Run(
		dockerPackage,
		DockerTimerType,
		dockerTimerSubtype,
	)
}

// retryOperation retry operation n times
func retryOperation(tle *timer.TLE) {
	conf, _ := tle.User.(*conf.Conf)
	incCounter(conf)
	if conf.RetryCounter > conf.Env.Retries {
		resetCounter(conf)
		logger.Warning(fmt.Sprintf(
			"Bouncing %s Service %s Exceeded Retry attempts Cur: %d Max: %d",
			conf.Env.Name,
			conf.Env.Package,
			conf.RetryCounter,
			conf.Env.Retries,
		))
		bounceService(conf)
	} else {
		timer.Launch(
			timer.Name(conf.Env.Name),
			timer.Timeout(conf.Env.Interval),
			timer.Type(tle.Type),
			timer.SubType(dockerTimerRetry),
			timer.User(conf),
			timer.Key(tle.Key),
		)
		logger.Info(fmt.Sprintf(
			"Retrying Probe %s Service %s attempts Cur: %d Max: %d",
			conf.Env.Name,
			conf.Env.Package,
			conf.RetryCounter,
			conf.Env.Retries,
		))
	}
}

// armTimer launch default Docker timer
func armTimer(conf *conf.Conf) {
	timer.Launch(
		timer.Name(conf.Env.Name),
		timer.Timeout(conf.Env.Interval),
		timer.Type(DockerTimerType),
		timer.SubType(dockerTimerSubtype),
		timer.User(conf),
	)
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
		armTimer(conf)
	} else {
		metric.SetDockerMetric(0)
		if !strings.Contains(
			strings.ToLower(err.Error()), "cannot connect to the docker daemon") {
			logger.Warning(err.Error())
			bounceService(conf)
		} else {
			armTimer(conf)
		}
	}
}
