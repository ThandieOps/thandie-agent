package main

import (
	"github.com/ThandieOps/thandie-agent/internal/logger"
)

func main() {
	defer logger.Close() // Ensure log file is closed on exit
	Execute()
}
