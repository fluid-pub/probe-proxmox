package manager

import (
	"fluid/probes/core/controlplane"
	"fluid/probes/core/state"
)

// Manager wraps core state persistence.
type Manager struct {
	coreManager *state.Manager
}

// NewManager accepts state.ConfigProvider (*config.Config or *core.MergedConfigProvider).
func NewManager(cfg state.ConfigProvider) (*Manager, error) {
	coreManager, err := state.NewManager(cfg)
	if err != nil {
		return nil, err
	}

	return &Manager{
		coreManager: coreManager,
	}, nil
}

func (m *Manager) Stop() {
	m.coreManager.Stop()
}

func (m *Manager) SaveEntity(entityName string, data interface{}) error {
	return m.coreManager.SaveEntity(entityName, data)
}

func (m *Manager) GetPushManager() *controlplane.PushManager {
	return m.coreManager.GetPushManager()
}

func (m *Manager) SetConfigCallbacks(getVersion controlplane.ConfigVersionFunc, onChanged controlplane.ConfigChangedCallback) {
	m.coreManager.SetConfigCallbacks(getVersion, onChanged)
}
