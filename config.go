package morgonfikan

import (
	"embed"
	"fmt"

	"github.com/jonsten/morgonfikan/internal/menu"
)

//go:embed config/*.yaml
var configuration embed.FS

func LoadEmbeddedConfig() (*menu.Config, error) {
	scheduleData, err := configuration.ReadFile("config/schedule.yaml")
	if err != nil {
		return nil, fmt.Errorf("read embedded schedule: %w", err)
	}
	closureData, err := configuration.ReadFile("config/closures.yaml")
	if err != nil {
		return nil, fmt.Errorf("read embedded closures: %w", err)
	}
	return menu.Parse(scheduleData, closureData)
}
