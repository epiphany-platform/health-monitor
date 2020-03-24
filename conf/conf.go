package conf

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"

	"github.com/health-monitor/logger"
	"github.com/health-monitor/timer"
	"gopkg.in/yaml.v3"
)

type (
	// Conf Liveness monitor configuration
	Conf struct {
		RetryCounter int
		RestartCount uint32
		Env          struct {
			Name        string `yaml:"Name"`
			Package     string `yaml:"Package"`
			ActionFatal bool   `yaml:"ActionFatal"`
			IP          string `yaml:"IP"`
			Interval    int    `yaml:"Interval"`
			Path        string `yaml:"Path"`
			Port        int    `yaml:"Port"`
			RequestType string `yaml:"RequestType"`
			Response    string `yaml:"Response"`
			Retries     int    `yaml:"Retries"`
			RetryDelay  int    `yaml:"RetryDelay"`
		} `yaml:"Env"`
	}
)

var (
	// Confs Array of Liveness monitor configuration Probes
	Confs []*Conf
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

// Exist checks an entry already exists and returnthe index and entry
func Exist(Name string) (int, bool) {
	for pos, conf := range Confs {
		if conf.Env.Name == Name {
			return pos, true
		}
	}
	return -1, false
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
	if !(conf.Env.Interval >= 10 && conf.Env.Interval <= 59) {
		return errors.New("YAML Interval out-of-range")
	}

	if !(conf.Env.RetryDelay >= 5 && conf.Env.RetryDelay <= 15) {
		return errors.New("YAML RetryDelay out-of-range")
	}
	if strings.EqualFold("http", conf.Env.Package) {
		if net.ParseIP(conf.Env.IP) == nil {
			return errors.New("YAML IP address out-of-bound")
		}
		if conf.Env.Port == 0 || conf.Env.Path == "" {
			return errors.New("YAML Port or Path missing")
		}
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
			if pos, ok := Exist(conf.Env.Name); !ok {
				Confs = append(Confs, conf)
			} else {
				Confs[pos].Env = conf.Env
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
