package config

import (
	"fmt"

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
)

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
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

	// Output flags
	rootCmd.PersistentFlags().String("output", "output.db", "SQLite output file")

	// Migrate command specific flags
	migrateCmd.Flags().String("sqlite", "output.db", "SQLite output file")
	migrateCmd.Flags().Bool("with-data", false, "Include data in migration")

	// Add commands
	rootCmd.AddCommand(migrateCmd)

	// Bind all flags to viper
	viper.BindPFlags(rootCmd.PersistentFlags())
	viper.BindPFlags(migrateCmd.Flags())
}

func initConfig() {

	viper.SetEnvPrefix("PG2LITE")
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
		OutputFile:       viper.GetString("output"),
	}
}

func runMigrate(cmd *cobra.Command, args []string) error {
	config := getConfig()
	sqliteFile := config.OutputFile
	withData := viper.GetBool("with-data")

	migrator, err := database.NewMigrator(config)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	return migrator.Migrate(sqliteFile, withData)
}
