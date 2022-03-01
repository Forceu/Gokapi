package environment

/**
Variables that are set during build
*/

// IsDocker has to be true if compiled for the Docker image (auto-generated value)
var IsDocker = "false"

// BuildTime is the time of the build (auto-generated value)
var BuildTime = "Dev Build"

// Builder is the name of builder (auto-generated value)
var Builder = "Manual Build"

// IsDockerInstance returns true if the binary was compiled with the official docker makefile, which
// sets IsDocker to true
func IsDockerInstance() bool {
	return IsDocker != "false"
}
