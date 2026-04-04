package pipeline

func (s *Service) updateQueueLengthMetric() {
	if s.metrics != nil {
		s.metrics.QueueLength.Set(float64(len(s.tasks)))
	}
}
