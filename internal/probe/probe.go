package probe

import (
	"fmt"
	"log"

	"fluid/probes/core"
	"fluid/probes/core/state"
	"fluid/probes/proxmox/internal/config"
	"fluid/probes/proxmox/internal/manager"
	"fluid/probes/proxmox/internal/probe/entities"
	"fluid/probes/proxmox/internal/proxmox"
)

// Probe wraps the core probe with Proxmox-specific wiring.
type Probe struct {
	*core.Probe
	config *config.Config
	client *proxmox.Client
}

func getProxmoxConfig(cfg state.ConfigProvider) *config.Config {
	if c, ok := cfg.(*config.Config); ok {
		return c
	}
	if m, ok := cfg.(*core.MergedConfigProvider); ok {
		if c, ok := m.Local().(*config.Config); ok {
			return c
		}
	}
	return nil
}

// NewProbe accepts state.ConfigProvider (*config.Config or *core.MergedConfigProvider).
func NewProbe(cfg state.ConfigProvider) (*Probe, error) {
	proxmoxCfg := getProxmoxConfig(cfg)
	if proxmoxCfg == nil {
		return nil, fmt.Errorf("NewProbe requires *config.Config or *core.MergedConfigProvider with *config.Config as local")
	}

	client, err := proxmox.NewClient(&proxmoxCfg.Proxmox)
	if err != nil {
		return nil, fmt.Errorf("proxmox client: %w", err)
	}

	stateManager, err := manager.NewManager(cfg)
	if err != nil {
		return nil, err
	}

	coreProbe := core.NewProbe(cfg, client, stateManager)

	a := &Probe{
		Probe:  coreProbe,
		config: proxmoxCfg,
		client: client,
	}

	coreProbe.RegisterEntity(entities.NewClusterResourcesEntity())
	coreProbe.RegisterEntity(entities.NewQemuEntity())
	coreProbe.RegisterEntity(entities.NewAccessUsersEntity())

	if merged, ok := cfg.(*core.MergedConfigProvider); ok && stateManager.GetPushManager() != nil {
		stateManager.SetConfigCallbacks(
			merged.GetConfigVersion,
			func(runtimeJSON []byte, configVersion string) {
				runtime, version, err := core.ParseRuntimeConfig(runtimeJSON)
				if err != nil {
					log.Printf("Parse runtime config on reload: %v", err)
					return
				}
				if version != "" {
					configVersion = version
				}
				if err := merged.SetRemote(runtime, configVersion); err != nil {
					log.Printf("Set remote config on reload: %v", err)
					return
				}
				if err := coreProbe.ReloadConfig(); err != nil {
					log.Printf("ReloadConfig failed: %v", err)
				}
			},
		)
	}

	return a, nil
}

func (a *Probe) Start() error {
	log.Println("Proxmox probe starting...")
	return a.Probe.Start()
}

func (a *Probe) GetStatus() map[string]interface{} {
	status := a.Probe.GetStatus()
	status["proxmox_configured"] = true
	return status
}
