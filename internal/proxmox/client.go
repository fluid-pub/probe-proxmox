package proxmox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"fluid/probes/proxmox/internal/config"
	"fluid/probes/proxmox/internal/models"
)

// Client calls the Proxmox VE HTTPS API (api2/json).
type Client struct {
	BaseURL               string
	APIToken              string
	Timeout               time.Duration
	MaxRetries            int
	TLSInsecureSkipVerify bool
	httpClient            *http.Client
}

// NewClient builds a Proxmox API client.
func NewClient(cfg *config.ProxmoxConfig) (*Client, error) {
	if cfg.APIURL == "" {
		return nil, fmt.Errorf("API URL is required")
	}
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("API token is required")
	}

	timeout := 30 * time.Second
	if cfg.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout: %w", err)
		}
	}

	maxRetries := 3
	if cfg.MaxRetries > 0 {
		maxRetries = cfg.MaxRetries
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.TLSInsecureSkipVerify {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			}
		} else {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}
	}

	return &Client{
		BaseURL:               strings.TrimSuffix(strings.TrimSpace(cfg.APIURL), "/"),
		APIToken:              cfg.APIToken,
		Timeout:               timeout,
		MaxRetries:            maxRetries,
		TLSInsecureSkipVerify: cfg.TLSInsecureSkipVerify,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

func (c *Client) makeRequest(method, apiPath string) (*http.Response, error) {
	url := fmt.Sprintf("%s/api2/json/%s", c.BaseURL, strings.TrimPrefix(apiPath, "/"))
	log.Printf("Proxmox %s %s", method, url)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// PVEAPIToken=user@realm!token_id=secret
	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.APIToken))

	var resp *http.Response
	var lastErr error
	for i := 0; i < c.MaxRetries; i++ {
		resp, lastErr = c.httpClient.Do(req)
		if lastErr == nil && resp.StatusCode < 500 {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if i < c.MaxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request after %d retries: %w", c.MaxRetries, lastErr)
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// GetClusterResources returns cluster-wide resources (VMs, LXCs, nodes, storage, ...).
func (c *Client) GetClusterResources() ([]models.ClusterResource, error) {
	resp, err := c.makeRequest("GET", "cluster/resources")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var envelope struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	out := make([]models.ClusterResource, 0, len(envelope.Data))
	for _, raw := range envelope.Data {
		r := parseClusterResource(raw)
		out = append(out, r)
	}

	log.Printf("Retrieved %d cluster resources", len(out))
	return out, nil
}

// GetQEMUVms returns QEMU/KVM guests. Uses GET /cluster/resources?type=vm (PVE enum: vm, storage, node, sdn — not "qemu"), then keeps rows with type "qemu" only (excludes lxc).
func (c *Client) GetQEMUVms() ([]models.QemuVM, error) {
	resp, err := c.makeRequest("GET", "cluster/resources?type=vm")
	if err != nil {
		log.Printf("cluster/resources?type=vm unavailable (%v), using full cluster/resources and filtering", err)
		return c.getQEMUVmsFromFullList()
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	vms, err := decodeQemuVMsFromClusterResourceJSON(bodyBytes)
	if err != nil {
		log.Printf("decode qemu filter response failed (%v), falling back to full list", err)
		return c.getQEMUVmsFromFullList()
	}

	log.Printf("Retrieved %d QEMU VMs", len(vms))
	return vms, nil
}

func (c *Client) getQEMUVmsFromFullList() ([]models.QemuVM, error) {
	all, err := c.GetClusterResources()
	if err != nil {
		return nil, err
	}
	out := make([]models.QemuVM, 0)
	for _, r := range all {
		if q, ok := models.QemuVMFromClusterResource(r); ok {
			out = append(out, q)
		}
	}
	log.Printf("Retrieved %d QEMU VMs (from full cluster resources)", len(out))
	return out, nil
}

func decodeQemuVMsFromClusterResourceJSON(bodyBytes []byte) ([]models.QemuVM, error) {
	var envelope struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return nil, err
	}
	out := make([]models.QemuVM, 0, len(envelope.Data))
	for _, raw := range envelope.Data {
		r := parseClusterResource(raw)
		if q, ok := models.QemuVMFromClusterResource(r); ok {
			out = append(out, q)
		}
	}
	return out, nil
}

// GetAccessUsers lists Proxmox access users (GET /access/users). Tries full=1 first, then plain list.
func (c *Client) GetAccessUsers() ([]models.AccessUser, error) {
	resp, err := c.makeRequest("GET", "access/users?full=1")
	if err != nil {
		log.Printf("access/users?full=1 failed (%v), retrying without full", err)
		resp, err = c.makeRequest("GET", "access/users")
		if err != nil {
			return nil, fmt.Errorf("access users: %w", err)
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	users, err := parseAccessUsersJSON(bodyBytes)
	if err != nil {
		return nil, err
	}

	log.Printf("Retrieved %d access users", len(users))
	return users, nil
}

func parseAccessUsersJSON(bodyBytes []byte) ([]models.AccessUser, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return nil, fmt.Errorf("parse access users envelope: %w", err)
	}

	var strs []string
	if err := json.Unmarshal(envelope.Data, &strs); err == nil {
		out := make([]models.AccessUser, 0, len(strs))
		for _, u := range strs {
			if u == "" {
				continue
			}
			out = append(out, models.AccessUser{
				UserID: u,
				Fluid:  models.CalculateFluid(u),
			})
		}
		return out, nil
	}

	var maps []map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &maps); err == nil {
		return accessUsersFromMaps(maps), nil
	}

	var single map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &single); err == nil && single != nil {
		return accessUsersFromMaps([]map[string]interface{}{single}), nil
	}

	return nil, fmt.Errorf("unexpected access/users data shape")
}

func accessUsersFromMaps(maps []map[string]interface{}) []models.AccessUser {
	out := make([]models.AccessUser, 0, len(maps))
	for _, m := range maps {
		userid, _ := m["userid"].(string)
		if userid == "" {
			continue
		}
		u := models.AccessUser{
			UserID: userid,
			Fluid:  models.CalculateFluid(userid),
		}
		if email, ok := m["email"].(string); ok {
			u.Email = email
		}
		if enable, ok := m["enable"].(float64); ok {
			e := int(enable)
			u.Enable = &e
		}
		if realm, ok := m["realm"].(string); ok {
			u.Realm = realm
		}
		if comment, ok := m["comment"].(string); ok {
			u.Comment = comment
		}
		if g, ok := m["groups"].([]interface{}); ok {
			for _, x := range g {
				if s, ok := x.(string); ok {
					u.Groups = append(u.Groups, s)
				}
			}
		}
		out = append(out, u)
	}
	return out
}

func parseClusterResource(m map[string]interface{}) models.ClusterResource {
	r := models.ClusterResource{}

	if v, ok := m["id"].(string); ok {
		r.ID = v
	}
	if v, ok := m["type"].(string); ok {
		r.Type = v
	}
	if v, ok := m["node"].(string); ok {
		r.Node = v
	}
	if v, ok := m["name"].(string); ok {
		r.Name = v
	}
	if v, ok := m["status"].(string); ok {
		r.Status = v
	}
	if v, ok := m["pool"].(string); ok {
		r.Pool = v
	}
	if v, ok := m["storage"].(string); ok {
		r.Storage = v
	}

	r.VMID = intOrNil(m["vmid"])
	r.Template = intOrNil(m["template"])
	r.Running = intOrNil(m["running"])
	r.CgroupMode = intOrNil(m["cgroup_mode"])

	r.MaxCPU = floatOrNil(m["maxcpu"])
	r.CPU = floatOrNil(m["cpu"])

	r.MaxDisk = uint64OrNil(m["maxdisk"])
	r.Disk = uint64OrNil(m["disk"])
	r.MaxMem = uint64OrNil(m["maxmem"])
	r.Mem = uint64OrNil(m["mem"])
	r.Uptime = uint64OrNil(m["uptime"])

	if r.ID == "" {
		r.ID = fallbackID(r)
	}
	r.Fluid = models.CalculateFluid(r.ID)

	return r
}

func fallbackID(r models.ClusterResource) string {
	if r.VMID != nil {
		return fmt.Sprintf("%s/%s/%d", r.Type, r.Node, *r.VMID)
	}
	if r.Storage != "" && r.Node != "" {
		return fmt.Sprintf("%s/%s/%s", r.Type, r.Node, r.Storage)
	}
	return fmt.Sprintf("%s/%s", r.Type, r.Node)
}

func intOrNil(v interface{}) *int {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case float64:
		i := int(n)
		return &i
	case int:
		return &n
	default:
		return nil
	}
}

func floatOrNil(v interface{}) *float64 {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case float64:
		return &n
	default:
		return nil
	}
}

func uint64OrNil(v interface{}) *uint64 {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case float64:
		u := uint64(n)
		return &u
	default:
		return nil
	}
}
