package main

import (
	"os"

	"github.com/partir/core/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the health of the Partir environment",
	Run: func(cmd *cobra.Command, args []string) {
		report := doctor.RunDiagnostics()
		report.Print()

		allOk := true
		for _, res := range report.Checks {
			if res.Status == "fail" {
				allOk = false
			}
		}

		if !allOk {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
