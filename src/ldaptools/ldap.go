package ldaptools

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"os/exec"
	"regexp"

	"github.com/go-ldap/ldap/v3"
)

type LdapAdminConnection struct {
	conn          *ldap.Conn
	baseDN        string
	adminDN       string
	adminUsername string
	adminPassword string
	host          string
	webHost       string

	lldapCliPath string
}

func NewLdapAdminConnection(host string, webHost string, baseDN string, adminDN string, username string, password string, lldapCliPath string) (LdapAdminConnection, error) {
	l, err := ldap.DialURL(fmt.Sprintf("ldap://%s", host))
	if err != nil {
		log.Fatal(err)
	}

	// Note: the connection is not deferred as we want to re-use the same connection for all operations
	// to avoid multiple re-connects to the LDAP server

	err = l.Bind(adminDN, password)
	if err != nil {
		return LdapAdminConnection{}, err
	}

	return LdapAdminConnection{
		conn:          l,
		baseDN:        baseDN,
		adminDN:       adminDN,
		adminUsername: username,
		adminPassword: password,
		host:          host,
		webHost:       webHost,
		lldapCliPath:  lldapCliPath,
	}, nil
}

func (lac *LdapAdminConnection) SetUserPassword(userDN string, newPassword string) error {
	slog.Debug("password change requested for user", "user", userDN)
	req := ldap.NewPasswordModifyRequest(userDN, "", newPassword)
	_, err := lac.conn.PasswordModify(req)
	return err
}

func (lac *LdapAdminConnection) GetUserRecord(username string, fields []string) (*ldap.Entry, error) {
	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		lac.baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectclass=person)(uid=%s))", ldap.EscapeFilter(username)),
		fields,
		nil,
	)

	sr, err := lac.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	if len(sr.Entries) != 1 {
		return nil, fmt.Errorf("exactly 1 user expected for given username")
	}

	return sr.Entries[0], err
}

func (lac *LdapAdminConnection) AddUserToGroup(username string, group string) error {
	slog.Debug("adding user to group", "user", username, "group", group)
	command := exec.Command(lac.lldapCliPath, "user", "group", "add", username, group)
	command.Env = []string{
		fmt.Sprintf("LLDAP_HTTPURL=%s", fmt.Sprintf("http://%s", lac.webHost)),
		fmt.Sprintf("LLDAP_USERNAME=%s", lac.adminUsername),
		fmt.Sprintf("LLDAP_PASSWORD=%s", lac.adminPassword),
	}

	var stderr bytes.Buffer

	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		slog.Error("failed to run lldap-cli", "stderr", stderr.String())
	}

	return err
}

func (lac *LdapAdminConnection) TryBindLdap(username string, password string) error {
	l, err := ldap.DialURL(fmt.Sprintf("ldap://%s", lac.host))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := l.Close()
		if err != nil {
			slog.Error("failed to close ldap connection", "error", err.Error())
		}
	}()

	err = l.Bind(username, password)
	return err
}

func ExtractCNfromDN(dn string) string {
	re := regexp.MustCompile("cn=(?P<cn>[^,]*),")
	matches := re.FindStringSubmatch(dn)
	cnIndex := re.SubexpIndex("cn")
	return matches[cnIndex]
}
