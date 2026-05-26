package entities

import (
	"fmt"
	"log"

	"fluid/probes/core"
	"fluid/probes/proxmox/internal/models"
	"fluid/probes/proxmox/internal/proxmox"
)

// ClusterResourcesEntity syncs GET /cluster/resources into state.
type ClusterResourcesEntity struct{}

// NewClusterResourcesEntity registers the cluster_resources entity.
func NewClusterResourcesEntity() *ClusterResourcesEntity {
	return &ClusterResourcesEntity{}
}

func (e *ClusterResourcesEntity) Name() string {
	return "cluster_resources"
}

func (e *ClusterResourcesEntity) Refresh(client core.Client) (interface{}, error) {
	pveClient, ok := client.(*proxmox.Client)
	if !ok {
		return nil, fmt.Errorf("invalid client type for cluster_resources entity, expected *proxmox.Client")
	}

	log.Println("Retrieving Proxmox cluster resources...")

	resources, err := pveClient.GetClusterResources()
	if err != nil {
		return nil, fmt.Errorf("retrieve cluster resources: %w", err)
	}

	log.Printf("Retrieved %d cluster resources", len(resources))
	return resources, nil
}

func (e *ClusterResourcesEntity) Save(stateManager core.StateManager, data interface{}) error {
	resources, ok := data.([]models.ClusterResource)
	if !ok {
		return fmt.Errorf("invalid data type for cluster_resources entity")
	}

	if err := stateManager.SaveEntity(e.Name(), resources); err != nil {
		return fmt.Errorf("save cluster resources: %w", err)
	}

	log.Printf("Cluster resources state saved successfully")
	return nil
}
