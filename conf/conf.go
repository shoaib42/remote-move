package conf

import (
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type VoidT struct{}

type Configuration struct {
	SrcDirs        []string `yaml:"srcDirs"`
	DestRootDir    string   `yaml:"destRootDir"`
	ExcludeDirs    []string `yaml:"excludeDirs"`
	AllowedCIDRs   []string `yaml:"allowedCIDRs"`
	ServerBindAddr string   `yaml:"serverBindAddr"`
	ServerBindPort string   `yaml:"serverBindPort"`
	ChownUsrGrp    string   `yaml:"chownUsrGrp"`
	Uid            int
	Gid            int
}

var Void VoidT

var Confs Configuration

func LoadConfiguration(filepath string) error {
	yamlFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &Confs)
	if err != nil {
		return err
	}

	uid_gid := strings.Split(Confs.ChownUsrGrp, ":")
	if 2 != len(uid_gid) {
		log.Fatal("failed to parse chownUsrGrp, should be of format Uid:Gid (127:127)")
	}
	if Confs.Uid, err = strconv.Atoi(uid_gid[0]); nil != err {
		log.Fatal("failed to parse usr_id from chownUsrGrp, provided : " + uid_gid[0] + " should numeric int value")
	}
	if Confs.Gid, err = strconv.Atoi(uid_gid[1]); nil != err {
		log.Fatal("failed to parse grp_id from chownUsrGrp, provided : " + uid_gid[1] + " should numeric int value")
	}
	return nil
}
