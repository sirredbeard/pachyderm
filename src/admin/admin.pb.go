// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: admin/admin.proto

package admin

import (
	context "context"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	versionpb "github.com/pachyderm/pachyderm/v2/src/version/versionpb"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type ClusterInfo struct {
	ID                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	DeploymentID         string   `protobuf:"bytes,2,opt,name=deployment_id,json=deploymentId,proto3" json:"deployment_id,omitempty"`
	VersionWarningsOk    bool     `protobuf:"varint,3,opt,name=version_warnings_ok,json=versionWarningsOk,proto3" json:"version_warnings_ok,omitempty"`
	VersionWarnings      []string `protobuf:"bytes,4,rep,name=version_warnings,json=versionWarnings,proto3" json:"version_warnings,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ClusterInfo) Reset()         { *m = ClusterInfo{} }
func (m *ClusterInfo) String() string { return proto.CompactTextString(m) }
func (*ClusterInfo) ProtoMessage()    {}
func (*ClusterInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_8595c8dce2486799, []int{0}
}
func (m *ClusterInfo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ClusterInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ClusterInfo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ClusterInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ClusterInfo.Merge(m, src)
}
func (m *ClusterInfo) XXX_Size() int {
	return m.Size()
}
func (m *ClusterInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_ClusterInfo.DiscardUnknown(m)
}

var xxx_messageInfo_ClusterInfo proto.InternalMessageInfo

func (m *ClusterInfo) GetID() string {
	if m != nil {
		return m.ID
	}
	return ""
}

func (m *ClusterInfo) GetDeploymentID() string {
	if m != nil {
		return m.DeploymentID
	}
	return ""
}

func (m *ClusterInfo) GetVersionWarningsOk() bool {
	if m != nil {
		return m.VersionWarningsOk
	}
	return false
}

func (m *ClusterInfo) GetVersionWarnings() []string {
	if m != nil {
		return m.VersionWarnings
	}
	return nil
}

type InspectClusterRequest struct {
	ClientVersion        *versionpb.Version `protobuf:"bytes,1,opt,name=client_version,json=clientVersion,proto3" json:"client_version,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *InspectClusterRequest) Reset()         { *m = InspectClusterRequest{} }
func (m *InspectClusterRequest) String() string { return proto.CompactTextString(m) }
func (*InspectClusterRequest) ProtoMessage()    {}
func (*InspectClusterRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8595c8dce2486799, []int{1}
}
func (m *InspectClusterRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *InspectClusterRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_InspectClusterRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *InspectClusterRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InspectClusterRequest.Merge(m, src)
}
func (m *InspectClusterRequest) XXX_Size() int {
	return m.Size()
}
func (m *InspectClusterRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_InspectClusterRequest.DiscardUnknown(m)
}

var xxx_messageInfo_InspectClusterRequest proto.InternalMessageInfo

func (m *InspectClusterRequest) GetClientVersion() *versionpb.Version {
	if m != nil {
		return m.ClientVersion
	}
	return nil
}

func init() {
	proto.RegisterType((*ClusterInfo)(nil), "admin_v2.ClusterInfo")
	proto.RegisterType((*InspectClusterRequest)(nil), "admin_v2.InspectClusterRequest")
}

func init() { proto.RegisterFile("admin/admin.proto", fileDescriptor_8595c8dce2486799) }

var fileDescriptor_8595c8dce2486799 = []byte{
	// 337 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x4c, 0x4c, 0xc9, 0xcd,
	0xcc, 0xd3, 0x07, 0x93, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42, 0x1c, 0x60, 0x4e, 0x7c, 0x99,
	0x91, 0x94, 0x48, 0x7a, 0x7e, 0x7a, 0x3e, 0x58, 0x50, 0x1f, 0xc4, 0x82, 0xc8, 0x4b, 0xc9, 0x97,
	0xa5, 0x16, 0x15, 0x67, 0xe6, 0xe7, 0xe9, 0x43, 0xe9, 0x82, 0x24, 0x18, 0x0b, 0xa2, 0x40, 0x69,
	0x3b, 0x23, 0x17, 0xb7, 0x73, 0x4e, 0x69, 0x71, 0x49, 0x6a, 0x91, 0x67, 0x5e, 0x5a, 0xbe, 0x90,
	0x18, 0x17, 0x53, 0x66, 0x8a, 0x04, 0xa3, 0x02, 0xa3, 0x06, 0xa7, 0x13, 0xdb, 0xa3, 0x7b, 0xf2,
	0x4c, 0x9e, 0x2e, 0x41, 0x4c, 0x99, 0x29, 0x42, 0xa6, 0x5c, 0xbc, 0x29, 0xa9, 0x05, 0x39, 0xf9,
	0x95, 0xb9, 0xa9, 0x79, 0x25, 0xf1, 0x99, 0x29, 0x12, 0x4c, 0x60, 0x25, 0x02, 0x8f, 0xee, 0xc9,
	0xf3, 0xb8, 0xc0, 0x25, 0x3c, 0x5d, 0x82, 0x78, 0x10, 0xca, 0x3c, 0x53, 0x84, 0xf4, 0xb8, 0x84,
	0xa1, 0xf6, 0xc5, 0x97, 0x27, 0x16, 0xe5, 0x65, 0xe6, 0xa5, 0x17, 0xc7, 0xe7, 0x67, 0x4b, 0x30,
	0x2b, 0x30, 0x6a, 0x70, 0x04, 0x09, 0x42, 0xa5, 0xc2, 0xa1, 0x32, 0xfe, 0xd9, 0x42, 0x9a, 0x5c,
	0x02, 0xe8, 0xea, 0x25, 0x58, 0x14, 0x98, 0x35, 0x38, 0x83, 0xf8, 0xd1, 0x14, 0x2b, 0x85, 0x72,
	0x89, 0x7a, 0xe6, 0x15, 0x17, 0xa4, 0x26, 0x97, 0x40, 0xdd, 0x1f, 0x94, 0x5a, 0x58, 0x9a, 0x5a,
	0x5c, 0x22, 0x64, 0xc3, 0xc5, 0x97, 0x9c, 0x93, 0x09, 0x72, 0x26, 0x54, 0x0b, 0xd8, 0x3b, 0xdc,
	0x46, 0xa2, 0x7a, 0xf0, 0x40, 0x88, 0x2f, 0x33, 0xd2, 0x0b, 0x83, 0x70, 0x82, 0x78, 0x21, 0x8a,
	0xa1, 0x5c, 0xa3, 0x40, 0x2e, 0x66, 0xc7, 0x00, 0x4f, 0x21, 0x2f, 0x2e, 0x3e, 0x54, 0xd3, 0x85,
	0xe4, 0xf5, 0x60, 0x61, 0xad, 0x87, 0xd5, 0x5e, 0x29, 0x51, 0x84, 0x02, 0xa4, 0x10, 0x55, 0x62,
	0x70, 0xb2, 0x3c, 0xf1, 0x48, 0x8e, 0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x07, 0x8f, 0xe4, 0x18, 0xa3,
	0xb4, 0xd3, 0x33, 0x4b, 0x32, 0x4a, 0x93, 0xf4, 0x92, 0xf3, 0x73, 0xf5, 0x0b, 0x12, 0x93, 0x33,
	0x2a, 0x53, 0x52, 0x8b, 0x90, 0x59, 0x65, 0x46, 0xfa, 0xc5, 0x45, 0xc9, 0x90, 0x58, 0x4e, 0x62,
	0x03, 0xc7, 0x92, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0x62, 0xd6, 0x69, 0x0e, 0xfb, 0x01, 0x00,
	0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// APIClient is the client API for API service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type APIClient interface {
	InspectCluster(ctx context.Context, in *InspectClusterRequest, opts ...grpc.CallOption) (*ClusterInfo, error)
}

type aPIClient struct {
	cc *grpc.ClientConn
}

func NewAPIClient(cc *grpc.ClientConn) APIClient {
	return &aPIClient{cc}
}

func (c *aPIClient) InspectCluster(ctx context.Context, in *InspectClusterRequest, opts ...grpc.CallOption) (*ClusterInfo, error) {
	out := new(ClusterInfo)
	err := c.cc.Invoke(ctx, "/admin_v2.API/InspectCluster", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// APIServer is the server API for API service.
type APIServer interface {
	InspectCluster(context.Context, *InspectClusterRequest) (*ClusterInfo, error)
}

// UnimplementedAPIServer can be embedded to have forward compatible implementations.
type UnimplementedAPIServer struct {
}

func (*UnimplementedAPIServer) InspectCluster(ctx context.Context, req *InspectClusterRequest) (*ClusterInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InspectCluster not implemented")
}

func RegisterAPIServer(s *grpc.Server, srv APIServer) {
	s.RegisterService(&_API_serviceDesc, srv)
}

func _API_InspectCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InspectClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(APIServer).InspectCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/admin_v2.API/InspectCluster",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(APIServer).InspectCluster(ctx, req.(*InspectClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _API_serviceDesc = grpc.ServiceDesc{
	ServiceName: "admin_v2.API",
	HandlerType: (*APIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "InspectCluster",
			Handler:    _API_InspectCluster_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "admin/admin.proto",
}

func (m *ClusterInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ClusterInfo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ClusterInfo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.VersionWarnings) > 0 {
		for iNdEx := len(m.VersionWarnings) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.VersionWarnings[iNdEx])
			copy(dAtA[i:], m.VersionWarnings[iNdEx])
			i = encodeVarintAdmin(dAtA, i, uint64(len(m.VersionWarnings[iNdEx])))
			i--
			dAtA[i] = 0x22
		}
	}
	if m.VersionWarningsOk {
		i--
		if m.VersionWarningsOk {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x18
	}
	if len(m.DeploymentID) > 0 {
		i -= len(m.DeploymentID)
		copy(dAtA[i:], m.DeploymentID)
		i = encodeVarintAdmin(dAtA, i, uint64(len(m.DeploymentID)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintAdmin(dAtA, i, uint64(len(m.ID)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *InspectClusterRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *InspectClusterRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *InspectClusterRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.ClientVersion != nil {
		{
			size, err := m.ClientVersion.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintAdmin(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintAdmin(dAtA []byte, offset int, v uint64) int {
	offset -= sovAdmin(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ClusterInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovAdmin(uint64(l))
	}
	l = len(m.DeploymentID)
	if l > 0 {
		n += 1 + l + sovAdmin(uint64(l))
	}
	if m.VersionWarningsOk {
		n += 2
	}
	if len(m.VersionWarnings) > 0 {
		for _, s := range m.VersionWarnings {
			l = len(s)
			n += 1 + l + sovAdmin(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *InspectClusterRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.ClientVersion != nil {
		l = m.ClientVersion.Size()
		n += 1 + l + sovAdmin(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovAdmin(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozAdmin(x uint64) (n int) {
	return sovAdmin(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ClusterInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAdmin
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ClusterInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ClusterInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAdmin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAdmin
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAdmin
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DeploymentID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAdmin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAdmin
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAdmin
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DeploymentID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field VersionWarningsOk", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAdmin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.VersionWarningsOk = bool(v != 0)
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field VersionWarnings", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAdmin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAdmin
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAdmin
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.VersionWarnings = append(m.VersionWarnings, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAdmin(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthAdmin
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *InspectClusterRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAdmin
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: InspectClusterRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: InspectClusterRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClientVersion", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAdmin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAdmin
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAdmin
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.ClientVersion == nil {
				m.ClientVersion = &versionpb.Version{}
			}
			if err := m.ClientVersion.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAdmin(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthAdmin
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipAdmin(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAdmin
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAdmin
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAdmin
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthAdmin
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupAdmin
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthAdmin
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthAdmin        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAdmin          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupAdmin = fmt.Errorf("proto: unexpected end of group")
)
