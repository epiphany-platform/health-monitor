package docker

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/epiphany-platform/health-monitor/conf"
	"github.com/epiphany-platform/health-monitor/logger"
	"github.com/epiphany-platform/health-monitor/metric"
	"github.com/epiphany-platform/health-monitor/timer"
)
// operationTimeout is the error returned when the docker operations are timeout.
type operationTimeout struct {
	err error
}

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

// recoveryDelayTimer initiate Recovery Delay timer allow service to recover
func recoveryDelayTimer(conf *conf.Conf) {
	timer.Launch(
		timer.Name(conf.Env.Name),
		timer.Timeout(conf.Env.RecoveryDelay),
		timer.Type(DockerTimerType),
		timer.SubType(dockerTimerWait),
		timer.User(conf),
	)
	logger.Info(
		fmt.Sprintf("Service %s Probe Delayed %d secs, allowance recovery of resources.",
			conf.Env.Name,
			conf.Env.RecoveryDelay),
	)
}

func dumpDockerDaemon(conf *conf.Conf) {
	cmd := exec.Command("pkill", "-SIGUSR1", "docker")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		logger.Err(err.Error())
	}
	if len(out.String()) > 0 {
		logger.Info(out.String())
	} else {
		logger.Info(fmt.Sprintf(
			"Name: %s Service: %s stack dumped for investigation.",
			conf.Env.Name,
			conf.Env.Package,
		))
	}
}

func killDockerDaemon(conf *conf.Conf) {
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
			"Name: %s Service: %s Restart Completed",
			conf.Env.Name,
			conf.Env.Package,
		))
	}
}

// restartService running docker daemon
func bounceService(conf *conf.Conf) {
	if !conf.Env.ActionFatal {
		resetCounter(conf)
		recoveryDelayTimer(conf)
	} else {
		dumpDockerDaemon(conf)
		killDockerDaemon(conf)
		metric.IncrementRestartCount()
		recoveryDelayTimer(conf)
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

// retryServiceTimer initiates timer to retry probe
func retryServiceTimer(conf *conf.Conf) {
	timer.Launch(
		timer.Name(conf.Env.Name),
		timer.Timeout(conf.Env.RetryDelay),
		timer.Type(DockerTimerType),
		timer.SubType(dockerTimerRetry),
		timer.User(conf),
	)
	logger.Info(fmt.Sprintf(
		"Retrying Probe %s Service %s attempts Cur: %d Max: %d",
		conf.Env.Name,
		conf.Env.Package,
		conf.RetryCounter,
		conf.Env.Retries,
	))
}

// retryOperation retry operation n times
func restartService(conf *conf.Conf) {
	if incCounter(conf) <= conf.Env.Retries {
		retryServiceTimer(conf)
	} else {
		logger.Warning(fmt.Sprintf(
			"Bouncing %s Service %s Exceeded Retry attempts Cur: %d Max: %d",
			conf.Env.Name,
			conf.Env.Package,
			conf.RetryCounter,
			conf.Env.Retries,
		))
		resetCounter(conf)
		bounceService(conf)
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

// EncodeURL format URL components to facilitate connection
func EncodeURL(scheme string, host string, port int, path string) *url.URL {
	return &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   path,
	}
}
func (e operationTimeout) Error() string {
	return fmt.Sprintf("operation timeout: %v", e.err)
}

// contextError checks the context, and returns error if the context is timeout.
func contextError(ctx context.Context) error {
	if ctx.Err() == context.DeadlineExceeded {
		return operationTimeout{err: ctx.Err()}
	}
	return ctx.Err()
}

// probeDocker running containers
func probeDocker(conf *conf.Conf) error {
	cli, err := client.NewClient(conf.Env.IP, "", nil, nil)
	if err != nil {
		return err
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(conf.Env.ProtocolTimeout) * time.Second)
	defer cancel()

	cli.NegotiateAPIVersion(ctx)
	_, err = cli.ContainerList(ctx, types.ContainerListOptions{})
	if ctxErr := contextError(ctx); ctxErr != nil {
		return ctxErr
	}


	cli.Close()
	return err
}

// resetCounter retry counter
func resetCounter(conf *conf.Conf) {
	conf.RetryCounter = 0
}

// incCounter increment retry counter
func incCounter(conf *conf.Conf) int {
	conf.RetryCounter++
	return conf.RetryCounter
}

// Probe specified docker HTTP endpoint
func Probe(tle *timer.TLE) {
	conf, _ := tle.User.(*conf.Conf)
	if err := probeDocker(conf); err == nil {
		metric.SetDockerMetric(1)
		resetCounter(conf)
		armTimer(conf)
	} else {
		metric.SetDockerMetric(0)
		if !strings.Contains(
			strings.ToLower(err.Error()), "cannot connect to the docker daemon") {
			logger.Warning(err.Error())
			restartService(conf)
		} else {
			armTimer(conf)
		}
	}
}
