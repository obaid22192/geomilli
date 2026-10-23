package envload

import (
	"os"

	"github.com/joho/godotenv"
)

// Load reads .env automatically when present. It is safe to call multiple times.
func Load() {
	if custom := os.Getenv("ENV_FILE"); custom != "" {
		_ = godotenv.Overload(custom)
		return
	}
	_ = godotenv.Overload()
}
