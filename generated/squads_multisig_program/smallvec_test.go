package squads_multisig_program

import (
	"bytes"
	"testing"

	ag_solanago "github.com/gagliardetto/solana-go"
)

func TestEncode(t *testing.T) {
	var accountKeys SmallVec[uint8, ag_solanago.PublicKey] = SmallVec[uint8, ag_solanago.PublicKey]{
		Data: []ag_solanago.PublicKey{},
	}
	accountKeys.Data = append(accountKeys.Data, ag_solanago.MustPublicKeyFromBase58("SQDS4ep65T869zMMBKyuUq6aD6EgTu8psMjkvj52pCf"))
	var instructions SmallVec[uint8, CompiledInstruction] = SmallVec[uint8, CompiledInstruction]{
		Data: []CompiledInstruction{
			{ProgramIdIndex: 1,
				AccountIndexes: SmallVec[uint8, uint8]{
					Data: []uint8{1, 2, 3},
				},
				Data: SmallVec[uint16, uint8]{
					Data: []uint8{1, 2, 3, 4},
				},
			},
		},
	}
	msg := TransactionMessage{
		NumSigners:            1,
		NumWritableSigners:    2,
		NumWritableNonSigners: 3,
		AccountKeys:           accountKeys,
		Instructions:          instructions,
		AddressTableLookups: SmallVec[uint8, MessageAddressTableLookup]{
			Data: []MessageAddressTableLookup{},
		},
	}
	var buf bytes.Buffer

	if err := msg.EncodeWith(NewEncoder(&buf)); err != nil {
		t.Fatal(err)
	}
	var msgDecoded TransactionMessage
	if err := msgDecoded.DecodeWith(NewDecoder(bytes.NewBuffer(buf.Bytes()))); err != nil {
		t.Fatal(err)
	}
	if msg.NumSigners != msgDecoded.NumSigners {
		t.Fatal("NumSigners not equal")
	}
	if msg.NumWritableSigners != msgDecoded.NumWritableSigners {
		t.Fatal("NumWritableSigners not equal")
	}
	if msg.NumWritableNonSigners != msgDecoded.NumWritableNonSigners {
		t.Fatal("NumWritableNonSigners not equal")
	}
	for _, item := range msg.AccountKeys.Data {
		if item.String() != msgDecoded.AccountKeys.Data[0].String() {
			t.Fatal("AccountKeys not equal")
		}
	}
	for i, item := range msg.Instructions.Data {
		if item.ProgramIdIndex != msgDecoded.Instructions.Data[i].ProgramIdIndex {
			t.Fatal("Instructions not equal")
		}
		for j, item := range item.AccountIndexes.Data {
			if item != msgDecoded.Instructions.Data[i].AccountIndexes.Data[j] {
				t.Fatal("Instructions not equal")
			}
		}
		for j, item := range item.Data.Data {
			if item != msgDecoded.Instructions.Data[i].Data.Data[j] {
				t.Fatal("Instructions not equal")
			}
		}
	}
	for _, item := range msg.AddressTableLookups.Data {
		if item.AccountKey != msgDecoded.AddressTableLookups.Data[0].AccountKey {
			t.Fatal("AddressTableLookups not equal")
		}
		if !bytes.Equal(item.WritableIndexes.Data, msgDecoded.AddressTableLookups.Data[0].WritableIndexes.Data) {
			t.Fatal("AddressTableLookups not equal")
		}
		if !bytes.Equal(item.ReadonlyIndexes.Data, msgDecoded.AddressTableLookups.Data[0].ReadonlyIndexes.Data) {
			t.Fatal("AddressTableLookups not equal")
		}
	}

}
