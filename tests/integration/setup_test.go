package integration

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	// Try to load .env from project root (relative to tests/integration)
	_ = godotenv.Load("../../.env")

	// If .env doesn't exist, tests will rely on environment variables
	// Tests skip automatically if OPENAI_API_KEY is not set

	os.Exit(m.Run())
}
