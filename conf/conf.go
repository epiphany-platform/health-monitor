package conf

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/epiphany-platform/health-monitor/logger"
	"github.com/epiphany-platform/health-monitor/timer"
	"gopkg.in/yaml.v3"
)

type (
	// Conf Liveness monitor configuration
	Conf struct {
		RetryCounter int
		RestartCount uint32
		Env          struct {
			Name            string `yaml:"Name"`
			Package         string `yaml:"Package"`
			ActionFatal     bool   `yaml:"ActionFatal"`
			IP              string `yaml:"IP,omitempty"`
			Interval        int    `yaml:"Interval"`
			Path            string `yaml:"Path,omitempty"`
			Port            int    `yaml:"Port,omitempty"`
			RequestType     string `yaml:"RequestType,omitempty"`
			Response        string `yaml:"Response,omitempty"`
			Retries         int    `yaml:"Retries"`
			RetryDelay      int    `yaml:"RetryDelay"`
			RecoveryDelay   int    `yaml:"RecoveryDelay"`
			ProtocolTimeout int    `yaml:"ProtocolTimeout"`
		} `yaml:"Env"`
	}
)

var (
	// Confs Array of Liveness monitor configuration Probe
	Confs = make(map[string]*Conf)
)

// Run launch specified client timer
func Run(pkg string, pkgType, pkgSubType int) {
	for _, conf := range Confs {
		if strings.EqualFold(conf.Env.Package, pkg) {
			timer.Launch(
				timer.Name(conf.Env.Name),
				timer.Timeout(conf.Env.Interval),
				timer.Type(pkgType),
				timer.SubType(pkgSubType),
				timer.User(conf),
			)
		}
	}
}

// Len return the number liveness probes configure
func Len() int {
	return len(Confs)
}

// New allocates memory and return pointer newly allocated zero value of that type
func New() *Conf {
	return new(Conf)
}

func isDockerNormlize(conf *Conf) error {
	if strings.EqualFold("docker", conf.Env.Package) {
		if net.ParseIP(conf.Env.IP) == nil {
			if host := os.Getenv("DOCKER_HOST"); host != "" {
				conf.Env.IP = host
			} else {
				conf.Env.IP = client.DefaultDockerHost
			}
			if conf.Env.Port == 0 {
				conf.Env.Port = 2375
			}
		}
	}
	return nil
}

func isHTTPNormalize(conf *Conf) error {
	if strings.EqualFold("http", conf.Env.Package) {
		if net.ParseIP(conf.Env.IP) == nil {
			return errors.New("YAML IP address out-of-bound")
		}
		if conf.Env.Port == 0 || conf.Env.Path == "" {
			return errors.New("YAML Port or Path missing")
		}
		if conf.Env.RequestType == "" {
			return errors.New("YAML Request Typpe must be specified")
		}
	}
	return nil
}

// IsNormalize ensure conf consistency
func IsNormalize(conf *Conf) error {
	if conf.Env.Name == "" {
		return errors.New("YAML Name NOT defined")
	}

	if conf.Env.Package == "" {
		return errors.New("YAML Package NOT defined")
	}

	if !(conf.Env.Retries >= 3 && conf.Env.Retries <= 10) {
		return errors.New("YAML Retries out-of-range")
	}

	if !(conf.Env.Interval >= 5 && conf.Env.Interval <= 300) {
		return errors.New("YAML Interval out-of-range")
	}

	if !(conf.Env.RetryDelay >= 5 && conf.Env.RetryDelay <= 120) {
		return errors.New("YAML RetryDelay out-of-range")
	}

	if !(conf.Env.RecoveryDelay >= 10 && conf.Env.RecoveryDelay <= 300) {
		return errors.New("YAML RecoveryDelay out-of-range")
	}

	if !(conf.Env.ProtocolTimeout >= 2 && conf.Env.ProtocolTimeout <= 300) {
		return errors.New("YAML ProtocolTimeout out-of-range")
	}

	if err := isDockerNormlize(conf); err != nil {
		return err
	}

	if err := isHTTPNormalize(conf); err != nil {
		return err
	}
	return nil
}

// Unmarshal YAML conf file
func Unmarshal(b []byte) (err error) {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	for {
		conf := New()
		if err = dec.Decode(conf); err == nil {
			if err := IsNormalize(conf); err != nil {
				logger.Err(err.Error())
				panic(err)
			}
			if Confs[conf.Env.Name] == nil {
				Confs[conf.Env.Name] = conf
			} else {
				Confs[conf.Env.Name].Env = conf.Env
			}
			continue
		}
		if err == io.EOF {
			return nil
		}
		logger.Err(err.Error())
		panic(err)
	}
}

// Load YAML conf into memory
func Load(fName string) error {
	yamlFname, err := filepath.Abs(fName)
	if err != nil {
		logger.Err(err.Error())
		return err
	}

	yamlBuf, err := ioutil.ReadFile(yamlFname)
	if err != nil {
		logger.Err(err.Error())
		return err
	}
	return Unmarshal(yamlBuf)
}
