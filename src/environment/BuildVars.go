package environment

/**
Variables that are set during build
*/

// Has to be true if compiled for the Docker image
var IsDocker = "false"

// Time of the build
var BuildTime = "Dev Build"

// Name of builder
var Builder = "Manual Build"
