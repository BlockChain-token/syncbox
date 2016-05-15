package syncbox

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

// constants for protocol
const (
	RequestPrefix  = 'q'
	ResponsePrefix = 's'

	PacketDataSize  = 1024
	PacketAddrSize  = 4 // roughly allow 4000 GB message size
	PacketTotalSize = 1032

	ByteDelim   = byte(4)
	StringDelim = string(ByteDelim)

	TypeIdentity    = "IDENTITY"
	TypeDigest      = "DIGEST"
	TypeSyncRequest = "SYNC-REQUEST"
	TypeFile        = "FILE"

	StatusOK  = 200
	StatusBad = 400

	MessageAccept = "ACCEPT"
	MessageDeny   = "DENY"

	SyncboxServerUsernam = "SYNCBOX-SERVER"
)

// Packet is a fixed length message as the basic element to send acrosss network
type Packet struct {
	Size     [PacketAddrSize]byte // size is the maximun number of Sequence for packets consist of this message
	Sequence [PacketAddrSize]byte
	Data     [PacketDataSize]byte
}

// NewPacket instantiates a packet
func NewPacket(size int, sequence int, data [PacketDataSize]byte) (*Packet, error) {
	packet := &Packet{
		Data: data,
	}
	if err := packet.SetSize(size); err != nil {
		return nil, err
	}
	if err := packet.SetSequence(sequence); err != nil {
		return nil, err
	}
	return packet, nil
}

func intToBinary(num uint32) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, num)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func binaryToInt(bin []byte) (uint32, error) {
	var num uint32
	buf := bytes.NewReader(bin)
	err := binary.Read(buf, binary.LittleEndian, &num)
	if err != nil {
		return math.MaxInt32, err
	}
	return num, nil
}

// SetSize sets the total message size to the packet,
// maximum size is 2 ^ (PacketAddrSize * 8)
func (packet *Packet) SetSize(size int) error {
	bytes, err := intToBinary(uint32(size))
	if err != nil {
		return err
	}
	if len(bytes) > PacketAddrSize {
		return ErrorExceedsAddrLength
	}
	copy(packet.Size[:], bytes)
	return nil
}

// GetSize gets the size of the packet
func (packet *Packet) GetSize() (int, error) {
	num, err := binaryToInt(packet.Size[:])
	if err != nil {
		return -math.MaxInt16, err
	}
	return int(num), nil
}

// SetSequence sets the sequence of the packet
func (packet *Packet) SetSequence(sequence int) error {
	bytes, err := intToBinary(uint32(sequence))
	if err != nil {
		return err
	}
	if len(bytes) > PacketAddrSize {
		return ErrorExceedsAddrLength
	}
	copy(packet.Sequence[:], bytes)
	return nil
}

// GetSequence get sequence of the packet
func (packet *Packet) GetSequence() (int, error) {
	num, err := binaryToInt(packet.Sequence[:])
	if err != nil {
		return -math.MaxInt16, err
	}
	return int(num), nil
}

// ToBytes transfer a Packet to fixed length byte array
func (packet *Packet) ToBytes() [PacketTotalSize]byte {
	var bytes [PacketTotalSize]byte
	copy(bytes[0:PacketAddrSize], packet.Size[:])
	copy(bytes[PacketAddrSize:2*PacketAddrSize], packet.Sequence[:])
	copy(bytes[2*PacketAddrSize:PacketTotalSize], packet.Data[:])
	return bytes
}

// RebornPacket reborn a packet from a fixed length byte array
func RebornPacket(data [PacketTotalSize]byte) *Packet {
	var packet Packet
	copy(packet.Size[:], data[0:PacketAddrSize])
	copy(packet.Sequence[:], data[PacketAddrSize:2*PacketAddrSize])
	copy(packet.Data[:], data[2*PacketAddrSize:PacketTotalSize])
	return &packet
}

// Serialize transfer some data (a request/response) to series of packets
func Serialize(data []byte) ([]Packet, error) {
	size := int(math.Ceil(float64(len(data)) / PacketDataSize))
	var packets []Packet
	for sequence := 0; sequence < size; sequence++ {
		var payload [PacketDataSize]byte
		if sequence == size-1 {
			copy(payload[:], data[sequence*PacketDataSize:len(data)])
		} else {
			copy(payload[:], data[sequence*PacketDataSize:(sequence+1)*PacketDataSize])
		}
		packet, err := NewPacket(size, sequence, payload)
		if err != nil {
			return nil, err
		}
		packets = append(packets, *packet)
	}
	return packets, nil
}

// Deserialize transfer a series of packets to some data (a request or response)
func Deserialize(packets []Packet) []byte {
	packetsCount := len(packets)
	dataSize := packetsCount * PacketDataSize
	data := make([]byte, dataSize)
	offset := 0
	for _, packet := range packets {
		copy(data[offset:offset+PacketDataSize], packet.Data[:])
		offset += PacketDataSize
	}
	return data
}

// Request structure for request
type Request struct {
	Username string
	DataType string
	Data     []byte
}

func (req *Request) String() string {
	str := fmt.Sprintf("Username: %v\n", req.Username)
	str += fmt.Sprintf("DataType: %v\n", req.DataType)
	str += fmt.Sprintf("Data: %v\n", string(req.Data))
	return str
}

// Response structure for response
type Response struct {
	Status  int
	Message string
	Data    []byte
}

func (res *Response) String() string {
	str := fmt.Sprintf("Status: %v\n", res.Status)
	str += fmt.Sprintf("Message: %v\n", res.Message)
	str += fmt.Sprintf("Data: %v\n", string(res.Data))
	return str
}

// IdentityRequest is the Request data type of user identity
type IdentityRequest struct {
	Username string
}

func (req *IdentityRequest) String() string {
	str := fmt.Sprintf("Username: %v\n", req.Username)
	return str
}

// DigestRequest is the Request data type of a file tree digest
type DigestRequest struct {
	Dir *Dir
}

func (req *DigestRequest) String() string {
	str := fmt.Sprintf("Dir: %v\n", req.Dir)
	return str
}

// SyncRequest is the Request data type of a file CRUD request
type SyncRequest struct {
	Action string
	File   *File
}

func (req *SyncRequest) String() string {
	str := fmt.Sprintf("Action: %v\n", req.Action)
	str += fmt.Sprintf("File: %v\n", req.File)
	return str
}

// FileRequest is the Request data type of CRUD on file content
type FileRequest struct {
	File    *File
	Content []byte
}

func (req *FileRequest) String() string {
	str := fmt.Sprintf("File: %v\n", req.File)
	str += fmt.Sprintf("Content: %v\n", string(req.Content))
	return str
}

// ToJSON converts request to JSON string
func (req *Request) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// RebornRequest reborn request from JSON string
func RebornRequest(jsonStr string) (*Request, error) {
	jsonBytes := []byte(jsonStr)
	restoredReq := Request{}
	if err := json.Unmarshal(jsonBytes, &restoredReq); err != nil {
		return nil, err
	}
	return &restoredReq, nil
}

// ToJSON converts response to JSON string
func (res *Response) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// RebornResponse reborn response from JSON string
func RebornResponse(jsonStr string) (*Response, error) {
	jsonBytes := []byte(jsonStr)
	restoredRes := Response{}
	if err := json.Unmarshal(jsonBytes, &restoredRes); err != nil {
		return nil, err
	}
	return &restoredRes, nil
}