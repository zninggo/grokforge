package api

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/zninggo/grokforge/internal/auth"
	"github.com/zninggo/grokforge/internal/ws"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // single-node admin; reverse-proxy later
}

// WSHandler upgrades authenticated websocket connections.
type WSHandler struct {
	Hub    *ws.Hub
	Tokens *auth.TokenService
}

func (h *WSHandler) Connect(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		authz := c.Request().Header.Get("Authorization")
		if strings.HasPrefix(authz, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
		}
	}
	if token == "" {
		return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "token required")
	}
	if _, err := h.Tokens.ParseAccess(token); err != nil {
		return Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
	}
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	h.Hub.Register(conn)
	return nil
}
