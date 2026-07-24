package server

import "time"

// storedToken is the JSON the agent persists for a client ID in its encrypted token
// store. It is the agent's own storage schema, not a wire format: a refresh token lives
// here and nowhere else, since tokenResponse hands the client only the access token and
// its expiration. The SDK's api.AccessToken describes that client-facing token and
// deliberately has no refresh token fields.
//
// The field names and JSON tags must stay as they are: token files written by earlier
// agents are read back through this type.
type storedToken struct {
	AccessToken                string    `json:"access_token"`
	ExpirationDate             time.Time `json:"expiration_date"`
	RefreshToken               string    `json:"refresh_token"`
	RefreshTokenExpirationDate time.Time `json:"refresh_token_expiration_date"`
}
