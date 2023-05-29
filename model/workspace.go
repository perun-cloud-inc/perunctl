package model

type Workspace struct {
	Name string `yaml:"name"`
	Mode string `yaml:"mode"`

	Environments []*Environment `yaml:"environments"`
}
