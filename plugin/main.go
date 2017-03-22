package main

import "C"
import _ "github.com/robotalks/talk-vision/vision"

// Version number
const Version = "0.1.0"

// VersionSuffix provides suffix info
var VersionSuffix = "-dev"

// PluginVersion returns the version of plugin
func PluginVersion() string {
	return Version + VersionSuffix
}
