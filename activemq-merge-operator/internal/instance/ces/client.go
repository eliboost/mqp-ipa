// Package ces capsules interaction logic with ces
package ces

// IdentityClient defines interface for getting namespace data
type IdentityClient interface {
	IdentityExist(identity string) (exists bool, err error)
}
