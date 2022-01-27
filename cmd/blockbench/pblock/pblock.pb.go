// This file was created by:
// - Checking out https://github.com/tendermint/spec.git
// - copying pblock.proto there under proto/tendermint/pblock/pblock.proto
// - running `make proto-gen` there
// - copying back the resulting pblock.pb.go
// - manually editing:
//   + removing the broken `types2` import
//   + changing `types1` and `types2` into `abci` everywhere

// source: tendermint/pblock/pblock.proto

package pblock

import (
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"
	time "time"

	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	github_com_gogo_protobuf_types "github.com/gogo/protobuf/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type PBlock struct {
	Height                int64                     `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	Time                  time.Time                 `protobuf:"bytes,2,opt,name=time,proto3,stdtime" json:"time"`
	Hash                  []byte                    `protobuf:"bytes,3,opt,name=hash,proto3" json:"hash,omitempty"`
	TxsResults            []*abci.ResponseDeliverTx `protobuf:"bytes,4,rep,name=txs_results,json=txsResults,proto3" json:"txs_results,omitempty"`
	BeginBlockEvents      []abci.Event              `protobuf:"bytes,5,rep,name=begin_block_events,json=beginBlockEvents,proto3" json:"begin_block_events"`
	EndBlockEvents        []abci.Event              `protobuf:"bytes,6,rep,name=end_block_events,json=endBlockEvents,proto3" json:"end_block_events"`
	ValidatorUpdates      []abci.ValidatorUpdate    `protobuf:"bytes,7,rep,name=validator_updates,json=validatorUpdates,proto3" json:"validator_updates"`
	ConsensusParamUpdates *abci.ConsensusParams     `protobuf:"bytes,8,opt,name=consensus_param_updates,json=consensusParamUpdates,proto3" json:"consensus_param_updates,omitempty"`
}

func (m *PBlock) Reset()         { *m = PBlock{} }
func (m *PBlock) String() string { return proto.CompactTextString(m) }
func (*PBlock) ProtoMessage()    {}
func (*PBlock) Descriptor() ([]byte, []int) {
	return fileDescriptor_95d168953e916cad, []int{0}
}
func (m *PBlock) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *PBlock) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_PBlock.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *PBlock) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PBlock.Merge(m, src)
}
func (m *PBlock) XXX_Size() int {
	return m.Size()
}
func (m *PBlock) XXX_DiscardUnknown() {
	xxx_messageInfo_PBlock.DiscardUnknown(m)
}

var xxx_messageInfo_PBlock proto.InternalMessageInfo

func (m *PBlock) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *PBlock) GetTime() time.Time {
	if m != nil {
		return m.Time
	}
	return time.Time{}
}

func (m *PBlock) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *PBlock) GetTxsResults() []*abci.ResponseDeliverTx {
	if m != nil {
		return m.TxsResults
	}
	return nil
}

func (m *PBlock) GetBeginBlockEvents() []abci.Event {
	if m != nil {
		return m.BeginBlockEvents
	}
	return nil
}

func (m *PBlock) GetEndBlockEvents() []abci.Event {
	if m != nil {
		return m.EndBlockEvents
	}
	return nil
}

func (m *PBlock) GetValidatorUpdates() []abci.ValidatorUpdate {
	if m != nil {
		return m.ValidatorUpdates
	}
	return nil
}

func (m *PBlock) GetConsensusParamUpdates() *abci.ConsensusParams {
	if m != nil {
		return m.ConsensusParamUpdates
	}
	return nil
}

func init() {
	proto.RegisterType((*PBlock)(nil), "tendermint.pblock.PBlock")
}

func init() { proto.RegisterFile("tendermint/pblock/pblock.proto", fileDescriptor_95d168953e916cad) }

var fileDescriptor_95d168953e916cad = []byte{
	// 451 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x92, 0xb1, 0x8e, 0xd3, 0x40,
	0x10, 0x86, 0x63, 0x12, 0xc2, 0x69, 0x83, 0xd0, 0xdd, 0x0a, 0x0e, 0x2b, 0x08, 0xc7, 0x5c, 0x95,
	0xca, 0x2b, 0x85, 0x86, 0x3a, 0x07, 0x14, 0x88, 0xe2, 0x64, 0x0e, 0x24, 0x68, 0xac, 0xb5, 0x3d,
	0xd8, 0x2b, 0xec, 0x5d, 0x6b, 0x77, 0x63, 0x85, 0xb7, 0xb8, 0xc7, 0xba, 0xf2, 0x4a, 0x2a, 0x40,
	0xc9, 0x43, 0xd0, 0x22, 0x8f, 0x6d, 0x70, 0x74, 0xcd, 0x55, 0x1e, 0xcf, 0x3f, 0xff, 0x37, 0xbb,
	0xb3, 0x43, 0x3c, 0x0b, 0x32, 0x05, 0x5d, 0x0a, 0x69, 0x59, 0x15, 0x17, 0x2a, 0xf9, 0xd6, 0x7d,
	0x82, 0x4a, 0x2b, 0xab, 0xe8, 0xc9, 0x7f, 0x3d, 0x68, 0x85, 0xf9, 0xb3, 0x81, 0x85, 0xc7, 0x89,
	0x60, 0xf6, 0x7b, 0x05, 0xa6, 0xad, 0x9f, 0x3f, 0x1f, 0x88, 0x98, 0x67, 0x15, 0xd7, 0xbc, 0xec,
	0xe5, 0x45, 0xa6, 0x54, 0x56, 0x00, 0xc3, 0xbf, 0x78, 0xf3, 0x95, 0x59, 0x51, 0x82, 0xb1, 0xbc,
	0xac, 0xba, 0x82, 0xc7, 0x99, 0xca, 0x14, 0x86, 0xac, 0x89, 0xda, 0xec, 0xd9, 0x9f, 0x31, 0x99,
	0x5e, 0xac, 0x9b, 0xee, 0xf4, 0x94, 0x4c, 0x73, 0x10, 0x59, 0x6e, 0x5d, 0xc7, 0x77, 0x96, 0xe3,
	0xb0, 0xfb, 0xa3, 0xaf, 0xc8, 0xa4, 0x61, 0xb9, 0xf7, 0x7c, 0x67, 0x39, 0x5b, 0xcd, 0x83, 0xb6,
	0x51, 0xd0, 0x37, 0x0a, 0x2e, 0xfb, 0x46, 0xeb, 0xa3, 0xeb, 0x9f, 0x8b, 0xd1, 0xd5, 0xaf, 0x85,
	0x13, 0xa2, 0x83, 0x52, 0x32, 0xc9, 0xb9, 0xc9, 0xdd, 0xb1, 0xef, 0x2c, 0x1f, 0x86, 0x18, 0xd3,
	0x73, 0x32, 0xb3, 0x5b, 0x13, 0x69, 0x30, 0x9b, 0xc2, 0x1a, 0x77, 0xe2, 0x8f, 0x97, 0xb3, 0xd5,
	0x59, 0x30, 0x18, 0x46, 0x73, 0xf3, 0x20, 0x04, 0x53, 0x29, 0x69, 0xe0, 0x35, 0x14, 0xa2, 0x06,
	0x7d, 0xb9, 0x0d, 0x89, 0xdd, 0x9a, 0xb0, 0x75, 0xd1, 0x77, 0x84, 0xc6, 0x90, 0x09, 0x19, 0xe1,
	0xdc, 0x22, 0xa8, 0x41, 0x5a, 0xe3, 0xde, 0x47, 0xd6, 0xe9, 0x2d, 0xd6, 0x9b, 0x46, 0x5e, 0x4f,
	0x9a, 0xc3, 0x85, 0xc7, 0xe8, 0xc3, 0x0b, 0x63, 0xda, 0xd0, 0xb7, 0xe4, 0x18, 0x64, 0x7a, 0x48,
	0x9a, 0xde, 0x81, 0xf4, 0x08, 0x64, 0x3a, 0xe4, 0x7c, 0x20, 0x27, 0x35, 0x2f, 0x44, 0xca, 0xad,
	0xd2, 0xd1, 0xa6, 0x4a, 0xb9, 0x05, 0xe3, 0x3e, 0x40, 0x90, 0x7f, 0x0b, 0xf4, 0xa9, 0xaf, 0xfc,
	0x88, 0x85, 0xfd, 0xe1, 0xea, 0xc3, 0xb4, 0xa1, 0x9f, 0xc9, 0xd3, 0xa4, 0x19, 0x83, 0x34, 0x1b,
	0x13, 0xe1, 0x7b, 0xff, 0x43, 0x1f, 0xe1, 0x73, 0xbc, 0x18, 0xa2, 0xdb, 0x75, 0x39, 0xef, 0x0d,
	0x17, 0xb8, 0x1f, 0xe1, 0x93, 0xe4, 0x20, 0xd1, 0xa1, 0xd7, 0xef, 0xaf, 0x77, 0x9e, 0x73, 0xb3,
	0xf3, 0x9c, 0xdf, 0x3b, 0xcf, 0xb9, 0xda, 0x7b, 0xa3, 0x9b, 0xbd, 0x37, 0xfa, 0xb1, 0xf7, 0x46,
	0x5f, 0x56, 0x99, 0xb0, 0x05, 0x8f, 0x83, 0x44, 0x95, 0xcc, 0xe6, 0x4a, 0x27, 0x39, 0x17, 0x92,
	0x95, 0x22, 0xcd, 0xb8, 0x4e, 0x59, 0x52, 0xa6, 0x0c, 0x87, 0x15, 0x83, 0x4c, 0xf2, 0x6e, 0xa7,
	0xe3, 0x29, 0xae, 0xc3, 0xcb, 0xbf, 0x01, 0x00, 0x00, 0xff, 0xff, 0xda, 0x40, 0x03, 0x6c, 0xf6,
	0x02, 0x00, 0x00,
}

func (m *PBlock) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PBlock) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *PBlock) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.ConsensusParamUpdates != nil {
		{
			size, err := m.ConsensusParamUpdates.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintPblock(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x42
	}
	if len(m.ValidatorUpdates) > 0 {
		for iNdEx := len(m.ValidatorUpdates) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.ValidatorUpdates[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintPblock(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x3a
		}
	}
	if len(m.EndBlockEvents) > 0 {
		for iNdEx := len(m.EndBlockEvents) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.EndBlockEvents[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintPblock(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x32
		}
	}
	if len(m.BeginBlockEvents) > 0 {
		for iNdEx := len(m.BeginBlockEvents) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.BeginBlockEvents[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintPblock(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x2a
		}
	}
	if len(m.TxsResults) > 0 {
		for iNdEx := len(m.TxsResults) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.TxsResults[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintPblock(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.Hash) > 0 {
		i -= len(m.Hash)
		copy(dAtA[i:], m.Hash)
		i = encodeVarintPblock(dAtA, i, uint64(len(m.Hash)))
		i--
		dAtA[i] = 0x1a
	}
	n2, err2 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Time, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Time):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintPblock(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x12
	if m.Height != 0 {
		i = encodeVarintPblock(dAtA, i, uint64(m.Height))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintPblock(dAtA []byte, offset int, v uint64) int {
	offset -= sovPblock(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *PBlock) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Height != 0 {
		n += 1 + sovPblock(uint64(m.Height))
	}
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Time)
	n += 1 + l + sovPblock(uint64(l))
	l = len(m.Hash)
	if l > 0 {
		n += 1 + l + sovPblock(uint64(l))
	}
	if len(m.TxsResults) > 0 {
		for _, e := range m.TxsResults {
			l = e.Size()
			n += 1 + l + sovPblock(uint64(l))
		}
	}
	if len(m.BeginBlockEvents) > 0 {
		for _, e := range m.BeginBlockEvents {
			l = e.Size()
			n += 1 + l + sovPblock(uint64(l))
		}
	}
	if len(m.EndBlockEvents) > 0 {
		for _, e := range m.EndBlockEvents {
			l = e.Size()
			n += 1 + l + sovPblock(uint64(l))
		}
	}
	if len(m.ValidatorUpdates) > 0 {
		for _, e := range m.ValidatorUpdates {
			l = e.Size()
			n += 1 + l + sovPblock(uint64(l))
		}
	}
	if m.ConsensusParamUpdates != nil {
		l = m.ConsensusParamUpdates.Size()
		n += 1 + l + sovPblock(uint64(l))
	}
	return n
}

func sovPblock(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozPblock(x uint64) (n int) {
	return sovPblock(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *PBlock) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPblock
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
			return fmt.Errorf("proto: PBlock: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PBlock: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Height", wireType)
			}
			m.Height = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPblock
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Height |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Time", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPblock
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
				return ErrInvalidLengthPblock
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPblock
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Time, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Hash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPblock
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthPblock
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthPblock
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Hash = append(m.Hash[:0], dAtA[iNdEx:postIndex]...)
			if m.Hash == nil {
				m.Hash = []byte{}
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TxsResults", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPblock
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
				return ErrInvalidLengthPblock
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPblock
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TxsResults = append(m.TxsResults, &abci.ResponseDeliverTx{})
			if err := m.TxsResults[len(m.TxsResults)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BeginBlockEvents", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPblock
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
				return ErrInvalidLengthPblock
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPblock
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BeginBlockEvents = append(m.BeginBlockEvents, abci.Event{})
			if err := m.BeginBlockEvents[len(m.BeginBlockEvents)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field EndBlockEvents", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPblock
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
				return ErrInvalidLengthPblock
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPblock
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.EndBlockEvents = append(m.EndBlockEvents, abci.Event{})
			if err := m.EndBlockEvents[len(m.EndBlockEvents)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ValidatorUpdates", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPblock
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
				return ErrInvalidLengthPblock
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPblock
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ValidatorUpdates = append(m.ValidatorUpdates, abci.ValidatorUpdate{})
			if err := m.ValidatorUpdates[len(m.ValidatorUpdates)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ConsensusParamUpdates", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPblock
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
				return ErrInvalidLengthPblock
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthPblock
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.ConsensusParamUpdates == nil {
				m.ConsensusParamUpdates = &abci.ConsensusParams{}
			}
			if err := m.ConsensusParamUpdates.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPblock(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthPblock
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipPblock(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowPblock
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
					return 0, ErrIntOverflowPblock
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
					return 0, ErrIntOverflowPblock
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
				return 0, ErrInvalidLengthPblock
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupPblock
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthPblock
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthPblock        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowPblock          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupPblock = fmt.Errorf("proto: unexpected end of group")
)
