// Package version is used to set and retrieve the App Version
package version

// Note that the appVersion is set by goreleaser and Dockerfile
const appVersion = "v0.0.0"

// GetAppVersion can be leveraged to get a string representation of the current version of PgFga
func GetAppVersion() string {
	return appVersion
}
