package config

import "runtime"

type Async struct {
	Workers int `koanf:"workers"`
}

func (a Async) NormalizedWorkers() int {
	if a.Workers > 0 {
		return a.Workers
	}
	return max(runtime.NumCPU(), 1)
}
