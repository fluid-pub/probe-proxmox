module fluid/probes/proxmox

go 1.23

require (
	fluid/probes/core v0.1.0
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/gorilla/websocket v1.5.3 // indirect

replace fluid/probes/core => ./core
