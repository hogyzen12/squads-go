package multisigtransaction

import (
	"github.com/gagliardetto/solana-go"
	addresslookuptable "github.com/gagliardetto/solana-go/programs/address-lookup-table"
)

type CompiledKeyMeta struct {
	IsSigner   bool `json:"isSigner"`
	IsWritable bool `json:"isWritable"`
	IsInvoked  bool `json:"isInvoked"`
}

type CompiledKeys struct {
	Payer      solana.PublicKey           `json:"payer"`
	KeyMetaMap map[string]CompiledKeyMeta `json:"keyMetaMap"`
}

type AccountKeysFromLookups struct {
	Writable []solana.PublicKey `json:"writable"`
	Readonly []solana.PublicKey `json:"readonly"`
}

type MessageV0 struct {
	Header               solana.MessageHeader               `json:"header"`
	StaticAccountKeys    []solana.PublicKey                 `json:"staticAccountKeys"`
	RecentBlockhash      solana.Hash                        `json:"recentBlockhash"`
	CompiledInstructions []solana.CompiledInstruction       `json:"compiledInstructions"`
	AddressTableLookups  []solana.MessageAddressTableLookup `json:"addressTableLookups"`
}

type MessageAccountKeys struct {
	StaticAccountKeys      []solana.PublicKey     `json:"staticAccountKeys"`
	AccountKeysFromLookups AccountKeysFromLookups `json:"accountKeysFromLookups"`
}

func NewCompiledKeys(payer solana.PublicKey, keyMetaMap map[string]CompiledKeyMeta) *CompiledKeys {
	return &CompiledKeys{
		Payer:      payer,
		KeyMetaMap: keyMetaMap,
	}
}

func CompileKeys(instructions []solana.Instruction, payer solana.PublicKey) *CompiledKeys {
	keyMetaMap := make(map[string]CompiledKeyMeta)

	getOrInsertDefault := func(pubkey solana.PublicKey) *CompiledKeyMeta {
		address := pubkey.String()
		if keyMeta, exists := keyMetaMap[address]; exists {
			return &keyMeta
		}

		keyMeta := CompiledKeyMeta{
			IsSigner:   false,
			IsWritable: false,
			IsInvoked:  false,
		}
		keyMetaMap[address] = keyMeta
		return &keyMeta
	}

	payerKeyMeta := getOrInsertDefault(payer)
	payerKeyMeta.IsSigner = true
	payerKeyMeta.IsWritable = true
	keyMetaMap[payer.String()] = *payerKeyMeta

	for _, ix := range instructions {
		programKeyMeta := getOrInsertDefault(ix.ProgramID())
		programKeyMeta.IsInvoked = false
		keyMetaMap[ix.ProgramID().String()] = *programKeyMeta

		for _, accountMeta := range ix.Accounts() {
			keyMeta := getOrInsertDefault(accountMeta.PublicKey)
			keyMeta.IsSigner = keyMeta.IsSigner || accountMeta.IsSigner
			keyMeta.IsWritable = keyMeta.IsWritable || accountMeta.IsWritable
			keyMetaMap[accountMeta.PublicKey.String()] = *keyMeta
		}
	}

	return NewCompiledKeys(payer, keyMetaMap)
}

func (ck *CompiledKeys) GetMessageComponents() (solana.MessageHeader, []solana.PublicKey) {
	var writableSigners, readonlySigners, writableNonSigners, readonlyNonSigners []string

	for address, meta := range ck.KeyMetaMap {
		if meta.IsSigner && meta.IsWritable {
			writableSigners = append(writableSigners, address)
		} else if meta.IsSigner && !meta.IsWritable {
			readonlySigners = append(readonlySigners, address)
		} else if !meta.IsSigner && meta.IsWritable {
			writableNonSigners = append(writableNonSigners, address)
		} else {
			readonlyNonSigners = append(readonlyNonSigners, address)
		}
	}

	header := solana.MessageHeader{
		NumRequiredSignatures:       uint8(len(writableSigners) + len(readonlySigners)),
		NumReadonlySignedAccounts:   uint8(len(readonlySigners)),
		NumReadonlyUnsignedAccounts: uint8(len(readonlyNonSigners)),
	}

	var staticAccountKeys []solana.PublicKey

	for _, address := range writableSigners {
		pubkey, _ := solana.PublicKeyFromBase58(address)
		staticAccountKeys = append(staticAccountKeys, pubkey)
	}

	for _, address := range readonlySigners {
		pubkey, _ := solana.PublicKeyFromBase58(address)
		staticAccountKeys = append(staticAccountKeys, pubkey)
	}

	for _, address := range writableNonSigners {
		pubkey, _ := solana.PublicKeyFromBase58(address)
		staticAccountKeys = append(staticAccountKeys, pubkey)
	}

	for _, address := range readonlyNonSigners {
		pubkey, _ := solana.PublicKeyFromBase58(address)
		staticAccountKeys = append(staticAccountKeys, pubkey)
	}

	return header, staticAccountKeys
}

func (ck *CompiledKeys) ExtractTableLookup(lookupTable addresslookuptable.KeyedAddressLookupTable) (*solana.MessageAddressTableLookup, *AccountKeysFromLookups, bool) {
	writableIndexes, drainedWritableKeys := ck.drainKeysFoundInLookupTable(
		lookupTable.State.Addresses,
		func(keyMeta CompiledKeyMeta) bool {
			return !keyMeta.IsSigner && !keyMeta.IsInvoked && keyMeta.IsWritable
		},
	)

	readonlyIndexes, drainedReadonlyKeys := ck.drainKeysFoundInLookupTable(
		lookupTable.State.Addresses,
		func(keyMeta CompiledKeyMeta) bool {
			return !keyMeta.IsSigner && !keyMeta.IsInvoked && !keyMeta.IsWritable
		},
	)

	if len(writableIndexes) == 0 && len(readonlyIndexes) == 0 {
		return nil, nil, false
	}

	return &solana.MessageAddressTableLookup{
			AccountKey:      lookupTable.Key,
			WritableIndexes: writableIndexes,
			ReadonlyIndexes: readonlyIndexes,
		},
		&AccountKeysFromLookups{
			Writable: drainedWritableKeys,
			Readonly: drainedReadonlyKeys,
		},
		true
}

func (ck *CompiledKeys) drainKeysFoundInLookupTable(lookupTableEntries []solana.PublicKey, keyMetaFilter func(CompiledKeyMeta) bool) ([]uint8, []solana.PublicKey) {
	var lookupTableIndexes []uint8
	var drainedKeys []solana.PublicKey

	for address, keyMeta := range ck.KeyMetaMap {
		if keyMetaFilter(keyMeta) {
			key, _ := solana.PublicKeyFromBase58(address)

			for i, entry := range lookupTableEntries {
				if entry.Equals(key) {
					lookupTableIndexes = append(lookupTableIndexes, uint8(i))
					drainedKeys = append(drainedKeys, key)
					delete(ck.KeyMetaMap, address)
					break
				}
			}
		}
	}

	return lookupTableIndexes, drainedKeys
}

func (mk *MessageAccountKeys) CompileInstructions(instructions []solana.Instruction) []solana.CompiledInstruction {
	accountIndexMap := make(map[string]uint16)
	index := uint16(0)

	for _, key := range mk.StaticAccountKeys {
		accountIndexMap[key.String()] = index
		index++
	}

	for _, key := range mk.AccountKeysFromLookups.Writable {
		accountIndexMap[key.String()] = index
		index++
	}

	for _, key := range mk.AccountKeysFromLookups.Readonly {
		accountIndexMap[key.String()] = index
		index++
	}

	var compiledInstructions []solana.CompiledInstruction

	for _, instruction := range instructions {
		programIDIndex := accountIndexMap[instruction.ProgramID().String()]

		var accounts []uint16
		for _, accountMeta := range instruction.Accounts() {
			accountIndex := accountIndexMap[accountMeta.PublicKey.String()]
			accounts = append(accounts, accountIndex)
		}

		instructionData, _ := instruction.Data()
		compiledInstructions = append(compiledInstructions, solana.CompiledInstruction{
			ProgramIDIndex: programIDIndex,
			Accounts:       accounts,
			Data:           instructionData,
		})
	}

	return compiledInstructions
}

func CompileToWrappedMessageV0(payerKey solana.PublicKey,
	recentBlockhash solana.Hash,
	instructions []solana.Instruction,
	addressLookupTableAccounts []addresslookuptable.KeyedAddressLookupTable) *solana.Message {

	compiledKeys := CompileKeys(instructions, payerKey)

	var addressTableLookups []solana.MessageAddressTableLookup
	accountKeysFromLookups := AccountKeysFromLookups{
		Writable: []solana.PublicKey{},
		Readonly: []solana.PublicKey{},
	}

	for _, lookupTable := range addressLookupTableAccounts {
		if lookup, keys, found := compiledKeys.ExtractTableLookup(lookupTable); found {
			addressTableLookups = append(addressTableLookups, *lookup)
			accountKeysFromLookups.Writable = append(accountKeysFromLookups.Writable, keys.Writable...)
			accountKeysFromLookups.Readonly = append(accountKeysFromLookups.Readonly, keys.Readonly...)
		}
	}

	header, staticAccountKeys := compiledKeys.GetMessageComponents()

	accountKeys := &MessageAccountKeys{
		StaticAccountKeys:      staticAccountKeys,
		AccountKeysFromLookups: accountKeysFromLookups,
	}

	compiledInstructions := accountKeys.CompileInstructions(instructions)
	messageV0 := solana.Message{
		Header:              header,
		AccountKeys:         staticAccountKeys,
		RecentBlockhash:     recentBlockhash,
		Instructions:        compiledInstructions,
		AddressTableLookups: solana.MessageAddressTableLookupSlice(addressTableLookups),
	}
	messageV0.SetVersion(solana.MessageVersionV0)
	return &messageV0
}
