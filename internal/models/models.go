package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CalculateFluid returns a stable anonymized hash for an arbitrary id string.
func CalculateFluid(id string) string {
	hash := sha256.Sum256([]byte(id))
	return hex.EncodeToString(hash[:])
}

// ClusterResource is one row from GET /cluster/resources (nodes, qemu, lxc, storage, etc.).
type ClusterResource struct {
	ID         string   `yaml:"id" json:"id"`
	Fluid      string   `yaml:"fluid" json:"fluid"`
	Type       string   `yaml:"type" json:"type"`
	VMID       *int     `yaml:"vmid,omitempty" json:"vmid,omitempty"`
	Node       string   `yaml:"node,omitempty" json:"node,omitempty"`
	Name       string   `yaml:"name,omitempty" json:"name,omitempty"`
	Status     string   `yaml:"status,omitempty" json:"status,omitempty"`
	Pool       string   `yaml:"pool,omitempty" json:"pool,omitempty"`
	Storage    string   `yaml:"storage,omitempty" json:"storage,omitempty"`
	Template   *int     `yaml:"template,omitempty" json:"template,omitempty"`
	MaxCPU     *float64 `yaml:"maxcpu,omitempty" json:"maxcpu,omitempty"`
	MaxDisk    *uint64  `yaml:"maxdisk,omitempty" json:"maxdisk,omitempty"`
	Disk       *uint64  `yaml:"disk,omitempty" json:"disk,omitempty"`
	MaxMem     *uint64  `yaml:"maxmem,omitempty" json:"maxmem,omitempty"`
	Mem        *uint64  `yaml:"mem,omitempty" json:"mem,omitempty"`
	CPU        *float64 `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Uptime     *uint64  `yaml:"uptime,omitempty" json:"uptime,omitempty"`
	Running    *int     `yaml:"running,omitempty" json:"running,omitempty"`
	CgroupMode *int     `yaml:"cgroup_mode,omitempty" json:"cgroup_mode,omitempty"`
}

// QemuVM is a QEMU/KVM guest from cluster resources (type qemu).
type QemuVM struct {
	ID       string   `yaml:"id" json:"id"`
	Fluid    string   `yaml:"fluid" json:"fluid"`
	VMID     int      `yaml:"vmid" json:"vmid"`
	Node     string   `yaml:"node,omitempty" json:"node,omitempty"`
	Name     string   `yaml:"name,omitempty" json:"name,omitempty"`
	Status   string   `yaml:"status,omitempty" json:"status,omitempty"`
	Pool     string   `yaml:"pool,omitempty" json:"pool,omitempty"`
	Template *int     `yaml:"template,omitempty" json:"template,omitempty"`
	MaxCPU   *float64 `yaml:"maxcpu,omitempty" json:"maxcpu,omitempty"`
	CPU      *float64 `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	MaxDisk  *uint64  `yaml:"maxdisk,omitempty" json:"maxdisk,omitempty"`
	Disk     *uint64  `yaml:"disk,omitempty" json:"disk,omitempty"`
	MaxMem   *uint64  `yaml:"maxmem,omitempty" json:"maxmem,omitempty"`
	Mem      *uint64  `yaml:"mem,omitempty" json:"mem,omitempty"`
	Uptime   *uint64  `yaml:"uptime,omitempty" json:"uptime,omitempty"`
	Running  *int     `yaml:"running,omitempty" json:"running,omitempty"`
}

// QemuVMFromClusterResource maps a qemu row from GET /cluster/resources.
func QemuVMFromClusterResource(r ClusterResource) (QemuVM, bool) {
	if r.Type != "qemu" {
		return QemuVM{}, false
	}
	vmid := 0
	if r.VMID != nil {
		vmid = *r.VMID
	}
	q := QemuVM{
		ID:       r.ID,
		Fluid:    r.Fluid,
		VMID:     vmid,
		Node:     r.Node,
		Name:     r.Name,
		Status:   r.Status,
		Pool:     r.Pool,
		Template: r.Template,
		MaxCPU:   r.MaxCPU,
		CPU:      r.CPU,
		MaxDisk:  r.MaxDisk,
		Disk:     r.Disk,
		MaxMem:   r.MaxMem,
		Mem:      r.Mem,
		Uptime:   r.Uptime,
		Running:  r.Running,
	}
	if q.ID == "" {
		q.ID = fmt.Sprintf("qemu/%s/%d", r.Node, vmid)
		q.Fluid = CalculateFluid(q.ID)
	}
	return q, true
}

// AccessUser is a Proxmox access user (no secrets).
type AccessUser struct {
	UserID  string   `yaml:"userid" json:"userid"`
	Fluid   string   `yaml:"fluid" json:"fluid"`
	Email   string   `yaml:"email,omitempty" json:"email,omitempty"`
	Enable  *int     `yaml:"enable,omitempty" json:"enable,omitempty"`
	Realm   string   `yaml:"realm,omitempty" json:"realm,omitempty"`
	Groups  []string `yaml:"groups,omitempty" json:"groups,omitempty"`
	Comment string   `yaml:"comment,omitempty" json:"comment,omitempty"`
}
