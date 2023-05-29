package model

type WorkspaceMode int64

const (
	Undefined WorkspaceMode = iota
	Local
)

const ACTIVE_STATUS = "Active"
const INACTIVE_STATUS = "Inactive"

func (s WorkspaceMode) String() string {
	switch s {
	case Local:
		return "local"
	}
	return "unknown"
}
