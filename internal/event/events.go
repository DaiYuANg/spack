package event

import "time"

const (
	variantServedEventName  = "variant.served"
	variantRemovedEventName = "variant.removed"
)

type VariantRemovalReason string

const (
	VariantRemovalReasonTTL  VariantRemovalReason = "ttl"
	VariantRemovalReasonSize VariantRemovalReason = "size"
)

type VariantServed struct {
	AssetPath     string
	ArtifactPath  string
	ServedAt      time.Time
	ContentType   string
	ContentCoding string
}

func (VariantServed) Name() string {
	return variantServedEventName
}

type VariantRemoved struct {
	ArtifactPath string
	Reason       VariantRemovalReason
	RemovedAt    time.Time
}

func (VariantRemoved) Name() string {
	return variantRemovedEventName
}
