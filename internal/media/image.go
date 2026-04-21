// Package media centralizes media-type and image-format normalization helpers.
package media

import (
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

var (
	imageFormatDescriptors = collectionx.NewList[ImageFormatDescriptor](
		ImageFormatDescriptor{
			Name:         "jpeg",
			MediaType:    "image/jpeg",
			Extension:    ".jpg",
			AcceptTokens: collectionx.NewList[string]("image/jpeg", "image/jpg"),
		},
		ImageFormatDescriptor{
			Name:         "png",
			MediaType:    "image/png",
			Extension:    ".png",
			AcceptTokens: collectionx.NewList[string]("image/png"),
		},
	)
	imageDescriptorsByName = collectionx.AssociateList[ImageFormatDescriptor, string, ImageFormatDescriptor](
		imageFormatDescriptors,
		func(_ int, descriptor ImageFormatDescriptor) (string, ImageFormatDescriptor) {
			return descriptor.Name, descriptor
		},
	)
	imageDescriptorsByMediaType = collectionx.AssociateList[ImageFormatDescriptor, string, ImageFormatDescriptor](
		imageFormatDescriptors,
		func(_ int, descriptor ImageFormatDescriptor) (string, ImageFormatDescriptor) {
			return descriptor.MediaType, descriptor
		},
	)
	imageDescriptorsByAcceptToken = buildImageDescriptorsByAcceptToken(imageFormatDescriptors)
)

func SupportedImageFormats() collectionx.List[string] {
	return collectionx.MapList[ImageFormatDescriptor, string](imageFormatDescriptors, func(_ int, descriptor ImageFormatDescriptor) string {
		return descriptor.Name
	})
}

func LookupImageDescriptor(format string) (ImageFormatDescriptor, bool) {
	normalized := strings.ToLower(strings.TrimSpace(format))
	return imageDescriptorsByName.Get(normalized)
}

func LookupImageDescriptorByMediaType(mediaType string) (ImageFormatDescriptor, bool) {
	normalized := strings.ToLower(strings.TrimSpace(mediaType))
	return imageDescriptorsByMediaType.Get(normalized)
}

func LookupImageDescriptorByAcceptToken(token string) (ImageFormatDescriptor, bool) {
	normalized := strings.ToLower(strings.TrimSpace(token))
	return imageDescriptorsByAcceptToken.Get(normalized)
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
	if formats == nil || formats.IsEmpty() {
		return nil
	}

	normalized := collectionx.FilterMapList[string, string](formats, func(_ int, format string) (string, bool) {
		value := NormalizeImageFormat(format)
		return value, value != ""
	})
	return collectionx.NewList[string](lo.Uniq[string](normalized.Values())...)
}

func buildImageDescriptorsByAcceptToken(
	descriptors collectionx.List[ImageFormatDescriptor],
) collectionx.Map[string, ImageFormatDescriptor] {
	out := collectionx.NewMap[string, ImageFormatDescriptor]()
	descriptors.Range(func(_ int, descriptor ImageFormatDescriptor) bool {
		descriptor.AcceptTokens.Range(func(_ int, token string) bool {
			normalized := strings.ToLower(strings.TrimSpace(token))
			if normalized != "" {
				out.Set(normalized, descriptor)
			}
			return true
		})
		return true
	})
	return out
}
