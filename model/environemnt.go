package model

type Environment struct {
	Name              string              `yaml:"name"`
	Description       string              `yaml:"description"`
	Workspace         string              `yaml:"workspace"`
	Target            Target              `yaml:"target"`
	Services          map[string]*Service `yaml:"services"`
	Status            string              `yaml:"status"`
	ContainerRegistry *Registry           `yaml:"registry,omitempty"`
}

type Registry struct {
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`
}

type Service struct {
	Name              string            `yaml:"name"`
	Description       string            `yaml:"description,omitempty"`
	Type              string            `yaml:"type"`
	Params            map[string]string `yaml:"params"`
	DependsOn         []string          `yaml:"depends_on,omitempty"`
	Build             *BuildConfig      `yaml:"build,omitempty"`
	PreRun            []*Command        `yaml:"pre-run,omitempty"`
	Run               *RunConfig        `yaml:"run"`
	PostRun           []Command         `yaml:"post-run,omitempty"`
	Status            string            `yaml:"status"`
	ContainerRegistry *Registry         `yaml:"registry,omitempty"`
}

type BuildConfig struct {
	Type   string            `yaml:"type"`
	Params map[string]string `yaml:"params"`
}

type Command struct {
	Cmd  string   `yaml:"cmd"`
	Args []string `yaml:"args"`
}

type RunConfig struct {
	Cmd    string           `yaml:"cmd"`
	Args   []string         `yaml:"args"`
	EnVars []EnVar          `yaml:"envars"`
	Ports  []Port           `yaml:"ports"`
	Mounts map[string]Mount `yaml:"mounts"`
}

type Mount struct {
	Name       string   `yaml:"name"`
	SourcePath string   `yaml:"source_path"`
	Path       string   `yaml:"path"`
	Configs    []Config `yaml:"files"`
}

type Config struct {
	ConfigName string `yaml:"name"`
	Content    string `yaml:"content"`
}

type Port struct {
	Port     string `yaml:"port"`
	HostPort string `yaml:"hostport"`
	Exposed  bool   `yaml:"exposed"`
}

type EnVar struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type Target struct {
	Name   string            `yaml:"name"`
	Type   string            `yaml:"type"`
	Params map[string]string `yaml:"params"`
}
