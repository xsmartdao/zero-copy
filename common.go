package zero_copy

const (
	Uint16Size  = 2
	Uint32Size  = 4
	Uint64Size  = 8
	Uint256Size = 32
)

const AddrLen = 20

type Address [AddrLen]byte

type Uint256 [Uint256Size]byte
