package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/server"
)

var (
	version   = "dev"
	cfgFile   string
	rootCmd   = &cobra.Command{
		Use:   "rpmmanager",
		Short: "RPM Manager - Graphical RPM Package Management",
	}
	serveCmd  = &cobra.Command{
		Use:   "serve",
		Short: "Start the RPM Manager server",
		RunE:  runServe,
	}
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("rpmmanager", version)
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: config.yaml)")
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("init server: %w", err)
	}

	return srv.Run()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
