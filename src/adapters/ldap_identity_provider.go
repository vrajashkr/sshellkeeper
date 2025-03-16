package adapters

import "github.com/vrajashkr/sshellkeeper/src/ldaptools"

type LdapIdentityProvider struct {
	ldapConn *ldaptools.LdapAdminConnection
}

func NewLdapIdentityProvider(ldapConn *ldaptools.LdapAdminConnection) LdapIdentityProvider {
	return LdapIdentityProvider{
		ldapConn: ldapConn,
	}
}

func (lip *LdapIdentityProvider) IsValidUser(username string) (bool, error) {
	_, err := lip.ldapConn.GetUserRecord(username, []string{"uid"})
	if err != nil {
		return false, err
	}

	return true, nil
}
