package pipeline

import (
	"cmp"
	"slices"

	"github.com/DaiYuANg/arcgo/collectionx"
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

func buildStages(registrations collectionx.List[stageRegistration]) collectionx.List[Stage] {
	if registrations.IsEmpty() {
		return collectionx.NewList[Stage]()
	}

	sorted := registrations.Values()
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

	return collectionx.NewList(lo.FilterMap(sorted, func(registration stageRegistration, _ int) (Stage, bool) {
		return registration.Stage, registration.Stage != nil
	})...)
}

func compareStageNames(left, right Stage) int {
	return cmp.Compare(left.Name(), right.Name())
}
