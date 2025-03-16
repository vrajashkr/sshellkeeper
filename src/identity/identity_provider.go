package identity

type IdentityProvider interface {
	IsValidUser(username string) (bool, error)
}
