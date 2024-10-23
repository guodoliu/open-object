package version

import "fmt"

var (
	Name             = "open-object"
	Version   string = ""
	GitCommit string = ""
)

func GetFullVersion(longFormat bool) string {
	git := GitCommit
	if !longFormat && GitCommit != "" {
		git = GitCommit[:12]
	}
	return fmt.Sprintf("%s-%s", Version, git)
}

func NameWithVersion(longFormat bool) string {
	return fmt.Sprintf("%s-%s", Name, GetFullVersion(longFormat))
}
