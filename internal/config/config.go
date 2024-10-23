package config

import (
	"fmt"
	"os"

	"github.com/robmartinson/pg2lite/internal/database"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "pg2lite",
		Short: "PostgreSQL to SQLite migration tool",
		Long: `A database migration tool that copies the structure and optionally 
the data from a PostgreSQL database to a SQLite database.`,
	}

	migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrate PostgreSQL database to SQLite",
		Long: `Migrate a PostgreSQL database to SQLite. This will create the database
structure and optionally copy all data to the new SQLite database.`,
		RunE: runMigrate,
	}

	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate database connection and configuration",
		Long: `Test the database connection and configuration settings
without performing any migration.`,
		RunE: runValidate,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("PostgreSQL to SQLite migrator v1.0")
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pgmigrate.yaml)")

	// Database connection flags
	rootCmd.PersistentFlags().String("pg", "", "PostgreSQL connection string (optional)")
	rootCmd.PersistentFlags().String("host", "localhost", "PostgreSQL host")
	rootCmd.PersistentFlags().Int("port", 5432, "PostgreSQL port")
	rootCmd.PersistentFlags().String("db", "", "PostgreSQL database name")
	rootCmd.PersistentFlags().String("user", "", "PostgreSQL user")
	rootCmd.PersistentFlags().String("password", "", "PostgreSQL password")

	// SSH tunnel flags
	rootCmd.PersistentFlags().String("sshkey", "", "Path to SSH private key file")
	rootCmd.PersistentFlags().String("sshuser", "", "SSH user")
	rootCmd.PersistentFlags().String("sshhost", "", "SSH host")
	rootCmd.PersistentFlags().Int("sshport", 22, "SSH port")

	// Migrate command specific flags
	migrateCmd.Flags().String("sqlite", "output.db", "SQLite output file")
	migrateCmd.Flags().Bool("with-data", false, "Include data in migration")

	// Add commands
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)

	// Bind all flags to viper
	viper.BindPFlags(rootCmd.PersistentFlags())
	viper.BindPFlags(migrateCmd.Flags())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".pgmigrate")
	}

	viper.SetEnvPrefix("PGMIGRATE")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func getConfig() database.Config {
	return database.Config{
		ConnectionString: viper.GetString("pg"),
		Host:             viper.GetString("host"),
		Port:             viper.GetInt("port"),
		Database:         viper.GetString("db"),
		User:             viper.GetString("user"),
		Password:         viper.GetString("password"),
		SSHKey:           viper.GetString("sshkey"),
		SSHUser:          viper.GetString("sshuser"),
		SSHHost:          viper.GetString("sshhost"),
		SSHPort:          viper.GetInt("sshport"),
	}
}

func runMigrate(cmd *cobra.Command, args []string) error {
	config := getConfig()
	sqliteFile := viper.GetString("sqlite")
	withData := viper.GetBool("with-data")

	migrator, err := database.NewMigrator(config)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	return migrator.Migrate(sqliteFile, withData)
}

func runValidate(cmd *cobra.Command, args []string) error {
	config := getConfig()

	migrator, err := database.NewMigrator(config)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	fmt.Println("Configuration is valid and database is accessible")
	return nil
}
