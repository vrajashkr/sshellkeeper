package main

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
	"github.com/vrajashkr/sshellkeeper/src/adapters"
	"github.com/vrajashkr/sshellkeeper/src/game"
	"github.com/vrajashkr/sshellkeeper/src/ldaptools"
	"github.com/vrajashkr/sshellkeeper/src/sshserver"
)

func loadConfig() error {
	viper.SetConfigName("appconfig")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	return err
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	slog.Info("starting sshellkeeper server")

	err := loadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		os.Exit(1)
	}

	slog.Info("loaded configuration")

	// Init LDAP connection
	ldapAdminConn, err := ldaptools.NewLdapAdminConnection(
		viper.GetString("ldap.host"),
		viper.GetString("lldap.web_host"),
		viper.GetString("ldap.base_dn"),
		viper.GetString("ldap.admin_dn"),
		viper.GetString("ldap.username"),
		viper.GetString("ldap.password"),
		viper.GetString("lldap.cli_path"),
	)
	if err != nil {
		slog.Error("failed to initialize admin LDAP connection", "error", err.Error())
		os.Exit(1)
	}

	slog.Info("initialized LDAP")

	// Init identity provider
	ldapIdp := adapters.NewLdapIdentityProvider(&ldapAdminConn)

	// Init SSH server
	sshServer, err := sshserver.NewSSHServer(
		&ldapIdp,
		viper.GetString("server.host"),
		viper.GetString("server.port"),
	)
	if err != nil {
		slog.Error("failed to init SSH server", "error", err)
		// FIXME: should also terminate the LDAP connection if failure
		os.Exit(1)
	}

	slog.Info("initialized identity provider")

	// Init GameAdmin
	ga := adapters.NewLdapGameAdmin(&ldapAdminConn)

	slog.Info("initialized game admin")

	// Init GameEngine
	ge := game.NewGameEngine(&ga)

	slog.Info("initialized game engine")

	slog.Info("starting listener")
	// Start listener
	sshServer.Listen(ge.RunPlayerGameFlow)

	// TODO: proper handling for shutting down the connections
}
