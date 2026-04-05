// Package media centralizes media-type and image-format normalization helpers.
package media

import (
	"slices"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/anthonynsimon/bild/imgio"
	"github.com/samber/lo"
)

type ImageFormatCapability struct {
	Name         string
	MediaType    string
	Extension    string
	AcceptTokens collectionx.List[string]
	Encoder      func(quality int) imgio.Encoder
}

var imageFormatCapabilities = []ImageFormatCapability{
	{
		Name:         "jpeg",
		MediaType:    "image/jpeg",
		Extension:    ".jpg",
		AcceptTokens: collectionx.NewList("image/jpeg", "image/jpg"),
		Encoder:      imgio.JPEGEncoder,
	},
	{
		Name:         "png",
		MediaType:    "image/png",
		Extension:    ".png",
		AcceptTokens: collectionx.NewList("image/png"),
		Encoder: func(_ int) imgio.Encoder {
			return imgio.PNGEncoder()
		},
	},
}

func SupportedImageFormats() collectionx.List[string] {
	return collectionx.NewList(lo.Map(imageFormatCapabilities, func(capability ImageFormatCapability, _ int) string {
		return capability.Name
	})...)
}

func LookupImageCapability(format string) (ImageFormatCapability, bool) {
	normalized := strings.ToLower(strings.TrimSpace(format))
	return findImageCapability(func(capability ImageFormatCapability) bool {
		return capability.Name == normalized
	})
}

func LookupImageCapabilityByMediaType(mediaType string) (ImageFormatCapability, bool) {
	normalized := strings.ToLower(strings.TrimSpace(mediaType))
	return findImageCapability(func(capability ImageFormatCapability) bool {
		return capability.MediaType == normalized
	})
}

func LookupImageCapabilityByAcceptToken(token string) (ImageFormatCapability, bool) {
	normalized := strings.ToLower(strings.TrimSpace(token))
	return findImageCapability(func(capability ImageFormatCapability) bool {
		return slices.Contains(capability.AcceptTokens.Values(), normalized)
	})
}

func NormalizeImageFormat(format string) string {
	normalized := strings.ToLower(strings.TrimSpace(format))
	switch normalized {
	case "jpg":
		return "jpeg"
	default:
		if capability, ok := LookupImageCapability(normalized); ok {
			return capability.Name
		}
		return ""
	}
}

func ImageFormat(mediaType string) string {
	if capability, ok := LookupImageCapabilityByMediaType(mediaType); ok {
		return capability.Name
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

func findImageCapability(match func(ImageFormatCapability) bool) (ImageFormatCapability, bool) {
	return lo.Find(imageFormatCapabilities, match)
}
