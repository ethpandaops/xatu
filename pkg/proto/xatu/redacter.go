package xatu

import (
	"strings"

	"google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

type Redacter interface {
	Apply(msg proto.Message) []string
}

type RedacterConfig struct {
	FieldPaths []string `yaml:"fieldPaths"`
}

func (r *RedacterConfig) Validate() error {
	return nil
}

type redacter struct {
	config *RedacterConfig
}

func NewRedacter(config *RedacterConfig) (Redacter, error) {
	return &redacter{config: config}, nil
}

func (r *redacter) Apply(msg proto.Message) []string {
	removedFields := []string{}

	for _, fieldPath := range r.config.FieldPaths {
		if r.setFieldToEmpty(msg, fieldPath) {
			removedFields = append(removedFields, fieldPath)
		}
	}

	return removedFields
}

func (r *redacter) setFieldToEmpty(msg proto.Message, fieldPath string) bool {
	fields := strings.Split(fieldPath, ".")

	md := msg.ProtoReflect()

	for i, fieldName := range fields {
		fd := md.Descriptor().Fields().ByName(protoreflect.Name(fieldName))
		if fd == nil {
			// Ignore fields that don't exist
			return false
		}

		if i == len(fields)-1 {
			// Last field, set to zero value
			md.Clear(fd)

			return true
		} else {
			// Navigate to next message
			if md.Get(fd).Message().IsValid() {
				md = md.Get(fd).Message()
			} else {
				// If the field is not set, create a new message
				md = md.Mutable(fd).Message()
			}
		}
	}

	return false
}
