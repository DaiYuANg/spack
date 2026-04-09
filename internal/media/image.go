// Package media centralizes media-type and image-format normalization helpers.
package media

import (
	"slices"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/lo"
)

type ImageFormatDescriptor struct {
	Name         string
	MediaType    string
	Extension    string
	AcceptTokens collectionx.List[string]
}

var imageFormatDescriptors = []ImageFormatDescriptor{
	{
		Name:         "jpeg",
		MediaType:    "image/jpeg",
		Extension:    ".jpg",
		AcceptTokens: collectionx.NewList("image/jpeg", "image/jpg"),
	},
	{
		Name:         "png",
		MediaType:    "image/png",
		Extension:    ".png",
		AcceptTokens: collectionx.NewList("image/png"),
	},
}

func SupportedImageFormats() collectionx.List[string] {
	return collectionx.NewList(lo.Map(imageFormatDescriptors, func(descriptor ImageFormatDescriptor, _ int) string {
		return descriptor.Name
	})...)
}

func LookupImageDescriptor(format string) (ImageFormatDescriptor, bool) {
	normalized := strings.ToLower(strings.TrimSpace(format))
	return findImageDescriptor(func(descriptor ImageFormatDescriptor) bool {
		return descriptor.Name == normalized
	})
}

func LookupImageDescriptorByMediaType(mediaType string) (ImageFormatDescriptor, bool) {
	normalized := strings.ToLower(strings.TrimSpace(mediaType))
	return findImageDescriptor(func(descriptor ImageFormatDescriptor) bool {
		return descriptor.MediaType == normalized
	})
}

func LookupImageDescriptorByAcceptToken(token string) (ImageFormatDescriptor, bool) {
	normalized := strings.ToLower(strings.TrimSpace(token))
	return findImageDescriptor(func(descriptor ImageFormatDescriptor) bool {
		return slices.Contains(descriptor.AcceptTokens.Values(), normalized)
	})
}

func NormalizeImageFormat(format string) string {
	normalized := strings.ToLower(strings.TrimSpace(format))
	switch normalized {
	case "jpg":
		return "jpeg"
	default:
		if descriptor, ok := LookupImageDescriptor(normalized); ok {
			return descriptor.Name
		}
		return ""
	}
}

func ImageFormat(mediaType string) string {
	if descriptor, ok := LookupImageDescriptorByMediaType(mediaType); ok {
		return descriptor.Name
	}
	return ""
}

func IsImageMediaType(mediaType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mediaType)), "image/")
}

func NormalizeImageFormats(formats collectionx.List[string]) collectionx.List[string] {
	if formats.IsEmpty() {
		return collectionx.NewList[string]()
	}

	normalized := collectionx.FilterMapList(formats, func(_ int, format string) (string, bool) {
		value := NormalizeImageFormat(format)
		return value, value != ""
	})
	return collectionx.NewList(collectionx.NewOrderedSet(normalized.Values()...).Values()...)
}

func findImageDescriptor(match func(ImageFormatDescriptor) bool) (ImageFormatDescriptor, bool) {
	return lo.Find(imageFormatDescriptors, match)
}
