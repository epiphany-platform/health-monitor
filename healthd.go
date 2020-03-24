package main

import (
	"flag"
	"os"

	"github.com/health-monitor/channel"
	"github.com/health-monitor/conf"
	"github.com/health-monitor/docker"
	"github.com/health-monitor/http"
	"github.com/health-monitor/logger"
	"github.com/health-monitor/metric"
	daemon "github.com/health-monitor/notify"
	"github.com/health-monitor/timer"
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
	metric.Run()
}

// Run Docker Probes
func init() {
	docker.Run()
}

// Run HTTP Probes
func init() {
	http.Run()
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
}

func main() {
	waitTimerCompletions()
	os.Exit(0)
}
