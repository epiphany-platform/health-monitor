package conf

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/healthd/logger"
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

const (
	dockerType  = "docker"
	kubeletType = "kubelet"
)

// Len return the number liveness probes configure
func Len() int {
	return len(Confs)
}

// New allocates memory and return pointer newly allocated zero value of that type
func New() *Conf {
	return new(Conf)
}

// Exist checks an entry already exists and returnthe index and entry
func Exist(Name string) (pos int, conf *Conf) {
	for pos, conf = range Confs {
		if conf.Env.Name == Name {
			return pos, conf
		}
	}
	return -1, nil
}

// Unmarshal YAML conf file
func Unmarshal(b []byte) (err error) {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	for {
		conf := New()
		if err = dec.Decode(conf); err == nil {
			if pos, _ := Exist(conf.Env.Name); pos == -1 {
				Confs = append(Confs, conf)
			} else {
				Confs[pos].Env = conf.Env
			}
			continue
		}
		if err == io.EOF {
			return nil
		}
		return
	}
}

// Load YAML conf into memory
func Load(fileName string) (err error) {
	var yamlFile string
	if yamlFile, err = filepath.Abs(fileName); err != nil {
		logger.Err(err.Error())
		return
	}

	var yamlBuf []byte
	if yamlBuf, err = ioutil.ReadFile(yamlFile); err != nil {
		logger.Err(err.Error())
		return
	}
	return Unmarshal(yamlBuf)
}
