package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/epiphany-platform/health-monitor/logger"
)

const (
	// SdNotifyReady tells the service manager that service startup is finished
	// or the service finished loading its configuration.
	SdNotifyReady = "READY=1"

	// SdNotifyStopping tells the service manager that the service is beginning
	// its shutdown.
	SdNotifyStopping = "STOPPING=1"

	// SdNotifyReloading tells the service manager that this service is
	// reloading its configuration. Note that you must call SdNotifyReady when
	// it completed reloading.
	SdNotifyReloading = "RELOADING=1"

	// SdNotifyWatchdog tells the service manager to update the watchdog
	// timestamp for the service.
	SdNotifyWatchdog = "WATCHDOG=1"
)

// SdNotify sends a message to the init daemon.
func SdNotify(unsetEnvironment bool, state string) (bool, error) {
	socketAddr := &net.UnixAddr{
		Name: os.Getenv("NOTIFY_SOCKET"),
		Net:  "unixgram",
	}

	if socketAddr.Name == "" {
		err := errors.New("systemd environment variable:'NOTIFY_SOCKET' missing")
		logger.Err(err.Error())
		return false, err
	}

	if unsetEnvironment {
		if err := os.Unsetenv("NOTIFY_SOCKET"); err != nil {
			logger.Warning((err.Error()))
		}
	}

	conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
	if err != nil {
		logger.Err((err.Error()))
		return false, err
	}

	defer conn.Close()

	if _, err = conn.Write([]byte(state)); err != nil {
		logger.Err(err.Error())
		return false, err
	}
	return true, nil
}

// SdWatchdogEnabled retrieves watchdog environment variable
func SdWatchdogEnabled(unsetEnvironment bool) (int, error) {
	wusec := os.Getenv("WATCHDOG_USEC")
	wpid := os.Getenv("WATCHDOG_PID")

	if unsetEnvironment {
		wusecErr := os.Unsetenv("WATCHDOG_USEC")
		wpidErr := os.Unsetenv("WATCHDOG_PID")
		if wusecErr != nil {
			logger.Err(wusecErr.Error())
			return 0, wusecErr
		}
		if wpidErr != nil {
			logger.Err(wpidErr.Error())
			return 0, wpidErr
		}
	}

	if wusec == "" {
		logger.Info("Watchdog Timer NOT defined, Watchdog Timer will NOT be used.")
		return 0, nil
	}

	s, err := strconv.Atoi(wusec)
	if err != nil {
		err := fmt.Errorf("Watchdog Timer NOT be used, error converting WATCHDOG_USEC: %s", err)
		logger.Err(err.Error())
		return 0, err
	}

	if s <= 0 {
		err := fmt.Errorf("Watchdog timer NOT used, error WATCHDOG_USEC must be a positive number")
		logger.Err(err.Error())
		return 0, err
	}

	interval := s / 3

	if wpid != "" {
		p, err := strconv.Atoi(wpid)
		if err != nil {
			logger.Err(fmt.Errorf("Watchdog Timer NOT be used,  error converting WATCHDOG_PID: %s ", err).Error())
			return 0, err
		}

		if os.Getpid() != p {
			err := fmt.Errorf("Watchdog Timer NOT be used. error converting WATCHDOG_PID: %s", err)
			logger.Err(err.Error())
			return 0, err
		}
	}
	return interval / 1000000, nil
}
