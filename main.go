package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lixin9311/jobcan/internal/jobcan"
	"github.com/spf13/cobra"
)

var (
	verbose    = false
	username   string
	password   string
	cookieFile string

	client *jobcan.Client
)

var rootCmd = &cobra.Command{
	Use: "jobcan",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if username == "" || password == "" {
			return fmt.Errorf("username & password must be set")
		}
		client = jobcan.NewClient(cookieFile, username, password, verbose)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if err := client.Close(); err != nil {
			log.Fatal(err)
		}
	},
}

var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "toggle the jobcan working status",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		prevStatus, newStatus, err := client.Toggle(ctx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("previous status was %s\n", prevStatus)
		fmt.Printf("current status is %s\n", newStatus)
	},
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "check current jobcan clock in status",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		status, err := client.GetStatus(ctx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("current status is %s\n", status)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "jobcan username/email")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "jobcan password")
	rootCmd.PersistentFlags().StringVarP(&cookieFile, "cookie", "c", "cookies.json", "cookie file")
	rootCmd.AddCommand(toggleCmd, checkCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal()
	}
}
