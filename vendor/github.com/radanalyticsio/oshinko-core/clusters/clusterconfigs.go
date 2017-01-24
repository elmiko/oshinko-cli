package clusters

import (
	"fmt"
	"strconv"
	"strings"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
)

type MetricsConfig struct {
	Enable string
	Carbon string
	Graphite string
}

type ScorpionStareConfig struct {
	Enable string
	Image string
}

type ClusterConfig struct {

	MasterCount int
	WorkerCount int
	Name string
	SparkMasterConfig string
	SparkWorkerConfig string
	Metrics MetricsConfig
	ScorpionStare ScorpionStareConfig
}


var defaultConfig ClusterConfig = ClusterConfig{
	MasterCount: 1,
	WorkerCount: 1,
	Name: "default",
	SparkMasterConfig: "",
	SparkWorkerConfig: "",
	Metrics: MetricsConfig{"","docker.io/tmckay/carbon","docker.io/tmckay/graphite"},
	ScorpionStare: ScorpionStareConfig{"","docker.io/tmckay/scorpionstare"},
}

const Defaultname = "default"
const failOnMissing = true
const allowMissing = false

const MasterCountMustBeOne = "cluster configuration must have a masterCount of 1"
const WorkerCountMustBeAtLeastOne = "cluster configuration may not have a workerCount less than 1"
const ErrorWhileProcessing = "'%s', %s"
const NamedConfigDoesNotExist = "named config '%s' does not exist"


// This function is meant to support testability
func GetDefaultConfig() ClusterConfig {
	return defaultConfig
}

func assignConfig(res *ClusterConfig, src ClusterConfig) {
	if src.MasterCount != 0 {
		res.MasterCount = src.MasterCount
	}
	if src.WorkerCount != 0 {
		res.WorkerCount = src.WorkerCount
	}

	if src.SparkMasterConfig != "" {
		res.SparkMasterConfig = src.SparkMasterConfig
	}
	if src.SparkWorkerConfig != "" {
		res.SparkWorkerConfig = src.SparkWorkerConfig
	}
	if src.Metrics.Enable != "" {
		res.Metrics.Enable = src.Metrics.Enable
	}
	if src.Metrics.Carbon != "" {
		res.Metrics.Carbon = src.Metrics.Carbon
	}
	if src.Metrics.Graphite != "" {
		res.Metrics.Graphite = src.Metrics.Graphite
	}
	if src.ScorpionStare.Enable != "" {
		res.ScorpionStare.Enable = src.ScorpionStare.Enable
	}
	if src.ScorpionStare.Image != "" {
		res.ScorpionStare.Image = src.ScorpionStare.Image
	}

}

func checkConfiguration(config ClusterConfig) error {
	var err error
	if config.MasterCount != 1 {
		err = NewClusterError(MasterCountMustBeOne, ClusterConfigCode)
	} else if config.WorkerCount < 1 {
		err = NewClusterError(WorkerCountMustBeAtLeastOne, ClusterConfigCode)
	}
	return err
}


func getInt(value, configmapname string) (int, error) {
	i, err := strconv.Atoi(strings.Trim(value, "\n"))
	if err != nil {
		err = NewClusterError(fmt.Sprintf(ErrorWhileProcessing, configmapname, "expected integer"), ClusterConfigCode)
	}
	return i, err
}

func process(config *ClusterConfig, name, value, configmapname string) error {

	var err error

	// At present we only have a single level of configs, but if/when we have
	// nested configs then we would descend through the levels beginning here with
	// the first element in the name
	switch name {
	case "mastercount":
		config.MasterCount, err = getInt(value, configmapname + ".mastercount")
	case "workercount":
		config.WorkerCount, err = getInt(value, configmapname + ".workercount")
	case "sparkmasterconfig":
                config.SparkMasterConfig = strings.Trim(value, "\n")
	case "sparkworkerconfig":
                config.SparkWorkerConfig = strings.Trim(value, "\n")
	case "metrics.enable":
		config.Metrics.Enable = strings.Trim(value, "\n")
	case "metrics.carbon":
		config.Metrics.Carbon = strings.Trim(value, "\n")
	case "metrics.graphite":
		config.Metrics.Graphite = strings.Trim(value, "\n")
	case "scorpionstare.enable":
		config.ScorpionStare.Enable = strings.Trim(value, "\n")
	case "scorpionstare.image":
		config.ScorpionStare.Image = strings.Trim(value, "\n")
	}
	return err
}

func readConfig(name string, res *ClusterConfig, failOnMissing bool, cm kclient.ConfigMapsInterface) (err error) {

	cmap, err := cm.Get(name)
	if err != nil {
		if strings.Index(err.Error(), "not found") != -1 {
			if !failOnMissing {
				err = nil
			} else {
				err = NewClusterError(fmt.Sprintf(NamedConfigDoesNotExist, name), ClusterConfigCode)
			}
		} else {
			err = NewClusterError(err.Error(), ClientOperationCode)
		}
	}
	if err == nil && cmap != nil {
		for n, v := range (cmap.Data) {
			err = process(res, n, v, name)
			if err != nil {
				break
			}
		}
	}
	return err
}

func loadConfig(name string, cm kclient.ConfigMapsInterface) (res ClusterConfig, err error) {
	// If the default config has been modified use those mods.
	res = defaultConfig
	err = readConfig(Defaultname, &res, allowMissing, cm)
	if err == nil && name != "" && name != Defaultname {
		err = readConfig(name, &res, failOnMissing, cm)
	}
	return res, err
}

func GetClusterConfig(config *ClusterConfig, cm kclient.ConfigMapsInterface) (res ClusterConfig, err error) {
        var name string = ""
	if config != nil {
	   name = config.Name
	}
	res, err = loadConfig(name, cm)
	if err == nil && config != nil {
		assignConfig(&res, *config)
	}

	// Check that the final configuration is valid
	if err == nil {
		err = checkConfiguration(res)
	}
	return res, err
}