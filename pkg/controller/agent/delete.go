package agent

import (
	"errors"
	"fmt"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
)

// handleDelete removes the token cached under the request's client ID. Deleting a
// client ID with no cached token succeeds (it is a no-op), so the revoke flow does
// not have to special-case a missing token.
func (c *Controller) handleDelete(req *agentapi.Request) *agentapi.Response {
	st := c.tokenStore()
	if st == nil {
		return &agentapi.Response{Error: agentapi.RespLocked}
	}
	if err := st.Delete(req.ClientID); err != nil {
		if errors.Is(err, tokenstore.ErrInvalidClientID) {
			return &agentapi.Response{Error: errMsgInvalidClientID}
		}
		return &agentapi.Response{Error: fmt.Sprintf("%s: %s", errMsgDelete, err)}
	}
	return &agentapi.Response{OK: true}
}
