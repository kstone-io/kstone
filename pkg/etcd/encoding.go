/*
Copyright 2017 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	klog "k8s.io/klog/v2"
	kubectlScheme "k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/yaml"
)

const (
	StorageBinaryMediaType = "application/vnd.kubernetes.storagebinary"
	ProtobufMediaType      = "application/vnd.kubernetes.protobuf"
	YamlMediaType          = "application/yaml"
	JSONMediaType          = "application/json"

	ProtobufShortname = "proto"
	YamlShortname     = "yaml"
	JSONShortname     = "json"
)

// ProtoEncodingPrefix ... see k8s.io/apimachinery/pkg/runtime/serializer/protobuf.go
var ProtoEncodingPrefix = []byte{0x6b, 0x38, 0x73, 0x00}

var MediaTypeList = []string{JSONMediaType, YamlMediaType}

var MediaTypeMap = map[string]string{
	"raw":                  StorageBinaryMediaType,
	ProtobufShortname:      ProtobufMediaType,
	YamlShortname:          YamlMediaType,
	JSONShortname:          JSONMediaType,
	StorageBinaryMediaType: "raw",
	ProtobufMediaType:      ProtobufShortname,
	YamlMediaType:          YamlShortname,
	JSONMediaType:          JSONShortname,
}

var Codecs = kubectlScheme.Codecs

// ConvertToData converts content input to data with inMediaType
func ConvertToData(inMediaType string, in []byte) (map[string]string, error) {
	data := make(map[string]string)
	for _, outMediaType := range MediaTypeList {
		if inMediaType == StorageBinaryMediaType && outMediaType == ProtobufMediaType {
			data[MediaTypeMap[outMediaType]] = DecodeRawToString(in)
			continue
		}

		if inMediaType == ProtobufMediaType && outMediaType == StorageBinaryMediaType {
			return nil, fmt.Errorf("unsupported conversion: protobuf to kubernetes binary storage representation")
		}

		typeMeta, err := decodeTypeMeta(inMediaType, in)
		if err != nil {
			return nil, err
		}

		var encoded []byte
		if inMediaType == outMediaType {
			// Assumes that the stored version is "correct". Primarily a short cut to allow CRDs to work.
			encoded = in
			if outMediaType == JSONMediaType {
				encoded = append(encoded, '\n')
			}
		} else {
			inCodec, err := newCodec(typeMeta, inMediaType)
			if err != nil {
				return nil, err
			}
			outCodec, err := newCodec(typeMeta, outMediaType)
			if err != nil {
				return nil, err
			}

			obj, err := runtime.Decode(inCodec, in)
			if err != nil {
				return nil, fmt.Errorf("error decoding from %s: %s", inMediaType, err)
			}

			encoded, err = runtime.Encode(outCodec, obj)
			if err != nil {
				return nil, fmt.Errorf("error encoding to %s: %s", outMediaType, err)
			}
		}
		data[MediaTypeMap[outMediaType]] = string(encoded)
	}
	return data, nil
}

// DetectAndExtract searches the start of either json of protobuf data, and, if found, returns the mime type and data.
func DetectAndExtract(in []byte) (string, []byte, error) {
	if pb, ok := tryFindProto(in); ok {
		return StorageBinaryMediaType, pb, nil
	}
	if rawJs, ok := tryFindJSON(in); ok {
		js, err := rawJs.MarshalJSON()
		if err != nil {
			return "", nil, err
		}
		return JSONMediaType, js, nil
	}
	return "", nil, fmt.Errorf("error reading input, does not appear to contain valid JSON or binary data")
}

// tryFindProto searches for the 'k8s\0' prefix, and, if found, returns the data starting with the prefix.
func tryFindProto(in []byte) ([]byte, bool) {
	i := bytes.Index(in, ProtoEncodingPrefix)
	if i >= 0 && i < len(in) {
		return in[i:], true
	}
	return nil, false
}

const jsonStartChars = "{["

// tryFindJSON searches for the start of a valid json substring, and, if found, returns the json.
func tryFindJSON(in []byte) (*json.RawMessage, bool) {
	var js json.RawMessage

	i := bytes.IndexAny(in, jsonStartChars)
	for i >= 0 && i < len(in) {
		in = in[i:]
		if len(in) < 2 {
			break
		}
		err := json.Unmarshal(in, &js)
		if err == nil {
			return &js, true
		}
		in = in[1:]
		i = bytes.IndexAny(in, jsonStartChars)
	}
	return nil, false
}

// DecodeRawToString decodes the raw payload bytes contained within the 'Unknown' protobuf envelope of
// the given storage data.
func DecodeRawToString(in []byte) string {
	unknown, err := DecodeUnknown(in)
	if err != nil {
		return ""
	}

	return string(unknown.Raw)
}

// DecodeUnknown decodes the Unknown protobuf type from the given storage data.
func DecodeUnknown(in []byte) (*runtime.Unknown, error) {
	if len(in) < 4 {
		return nil, fmt.Errorf("input too short, expected 4 byte proto encoding prefix but got %v", in)
	}
	if !bytes.Equal(in[:4], ProtoEncodingPrefix) {
		return nil, fmt.Errorf("first 4 bytes %v, do not match proto encoding prefix of %v", in[:4], ProtoEncodingPrefix)
	}
	data := in[4:]

	unknown := &runtime.Unknown{}
	if err := unknown.Unmarshal(data); err != nil {
		return nil, err
	}
	return unknown, nil
}

// newCodec creates a new kubernetes storage codec for encoding and decoding persisted data.
func newCodec(typeMeta *runtime.TypeMeta, mediaType string) (runtime.Codec, error) {
	// For api machinery purposes, we treat StorageBinaryMediaType as ProtobufMediaType
	if mediaType == StorageBinaryMediaType {
		mediaType = ProtobufMediaType
	}
	mediaTypes := Codecs.SupportedMediaTypes()

	info, ok := runtime.SerializerInfoForMediaType(mediaTypes, mediaType)
	if !ok {
		if len(mediaTypes) == 0 {
			return nil, fmt.Errorf("no serializers registered for %v", mediaTypes)
		}
		info = mediaTypes[0]
	}
	gv, err := schema.ParseGroupVersion(typeMeta.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to parse meta APIVersion '%s': %s", typeMeta.APIVersion, err)
	}
	encoder := Codecs.EncoderForVersion(info.Serializer, gv)
	decoder := Codecs.DecoderToVersion(info.Serializer, gv)
	codec := Codecs.CodecForVersions(encoder, decoder, gv, gv)
	return codec, nil
}

// decodeTypeMeta gets the TypeMeta from the given data, either as JSON or Protobuf.
func decodeTypeMeta(inMediaType string, in []byte) (*runtime.TypeMeta, error) {
	switch inMediaType {
	case JSONMediaType:
		return typeMetaFromJSON(in)
	case StorageBinaryMediaType:
		return typeMetaFromBinaryStorage(in)
	case YamlMediaType:
		return typeMetaFromYaml(in)
	default:
		return nil, fmt.Errorf("unsupported inMediaType %s", inMediaType)
	}
}

// typeMetaFromJSON generates type for json
func typeMetaFromJSON(in []byte) (*runtime.TypeMeta, error) {
	var meta runtime.TypeMeta
	err := json.Unmarshal(in, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// typeMetaFromBinaryStorage generates type for binary storage
func typeMetaFromBinaryStorage(in []byte) (*runtime.TypeMeta, error) {
	unknown, err := DecodeUnknown(in)
	if err != nil {
		return nil, err
	}
	return &unknown.TypeMeta, nil
}

// typeMetaFromYaml generates type for yaml
func typeMetaFromYaml(in []byte) (*runtime.TypeMeta, error) {
	var meta runtime.TypeMeta
	err := yaml.Unmarshal(in, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// ConvertToJSON converts kv to json string
func ConvertToJSON(kv *mvccpb.KeyValue) string {
	decoder := kubectlScheme.Codecs.UniversalDeserializer()
	encoder := jsonserializer.NewSerializer(
		jsonserializer.DefaultMetaFactory,
		kubectlScheme.Scheme,
		kubectlScheme.Scheme,
		false,
	)
	objJSON := &bytes.Buffer{}

	obj, _, err := decoder.Decode(kv.Value, nil, nil)
	if err != nil {
		klog.Errorf("WARN: error decoding value %s: %v", string(kv.Value), err)
		return string(kv.Value)
	}
	objJSON.Reset()
	if err := encoder.Encode(obj, objJSON); err != nil {
		klog.Errorf("WARN: error encoding object %#v as JSON: %v", obj, err)
		return string(kv.Value)
	}
	return objJSON.String()
}
