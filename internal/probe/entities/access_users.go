package entities

import (
	"fmt"
	"log"

	"fluid/probes/core"
	"fluid/probes/proxmox/internal/models"
	"fluid/probes/proxmox/internal/proxmox"
)

// AccessUsersEntity syncs Proxmox access users into state (no passwords or API secrets).
type AccessUsersEntity struct{}

// NewAccessUsersEntity registers the access_users entity.
func NewAccessUsersEntity() *AccessUsersEntity {
	return &AccessUsersEntity{}
}

func (e *AccessUsersEntity) Name() string {
	return "access_users"
}

func (e *AccessUsersEntity) Refresh(client core.Client) (interface{}, error) {
	pveClient, ok := client.(*proxmox.Client)
	if !ok {
		return nil, fmt.Errorf("invalid client type for access_users entity, expected *proxmox.Client")
	}

	log.Println("Retrieving Proxmox access users...")

	users, err := pveClient.GetAccessUsers()
	if err != nil {
		return nil, fmt.Errorf("retrieve access users: %w", err)
	}

	log.Printf("Retrieved %d access users", len(users))
	return users, nil
}

func (e *AccessUsersEntity) Save(stateManager core.StateManager, data interface{}) error {
	users, ok := data.([]models.AccessUser)
	if !ok {
		return fmt.Errorf("invalid data type for access_users entity")
	}

	if err := stateManager.SaveEntity(e.Name(), users); err != nil {
		return fmt.Errorf("save access users: %w", err)
	}

	log.Printf("Access users state saved successfully")
	return nil
}
