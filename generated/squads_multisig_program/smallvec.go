package squads_multisig_program

import (
	"encoding/binary"
	"fmt"
	"io"

	ag_solanago "github.com/gagliardetto/solana-go"
)

// SmallVec corresponds to Rust's SmallVec<L, T>
type SmallVec[L LengthType, T any] struct {
	Data []T
}

// LengthType defines length type constraints
type LengthType interface {
	~uint8 | ~uint16
}

// Encodable defines the encoding interface
type Encodable interface {
	EncodeWith(e *Encoder) error
}

// Decodable defines the decoding interface
type Decodable interface {
	DecodeWith(d *Decoder) error
}

// Encoder for encoding
type Encoder struct {
	w io.Writer
}

// NewEncoder creates a new encoder
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode encodes values
func (e *Encoder) Encode(v interface{}) error {
	switch val := v.(type) {
	case Encodable:
		return val.EncodeWith(e)
	case *SmallVec[uint8, ag_solanago.PublicKey]:
		// Encode length
		if err := e.Encode(uint8(len(val.Data))); err != nil {
			return err
		}
		// Encode data
		for _, item := range val.Data {
			if err := e.Encode(item); err != nil {
				return err
			}
		}
		return nil
	case *SmallVec[uint8, CompiledInstruction]:
		// Encode length
		if err := e.Encode(uint8(len(val.Data))); err != nil {
			return err
		}
		// Encode data
		for _, item := range val.Data {
			if err := e.Encode(&item); err != nil {
				return err
			}
		}
		return nil
	case *SmallVec[uint8, MessageAddressTableLookup]:
		// Encode length
		if err := e.Encode(uint8(len(val.Data))); err != nil {
			return err
		}
		// Encode data
		for _, item := range val.Data {
			if err := e.Encode(&item); err != nil {
				return err
			}
		}
		return nil
	case SmallVec[uint16, uint8]:
		// Encode length
		if err := e.Encode(uint16(len(val.Data))); err != nil {
			return err
		}
		// Encode data
		for _, item := range val.Data {
			if err := e.Encode(&item); err != nil {
				return err
			}
		}
		return nil
	case *uint8, uint8:
		return binary.Write(e.w, binary.LittleEndian, val)
	case uint16:
		return binary.Write(e.w, binary.LittleEndian, val)
	case []uint8:
		_, err := e.w.Write(val)
		return err
	case ag_solanago.PublicKey:
		_, err := e.w.Write(val[:])
		return err
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// Decoder for decoding
type Decoder struct {
	r io.Reader
}

// NewDecoder creates a new decoder
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode decodes values
func (d *Decoder) Decode(v interface{}) error {
	switch val := v.(type) {
	case Decodable:
		return val.DecodeWith(d)
	case *SmallVec[uint8, ag_solanago.PublicKey]:
		// Decode length
		var length uint8
		if err := d.Decode(&length); err != nil {
			return err
		}
		// Initialize slice
		val.Data = make([]ag_solanago.PublicKey, length)
		// Decode data
		for i := range val.Data {
			if err := d.Decode(&val.Data[i]); err != nil {
				return err
			}
		}
		return nil
	case *SmallVec[uint8, CompiledInstruction]:
		// Decode length
		var length uint8
		if err := d.Decode(&length); err != nil {
			return err
		}
		// Initialize slice
		val.Data = make([]CompiledInstruction, length)
		// Decode data
		for i := range val.Data {
			if err := d.Decode(&val.Data[i]); err != nil {
				return err
			}
		}
		return nil
	case *SmallVec[uint8, MessageAddressTableLookup]:
		// Decode length
		var length uint8
		if err := d.Decode(&length); err != nil {
			return err
		}
		// Initialize slice
		val.Data = make([]MessageAddressTableLookup, length)
		// Decode data
		for i := range val.Data {
			if err := d.Decode(&val.Data[i]); err != nil {
				return err
			}
		}
		return nil
	case *uint8:
		return binary.Read(d.r, binary.LittleEndian, val)
	case *uint16:
		return binary.Read(d.r, binary.LittleEndian, val)
	case *[]uint8:
		_, err := io.ReadFull(d.r, *val)
		return err
	case []uint8:
		_, err := io.ReadFull(d.r, val[:])
		return err
	case *ag_solanago.PublicKey:
		_, err := io.ReadFull(d.r, val[:])
		return err
	case *SmallVec[uint8, uint8]:
		// Decode length
		var length uint8
		if err := d.Decode(&length); err != nil {
			return err
		}
		// Initialize slice
		val.Data = make([]uint8, length)
		// Decode data
		for i := range val.Data {
			if err := d.Decode(&val.Data[i]); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// TransactionMessage encoding implementation
func (tm *TransactionMessage) EncodeWith(e *Encoder) error {
	// Encode basic fields
	if err := e.Encode(tm.NumSigners); err != nil {
		return err
	}
	if err := e.Encode(tm.NumWritableSigners); err != nil {
		return err
	}
	if err := e.Encode(tm.NumWritableNonSigners); err != nil {
		return err
	}

	// Encode SmallVec fields
	if err := e.Encode(&tm.AccountKeys); err != nil {
		return err
	}
	if err := e.Encode(&tm.Instructions); err != nil {
		return err
	}
	if err := e.Encode(&tm.AddressTableLookups); err != nil {
		return err
	}

	return nil
}

// TransactionMessage decoding implementation
func (tm *TransactionMessage) DecodeWith(d *Decoder) error {
	// Decode basic fields
	if err := d.Decode(&tm.NumSigners); err != nil {
		return err
	}
	if err := d.Decode(&tm.NumWritableSigners); err != nil {
		return err
	}
	if err := d.Decode(&tm.NumWritableNonSigners); err != nil {
		return err
	}

	// Decode SmallVec fields
	if err := d.Decode(&tm.AccountKeys); err != nil {
		return err
	}
	if err := d.Decode(&tm.Instructions); err != nil {
		return err
	}
	if err := d.Decode(&tm.AddressTableLookups); err != nil {
		return err
	}

	return nil
}

// CompiledInstruction encoding implementation
func (ci *CompiledInstruction) EncodeWith(e *Encoder) error {
	if err := e.Encode(ci.ProgramIdIndex); err != nil {
		return err
	}

	// Encode accounts array
	accounts := ci.AccountIndexes.Data // Get underlying slice
	if err := e.Encode(uint8(len(accounts))); err != nil {
		return err
	}
	if err := e.Encode(accounts); err != nil {
		return err
	}

	// Encode data array
	data := ci.Data.Data // Get underlying slice
	if err := e.Encode(uint16(len(data))); err != nil {
		return err
	}
	return e.Encode(ci.Data.Data)
}

// CompiledInstruction decoding implementation
func (ci *CompiledInstruction) DecodeWith(d *Decoder) error {
	if err := d.Decode(&ci.ProgramIdIndex); err != nil {
		return err
	}
	// Decode accounts array
	var accountsLen uint8
	if err := d.Decode(&accountsLen); err != nil {
		return err
	}

	// Create slice with correct length and batch decode
	ci.AccountIndexes.Data = make([]uint8, accountsLen)
	if err := d.Decode(ci.AccountIndexes.Data); err != nil {
		return err
	}

	// Decode data array (using uint16 length)
	var dataLen uint16
	if err := d.Decode(&dataLen); err != nil {
		return err
	}

	// Create byte slice and batch read
	ci.Data.Data = make([]uint8, dataLen)
	_, err := io.ReadFull(d.r, ci.Data.Data)
	return err
}

// MessageAddressTableLookup encoding implementation (fixed version)
func (m *MessageAddressTableLookup) EncodeWith(e *Encoder) error {
	// Encode account public key
	if err := e.Encode(&m.AccountKey); err != nil {
		return err
	}

	// Encode writable indexes - using SmallVec's Data field
	writableIndexes := m.WritableIndexes.Data
	if err := e.Encode(uint8(len(writableIndexes))); err != nil {
		return err
	}
	// Batch encode for efficiency
	if err := e.Encode(writableIndexes); err != nil {
		return err
	}

	// Encode readonly indexes - using SmallVec's Data field
	readonlyIndexes := m.ReadonlyIndexes.Data
	if err := e.Encode(uint8(len(readonlyIndexes))); err != nil {
		return err
	}
	// Batch encode for efficiency
	return e.Encode(readonlyIndexes)
}

// MessageAddressTableLookup decoding implementation (fixed version)
func (m *MessageAddressTableLookup) DecodeWith(d *Decoder) error {
	// Decode account public key
	if err := d.Decode(&m.AccountKey); err != nil {
		return err
	}

	// Decode writable indexes
	var writableLen uint8
	if err := d.Decode(&writableLen); err != nil {
		return err
	}
	// Create slice and batch decode
	writableData := make([]uint8, writableLen)
	if err := d.Decode(writableData); err != nil {
		return err
	}
	m.WritableIndexes.Data = writableData

	// Decode readonly indexes
	var readonlyLen uint8
	if err := d.Decode(&readonlyLen); err != nil {
		return err
	}
	// Create slice and batch decode
	readonlyData := make([]uint8, readonlyLen)
	if err := d.Decode(readonlyData); err != nil {
		return err
	}
	m.ReadonlyIndexes.Data = readonlyData

	return nil
}
