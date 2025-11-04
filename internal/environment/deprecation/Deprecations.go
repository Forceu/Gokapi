package deprecation

import (
	"os"
	"strings"
)

type Deprecation struct {
	Id            string
	Name          string
	Description   string
	DocUrl        string
	checkFunction func() bool
}

func (d *Deprecation) IsSet() bool {
	if d.checkFunction == nil {
		panic("checkFunction is nil")
	}
	return d.checkFunction()
}

func GetActive() []Deprecation {
	result := make([]Deprecation, 0)
	for _, deprecation := range availableDeprecations {
		if deprecation.IsSet() {
			result = append(result, deprecation)
		}
	}
	return result
}

var availableDeprecations = []Deprecation{
	{
		Id:            "dockernonroot",
		Name:          "Docker Non-Root User",
		Description:   "Usage of DOCKER_NONROOT is deprecated in favor of docker --user option",
		DocUrl:        "https://gokapi.readthedocs.io/en/latest/setup.html#migration-from-docker-nonroot-to-docker-user",
		checkFunction: isNonRootSet,
	},
}

func isNonRootSet() bool {
	envVal := os.Getenv("DOCKER_NONROOT")
	if envVal == "" || strings.ToLower(envVal) == "false" {
		return false
	}
	return true
}
