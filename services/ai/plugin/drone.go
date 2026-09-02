package plugin

import "fmt"

type DronePlugin struct{}

func (p *DronePlugin) Name() string { return "Terbang" }
func (p *DronePlugin) Trigger(data string) {
	fmt.Printf("   🚁 [PLUGINS - DRONE] RPM Motor naik ke 8000, lepas landas menuju: %s\n", data)
}
