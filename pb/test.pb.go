// Code generated by protoc-gen-go. DO NOT EDIT.
// source: test.proto

/*
Package pb is a generated protocol buffer package.

It is generated from these files:
	test.proto

It has these top-level messages:
	IP
	URL
*/
package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// @inject_beego_orm_table: "ip"
type IP struct {
	// @inject_tag: orm:"address"
	Address string `protobuf:"bytes,1,opt,name=Address" json:"Address,omitempty" orm:"address"`
}

func (m *IP) Reset()                    { *m = IP{} }
func (m *IP) String() string            { return proto.CompactTextString(m) }
func (*IP) ProtoMessage()               {}
func (*IP) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *IP) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

// @inject_beego_orm_table: "url"
type URL struct {
	// @inject_tag: valid:"http|https"
	Scheme string `protobuf:"bytes,1,opt,name=scheme" json:"scheme,omitempty" valid:"http|https"`
	Url    string `protobuf:"bytes,2,opt,name=url" json:"url,omitempty"`
	// @inject_tag: valid:"nonzero"
	Port int32 `protobuf:"varint,3,opt,name=port" json:"port,omitempty" valid:"nonzero"`
}

func (m *URL) Reset()                    { *m = URL{} }
func (m *URL) String() string            { return proto.CompactTextString(m) }
func (*URL) ProtoMessage()               {}
func (*URL) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *URL) GetScheme() string {
	if m != nil {
		return m.Scheme
	}
	return ""
}

func (m *URL) GetUrl() string {
	if m != nil {
		return m.Url
	}
	return ""
}

func (m *URL) GetPort() int32 {
	if m != nil {
		return m.Port
	}
	return 0
}

func init() {
	proto.RegisterType((*IP)(nil), "pb.IP")
	proto.RegisterType((*URL)(nil), "pb.URL")
}

func init() { proto.RegisterFile("test.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 121 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2a, 0x49, 0x2d, 0x2e,
	0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x2a, 0x48, 0x52, 0x92, 0xe3, 0x62, 0xf2, 0x0c,
	0x10, 0x92, 0xe0, 0x62, 0x77, 0x4c, 0x49, 0x29, 0x4a, 0x2d, 0x2e, 0x96, 0x60, 0x54, 0x60, 0xd4,
	0xe0, 0x0c, 0x82, 0x71, 0x95, 0x9c, 0xb9, 0x98, 0x43, 0x83, 0x7c, 0x84, 0xc4, 0xb8, 0xd8, 0x8a,
	0x93, 0x33, 0x52, 0x73, 0x53, 0xa1, 0xf2, 0x50, 0x9e, 0x90, 0x00, 0x17, 0x73, 0x69, 0x51, 0x8e,
	0x04, 0x13, 0x58, 0x10, 0xc4, 0x14, 0x12, 0xe2, 0x62, 0x29, 0xc8, 0x2f, 0x2a, 0x91, 0x60, 0x56,
	0x60, 0xd4, 0x60, 0x0d, 0x02, 0xb3, 0x93, 0xd8, 0xc0, 0xf6, 0x19, 0x03, 0x02, 0x00, 0x00, 0xff,
	0xff, 0x71, 0xa9, 0x7b, 0x8b, 0x7d, 0x00, 0x00, 0x00,
}

func (_ *IP) TableName() string {
    return "ip"
}

func (_ *URL) TableName() string {
    return "url"
}
