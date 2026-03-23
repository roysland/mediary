//go:build !e2e

package server

func (s *Server) registerE2ERoutes() {}

func isPublicE2ERoute(string) bool {
	return false
}
