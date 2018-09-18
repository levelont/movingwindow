package api

/* All requests shall have the same handling.
 */
func (s *server) Routes() {
	s.router.HandleFunc("/", s.Index(s.Communication))
}
