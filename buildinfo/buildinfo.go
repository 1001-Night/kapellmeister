package buildinfo

var (
	Version = "0.0.1-dev"
	Commit  = "unknown"
)

type BuildInfo struct {
	Version string
	Commit  string
}

func BuildOptions() BuildInfo {
	return BuildInfo{
		Version: Version,
		Commit:  Commit,
	}
}