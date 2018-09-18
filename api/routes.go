package api

func (s *server) Routes() {
	s.router.HandleFunc("/", s.Index(s.Communication))
}
