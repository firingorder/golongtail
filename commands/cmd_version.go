package commands

import "fmt"

var (
	BuildVersion string = "<none>"
)

type VersionCmd struct {
}

func (r *VersionCmd) Run(ctx *Context) error {
	fmt.Println(BuildVersion)
	return nil
}