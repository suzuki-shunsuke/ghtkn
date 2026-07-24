package server

import (
	"errors"
	"fmt"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/tokenstore"
)

// handleSet stores the client-minted token under the request's client ID. It exists
// only for legacy (protocol version 0) clients that mint tokens themselves; version-1
// clients never send SET because the server owns the token lifecycle. The stored
// payload is exactly what the client sent, so no refresh token is ever attached this
// way.
//
// SET is rejected for a non-legacy client: for a current client the server owns the
// lifecycle, so accepting a client-pushed token would let it overwrite the
// server-managed one.
func (s *Server) handleSet(req *agentapi.Request, legacy bool) *agentapi.Response {
	if !legacy {
		return &agentapi.Response{Error: errMsgSetNotAllowed}
	}
	st := s.tokenStore()
	if st == nil {
		return &agentapi.Response{Error: agentapi.RespLocked}
	}
	if err := st.Set(req.ClientID, req.Token); err != nil {
		if errors.Is(err, tokenstore.ErrInvalidClientID) {
			return &agentapi.Response{Error: errMsgInvalidClientID}
		}
		return &agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgSet, err)}
	}
	return &agentapi.Response{OK: true}
}
