package pipeline

import (
	"cmp"
	"slices"

	"github.com/samber/lo"
)

type stageRegistration struct {
	Order int
	Stage Stage
}

func newStageRegistration(order int, stage Stage) stageRegistration {
	return stageRegistration{
		Order: order,
		Stage: stage,
	}
}

func buildStages(registrations []stageRegistration) []Stage {
	if len(registrations) == 0 {
		return nil
	}

	sorted := append([]stageRegistration(nil), registrations...)
	slices.SortFunc(sorted, func(left, right stageRegistration) int {
		if left.Order != right.Order {
			return cmp.Compare(left.Order, right.Order)
		}
		switch {
		case left.Stage == nil && right.Stage == nil:
			return 0
		case left.Stage == nil:
			return 1
		case right.Stage == nil:
			return -1
		default:
			return compareStageNames(left.Stage, right.Stage)
		}
	})

	return lo.FilterMap(sorted, func(registration stageRegistration, _ int) (Stage, bool) {
		return registration.Stage, registration.Stage != nil
	})
}

func compareStageNames(left, right Stage) int {
	return cmp.Compare(left.Name(), right.Name())
}
