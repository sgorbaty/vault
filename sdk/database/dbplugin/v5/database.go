package dbplugin

import (
	"context"
	"time"
)

// Database to manipulate users within an external system (typically a database).
type Database interface {
	// Initialize the database plugin. This is the equivalent of a constructor for the
	// database object itself.
	Initialize(ctx context.Context, req InitializeRequest) (InitializeResponse, error)

	// NewUser creates a new user within the database. This user is temporary in that it
	// will exist until the TTL expires.
	NewUser(ctx context.Context, req NewUserRequest) (NewUserResponse, error)

	// UpdateUser updates an existing user within the database.
	UpdateUser(ctx context.Context, req UpdateUserRequest) (UpdateUserResponse, error)

	// DeleteUser from the database. This should not error if the user didn't
	// exist prior to this call.
	DeleteUser(ctx context.Context, req DeleteUserRequest) (DeleteUserResponse, error)

	// Type returns the Name for the particular database backend implementation.
	// This type name is usually set as a constant within the database backend
	// implementation, e.g. "mysql" for the MySQL database backend. This is used
	// for things like metrics and logging. No behavior is switched on this.
	Type() (string, error)

	// Close attempts to close the underlying database connection that was
	// established by the backend.
	Close() error
}

// ///////////////////////////////////////////////////////////////////////////
// Database Request & Response Objects
// These request and response objects are *not* protobuf types because gRPC does not
// support all types that we need in a nice way. For instance, gRPC does not support
// map[string]interface{}. It does have an `Any` type, but converting it to a map
// requires extensive use of reflection and knowing what types to support ahead of
// time. Instead these types are made as user-friendly as possible so the conversion
// between protobuf types and request/response objects is handled by Vault developers
// rather than needing to be handled by external plugin developers.
// ///////////////////////////////////////////////////////////////////////////

// ///////////////////////////////////////////////////////
// Initialize()
// ///////////////////////////////////////////////////////

// InitializeRequest contains all information needed to initialize a database plugin.
type InitializeRequest struct {
	// Config to initialize the database with. This can include things like connection details,
	// a "root" username & password, etc. This will not include all configuration items specified
	// when configuring the database. Some values will be stripped out by the database engine
	// prior to being passed to the plugin.
	Config map[string]interface{}

	// VerifyConnection during initialization. If true, a connection should be made to the
	// database to verify the connection can be made. If false, no connection should be made
	// on initialization.
	VerifyConnection bool
}

// InitializeResponse returns any information Vault needs to know after initializing
// a database plugin.
type InitializeResponse struct {
	// Config that should be saved in Vault. This may differ from the config in the request,
	// but should contain everything required to Initialize the database.
	// REQUIRED in order to save the configuration into Vault after initialization
	Config map[string]interface{}
}

// ///////////////////////////////////////////////////////
// NewUser()
// ///////////////////////////////////////////////////////

// NewUserRequest request a new user is created
type NewUserRequest struct {
	// UsernameConfig is metadata that can be used to generate a username
	// within the database plugin
	UsernameConfig UsernameMetadata

	// Statements is an ordered list of commands to run within the database when
	// creating a new user. This frequently includes permissions to give the
	// user or similar actions.
	Statements Statements

	// RollbackStatements is an ordered list of commands to run within the database
	// if the new user creation process fails.
	RollbackStatements Statements

	// CredentialType is the type of credential to use when creating a new user.
	// Respective fields for the credential type will contain the credential value.
	CredentialType CredentialType

	// Password credentials to use when creating the user
	// Value is set when the credential type is 'password'
	Password string

	// PublicKey credentials to use when creating the user
	// Value is set when the credential type is 'rsa_2048_private_key'.
	PublicKey string

	// TODO: could we put the DN here? or even the certificate?

	// Expiration of the user. Not all database plugins will support this.
	Expiration time.Time
}

// UsernameMetadata is metadata the database plugin can use to generate a username
type UsernameMetadata struct {
	// DisplayName is the display name for the user.
	// This is set for all CredentialType values.
	DisplayName string

	// RoleName is the name of the role used to create the user.
	// This is set for all CredentialType values.
	RoleName string

	// Subject is the distinguished name for the client certificate credential.
	// This is set when the CredentialType is 'client_certificate'.
	Subject string
}

// NewUserResponse returns any information Vault needs to know after creating a new user.
type NewUserResponse struct {
	// Username of the user created within the database.
	// REQUIRED so Vault knows the name of the user that was created
	Username string
}

// CredentialType is a type of database credential.
type CredentialType int

const (
	CredentialTypePassword CredentialType = iota
	CredentialTypeRSA2048PrivateKey
	CredentialTypeClientCertificate
)

func (k CredentialType) String() string {
	switch k {
	case CredentialTypePassword:
		return "password"
	case CredentialTypeRSA2048PrivateKey:
		return "rsa_2048_private_key"
	case CredentialTypeClientCertificate:
		return "client_certificate"
	default:
		return "unknown"
	}
}

// ///////////////////////////////////////////////////////
// UpdateUser()
// ///////////////////////////////////////////////////////

type UpdateUserRequest struct {
	// Username to make changes to.
	Username string

	// CredentialType is the type of credential to use when creating a new user.
	// Respective fields for the credential type will contain the credential value.
	CredentialType CredentialType

	// Password indicates the new password to change to.
	// If nil, no change is requested.
	Password *ChangePassword

	// PublicKey indicates the new public key to change to.
	// If nil, no change is requested.
	PublicKey *ChangePublicKey

	// Expiration indicates the new expiration date to change to.
	// If nil, no change is requested.
	Expiration *ChangeExpiration
}

// ChangePublicKey of a given user
type ChangePublicKey struct {
	// NewPublicKey is the new public key credential for the user
	NewPublicKey string

	// Statements is an ordered list of commands to run within the database
	// when changing the user's public key credential.
	Statements Statements
}

// ChangePassword of a given user
type ChangePassword struct {
	// NewPassword for the user
	NewPassword string

	// Statements is an ordered list of commands to run within the database
	// when changing the user's password.
	Statements Statements
}

// ChangeExpiration of a given user
type ChangeExpiration struct {
	// NewExpiration of the user
	NewExpiration time.Time

	// Statements is an ordered list of commands to run within the database
	// when changing the user's expiration.
	Statements Statements
}

type UpdateUserResponse struct{}

// ///////////////////////////////////////////////////////
// DeleteUser()
// ///////////////////////////////////////////////////////

type DeleteUserRequest struct {
	// Username to delete from the database
	Username string

	// Statements is an ordered list of commands to run within the database
	// when deleting a user.
	Statements Statements
}

type DeleteUserResponse struct{}

// ///////////////////////////////////////////////////////
// Used across multiple functions
// ///////////////////////////////////////////////////////

// Statements wraps a collection of statements to run in a database when an
// operation is performed (create, update, etc.). This is a struct rather than
// a string slice so we can easily add more information to this in the future.
type Statements struct {
	// Commands is an ordered list of commands to execute in the database.
	// These commands may include templated fields such as {{username}} and {{password}}
	Commands []string
}
