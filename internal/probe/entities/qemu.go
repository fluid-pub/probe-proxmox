package entities

import (
	"fmt"
	"log"

	"fluid/probes/core"
	"fluid/probes/proxmox/internal/models"
	"fluid/probes/proxmox/internal/proxmox"
)

// QemuEntity syncs QEMU/KVM guests (PVE cluster resource type "qemu") into state.
type QemuEntity struct{}

// NewQemuEntity registers the qemu entity (same as PVE JSON field "type": "qemu").
func NewQemuEntity() *QemuEntity {
	return &QemuEntity{}
}

func (e *QemuEntity) Name() string {
	return "qemu"
}

func (e *QemuEntity) Refresh(client core.Client) (interface{}, error) {
	pveClient, ok := client.(*proxmox.Client)
	if !ok {
		return nil, fmt.Errorf("invalid client type for qemu entity, expected *proxmox.Client")
	}

	log.Println("Retrieving Proxmox QEMU VMs...")

	vms, err := pveClient.GetQEMUVms()
	if err != nil {
		return nil, fmt.Errorf("retrieve QEMU VMs: %w", err)
	}

	log.Printf("Retrieved %d QEMU VMs", len(vms))
	return vms, nil
}

func (e *QemuEntity) Save(stateManager core.StateManager, data interface{}) error {
	vms, ok := data.([]models.QemuVM)
	if !ok {
		return fmt.Errorf("invalid data type for qemu entity")
	}

	if err := stateManager.SaveEntity(e.Name(), vms); err != nil {
		return fmt.Errorf("save QEMU VMs: %w", err)
	}

	log.Printf("QEMU VMs state saved successfully")
	return nil
}
