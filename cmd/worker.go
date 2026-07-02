package cmd

import (
	"product-service/internal/app"

	"github.com/spf13/cobra"
)

// go run main.go worker
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "run kafka worker",
	Run: func(cmd *cobra.Command, args []string) {
		app.RunWorker()
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
