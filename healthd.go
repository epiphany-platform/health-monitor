package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/healthd/channel"
	"github.com/healthd/conf"
	"github.com/healthd/docker"
	"github.com/healthd/http"
	"github.com/healthd/logger"
	"github.com/healthd/metric"
	daemon "github.com/healthd/notify"
	"github.com/healthd/timer"
)

const (
	// WatchDog related timer information
	watchdogName    = "Watchdog"
	watchdogType    = 1001
	watchdogSubtype = 1002
)

var (

	// health liveness configuration file
	healthdConf = flag.String("-c", "healthd.yml", "YAML configuation file")
	// health liveness prometheus port #
	healthdPort = flag.String("-p", "2112", "Prometheus IP port #")
)

// Notify systemd startup/initialsation success
func init() {
	if ok, err := daemon.SdNotify(false, daemon.SdNotifyReady); !ok {
		logger.Err(err.Error())
		panic(err)
	}
}

// Initial logger interface to syslog
func init() {
	if err := logger.Init(); err != nil {
		panic(err)
	}
}

// Initial load health liveness check configuration
func init() {
	if err := conf.Load(*healthdConf); err != nil {
		logger.Err(err.Error())
		panic(err)
	}
}

// Setup watch watchdog timer
func init() {
	interval, err := daemon.SdWatchdogEnabled(false)
	if err != nil {
		logger.Err(err.Error())
		panic(err)
	} else {
		timer.Launch(
			timer.Name(watchdogName),
			timer.Timeout(int(interval)),
			timer.Type(watchdogType),
			timer.SubType(watchdogSubtype),
		)
	}
}

// Run Prometheus Metrics
func init() {
	metric.Run(healthdPort)
}

// Run Docker Probes
func init() {
	docker.Run()
}

// Run HTTP Probes
func init() {
	http.Run()
}

// daemonSignals catch specific signals
func init() {
	daemonChan := make(chan os.Signal, 1)

	signal.Notify(
		daemonChan,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGABRT,
		syscall.SIGTERM,
		syscall.SIGKILL,
	)

	go func() {
		for {
			switch <-daemonChan {
			case syscall.SIGHUP:
				{
					logger.Info(fmt.Sprintf("Reloading %s ", *healthdConf))
					daemon.SdNotify(false, daemon.SdNotifyReloading)
					if err := conf.Load(*healthdConf); err != nil {
						logger.Err(err.Error())
						panic(err)
					}
					daemon.SdNotify(false, daemon.SdNotifyReady)
				}

			case syscall.SIGQUIT:
			case syscall.SIGABRT:
			case syscall.SIGKILL:
			case syscall.SIGTERM:
				{
					daemon.SdNotify(false, daemon.SdNotifyStopping)
					os.Exit(0)
				}
			}
		}
	}()
}

// init completed
func init() {
	logger.Info("Completed initialization")
}

// Sends watchDog notify and timer setup
func watchDog(tle *timer.TLE) {
	daemon.SdNotify(false, daemon.SdNotifyWatchdog)
	interval, err := daemon.SdWatchdogEnabled(false)
	if err != nil {
		logger.Err(err.Error())
		panic(err)
	} else {
		timer.Launch(
			timer.Name(tle.Name),
			timer.Timeout(int(interval)),
			timer.Type(tle.Type),
			timer.SubType(tle.SubType),
		)
	}
}

// Orchestrate timer Completions
func orchestrate(tle *timer.TLE) {
	switch tle.Type {
	case watchdogType:
		{
			watchDog(tle)
		}
	case docker.DockerTimerType:
		{
			docker.Probe(tle)
		}
	case http.HTTPTimerType:
		{
			http.Probe(tle)
		}
	}
}

// WaitTimer channel timer completions
func waitTimerCompletions() {
	for channel.Active() {
		_, Chosen, message, _ := channel.Awaitio()
		switch message.Type().String() {
		case timer.Completion:
			{
				tle, _ := message.Interface().(*timer.TLE)
				orchestrate(tle)
				channel.Remove(Chosen)
			}
		}
	}
	logger.Err("Internal logic error, timer(s) NOT running.")
}

func main() {
	waitTimerCompletions()
	os.Exit(1)
}
