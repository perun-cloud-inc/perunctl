package model

type WorkspaceMode int64

const (
	ActiveStatus                 = "Active"
	InactiveStatus               = "Inactive"
	Local          WorkspaceMode = iota
)

func (s WorkspaceMode) String() string {
	switch s {
	case Local:
		return "local"
	}
	return "unknown"
}
