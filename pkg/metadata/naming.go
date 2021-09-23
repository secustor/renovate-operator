package metadata

import "strings"

const (
	discoveryGroupName = "discovery"
	workerGroupName    = "worker"
)

func buildName(name string, group string) string {
	builder := strings.Builder{}
	builder.WriteString(name)
	builder.WriteRune('-')
	builder.WriteString(group)
	return builder.String()
}
