package proof

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ResolvedStorageSlot describes one concrete storage slot and byte range.
type ResolvedStorageSlot struct {
	Slot   common.Hash `json:"slot"`
	Offset uint64      `json:"offset"`
	Bytes  uint64      `json:"bytes"`
	Label  string      `json:"label"`
	Type   string      `json:"type"`
}

// StorageSlotResolution is the result of resolving a Solidity storage query.
type StorageSlotResolution struct {
	HeadSlot  common.Hash           `json:"headSlot"`
	Encoding  string                `json:"encoding"`
	TypeLabel string                `json:"typeLabel"`
	Slots     []ResolvedStorageSlot `json:"slots"`
}

// ResolveStorageSlots resolves a Solidity storage-layout query into concrete slot metadata.
func ResolveStorageSlots(layout *StorageLayout, query string) (*StorageSlotResolution, error) {
	if layout == nil {
		return nil, fmt.Errorf("storage layout is required")
	}
	parsed, err := parseStorageQuery(query)
	if err != nil {
		return nil, err
	}

	entry, ok := findStorageLayoutEntry(layout.Storage, parsed.root)
	if !ok {
		return nil, fmt.Errorf("unknown variable %q", parsed.root)
	}
	rootSlot, err := parseStorageSlot(entry.Slot)
	if err != nil {
		return nil, fmt.Errorf("parse slot for %s: %w", entry.Label, err)
	}

	resolver := storageResolver{
		layout: layout,
		cache:  make(map[string]*big.Int),
	}
	cursor := storageCursor{
		slot:   rootSlot,
		offset: entry.Offset,
		typeID: entry.Type,
		path:   entry.Label,
	}
	for _, step := range parsed.steps {
		switch step.kind {
		case storageQueryMember:
			cursor, err = resolver.resolveMember(cursor, step.value)
		case storageQuerySubscript:
			cursor, err = resolver.resolveSubscript(cursor, step.value)
		default:
			err = fmt.Errorf("unsupported query step")
		}
		if err != nil {
			return nil, err
		}
	}

	currentType, err := resolver.typeInfo(cursor.typeID)
	if err != nil {
		return nil, err
	}
	headSlot, err := slotBigIntToHash(cursor.slot)
	if err != nil {
		return nil, fmt.Errorf("encode head slot for %s: %w", cursor.path, err)
	}

	result := &StorageSlotResolution{
		HeadSlot:  headSlot,
		Encoding:  currentType.Encoding,
		TypeLabel: currentType.Label,
	}
	if parsed.wordIndex != nil {
		if currentType.Encoding != "bytes" {
			return nil, fmt.Errorf("@word is only valid for bytes/string queries")
		}
		wordSlot := new(big.Int).Add(hashStorageSlot(cursor.slot), parsed.wordIndex)
		slotHash, err := slotBigIntToHash(wordSlot)
		if err != nil {
			return nil, fmt.Errorf("encode data word slot for %s: %w", cursor.path, err)
		}
		result.Slots = []ResolvedStorageSlot{{
			Slot:   slotHash,
			Offset: 0,
			Bytes:  32,
			Label:  fmt.Sprintf("%s@word(%s)", cursor.path, parsed.wordIndex.String()),
			Type:   currentType.Label,
		}}
		return result, nil
	}

	slots, err := resolver.expand(cursor)
	if err != nil {
		return nil, err
	}
	result.Slots = slots
	return result, nil
}

type storageResolver struct {
	layout *StorageLayout
	cache  map[string]*big.Int
}

type storageCursor struct {
	slot   *big.Int
	offset uint64
	typeID string
	path   string
}

func (r *storageResolver) resolveMember(cursor storageCursor, memberName string) (storageCursor, error) {
	typeInfo, err := r.typeInfo(cursor.typeID)
	if err != nil {
		return storageCursor{}, err
	}
	if !isStructType(typeInfo) {
		return storageCursor{}, fmt.Errorf("type %s does not support member access", typeInfo.Label)
	}
	member, ok := findStorageLayoutEntry(typeInfo.Members, memberName)
	if !ok {
		return storageCursor{}, fmt.Errorf("unknown member %q on %s", memberName, cursor.path)
	}
	memberSlot, err := parseStorageSlot(member.Slot)
	if err != nil {
		return storageCursor{}, fmt.Errorf("parse member slot for %s.%s: %w", cursor.path, member.Label, err)
	}
	return storageCursor{
		slot:   new(big.Int).Add(cursor.slot, memberSlot),
		offset: member.Offset,
		typeID: member.Type,
		path:   cursor.path + "." + member.Label,
	}, nil
}

func (r *storageResolver) resolveSubscript(cursor storageCursor, raw string) (storageCursor, error) {
	typeInfo, err := r.typeInfo(cursor.typeID)
	if err != nil {
		return storageCursor{}, err
	}
	switch {
	case typeInfo.Encoding == "mapping":
		return r.resolveMappingKey(cursor, raw, typeInfo)
	case isStaticArrayType(cursor.typeID, typeInfo), typeInfo.Encoding == "dynamic_array":
		return r.resolveArrayIndex(cursor, raw, typeInfo)
	default:
		return storageCursor{}, fmt.Errorf("type %s does not support indexing", typeInfo.Label)
	}
}

func (r *storageResolver) resolveMappingKey(cursor storageCursor, raw string, typeInfo StorageLayoutType) (storageCursor, error) {
	keyType, err := r.typeInfo(typeInfo.Key)
	if err != nil {
		return storageCursor{}, err
	}
	encodedKey, err := encodeMappingKey(typeInfo.Key, keyType, raw)
	if err != nil {
		return storageCursor{}, fmt.Errorf("invalid mapping key %q for %s: %w", raw, typeInfo.Label, err)
	}

	slotBytes, err := slotBigIntToBytes32(cursor.slot)
	if err != nil {
		return storageCursor{}, err
	}
	hashInput := append(encodedKey, slotBytes...)
	nextSlot := new(big.Int).SetBytes(crypto.Keccak256(hashInput))
	return storageCursor{
		slot:   nextSlot,
		offset: 0,
		typeID: typeInfo.Value,
		path:   cursor.path + "[" + strings.TrimSpace(raw) + "]",
	}, nil
}

func (r *storageResolver) resolveArrayIndex(cursor storageCursor, raw string, typeInfo StorageLayoutType) (storageCursor, error) {
	index, err := parseQueryInteger(raw)
	if err != nil {
		return storageCursor{}, fmt.Errorf("parse index %q for %s: %w", raw, cursor.path, err)
	}
	if typeInfo.Base == "" {
		return storageCursor{}, fmt.Errorf("array type %s is missing base type", typeInfo.Label)
	}

	baseType, err := r.typeInfo(typeInfo.Base)
	if err != nil {
		return storageCursor{}, err
	}

	baseSlot := new(big.Int).Set(cursor.slot)
	if typeInfo.Encoding == "dynamic_array" {
		baseSlot = hashStorageSlot(cursor.slot)
	} else {
		length, ok := staticArrayLength(cursor.typeID)
		if !ok {
			return storageCursor{}, fmt.Errorf("could not determine static array length for %s", typeInfo.Label)
		}
		if index.Cmp(length) >= 0 {
			return storageCursor{}, fmt.Errorf("index %s out of bounds for %s length %s", index.String(), cursor.path, length.String())
		}
	}

	labelIndex := strings.TrimSpace(raw)
	if canPackArrayElements(typeInfo.Base, baseType) {
		elementBytes, err := decimalStringUint64(baseType.NumberOfBytes)
		if err != nil {
			return storageCursor{}, fmt.Errorf("parse array element size for %s: %w", baseType.Label, err)
		}
		perSlot := uint64(32 / elementBytes)
		slotOffset := new(big.Int).Div(new(big.Int).Set(index), new(big.Int).SetUint64(perSlot))
		intraSlot := new(big.Int).Mod(new(big.Int).Set(index), new(big.Int).SetUint64(perSlot))
		return storageCursor{
			slot:   new(big.Int).Add(baseSlot, slotOffset),
			offset: intraSlot.Uint64() * elementBytes,
			typeID: typeInfo.Base,
			path:   cursor.path + "[" + labelIndex + "]",
		}, nil
	}

	stride, err := r.slotsOccupied(typeInfo.Base)
	if err != nil {
		return storageCursor{}, err
	}
	slotOffset := new(big.Int).Mul(new(big.Int).Set(index), stride)
	return storageCursor{
		slot:   new(big.Int).Add(baseSlot, slotOffset),
		offset: 0,
		typeID: typeInfo.Base,
		path:   cursor.path + "[" + labelIndex + "]",
	}, nil
}

func (r *storageResolver) expand(cursor storageCursor) ([]ResolvedStorageSlot, error) {
	typeInfo, err := r.typeInfo(cursor.typeID)
	if err != nil {
		return nil, err
	}
	switch {
	case typeInfo.Encoding == "mapping":
		return nil, fmt.Errorf("mapping %s requires explicit key selector", cursor.path)
	case typeInfo.Encoding == "dynamic_array":
		return nil, fmt.Errorf("dynamic array %s requires explicit index selector", cursor.path)
	case typeInfo.Encoding == "bytes":
		slotHash, err := slotBigIntToHash(cursor.slot)
		if err != nil {
			return nil, err
		}
		return []ResolvedStorageSlot{{
			Slot:   slotHash,
			Offset: 0,
			Bytes:  32,
			Label:  cursor.path,
			Type:   typeInfo.Label,
		}}, nil
	case isStructType(typeInfo):
		var out []ResolvedStorageSlot
		for _, member := range typeInfo.Members {
			memberSlot, err := parseStorageSlot(member.Slot)
			if err != nil {
				return nil, fmt.Errorf("parse member slot for %s.%s: %w", cursor.path, member.Label, err)
			}
			child := storageCursor{
				slot:   new(big.Int).Add(cursor.slot, memberSlot),
				offset: member.Offset,
				typeID: member.Type,
				path:   cursor.path + "." + member.Label,
			}
			childSlots, err := r.expand(child)
			if err != nil {
				return nil, err
			}
			out = append(out, childSlots...)
		}
		return out, nil
	case isStaticArrayType(cursor.typeID, typeInfo):
		length, ok := staticArrayLength(cursor.typeID)
		if !ok {
			return nil, fmt.Errorf("could not determine static array length for %s", cursor.path)
		}
		if !length.IsInt64() || length.Int64() < 0 || length.Int64() > math.MaxInt/2 {
			return nil, fmt.Errorf("static array %s is too large to expand", cursor.path)
		}
		var out []ResolvedStorageSlot
		for i := int64(0); i < length.Int64(); i++ {
			child, err := r.resolveArrayIndex(cursor, strconv.FormatInt(i, 10), typeInfo)
			if err != nil {
				return nil, err
			}
			childSlots, err := r.expand(child)
			if err != nil {
				return nil, err
			}
			out = append(out, childSlots...)
		}
		return out, nil
	default:
		byteLen, err := decimalStringUint64(typeInfo.NumberOfBytes)
		if err != nil {
			return nil, fmt.Errorf("parse byte size for %s: %w", cursor.path, err)
		}
		slotHash, err := slotBigIntToHash(cursor.slot)
		if err != nil {
			return nil, err
		}
		return []ResolvedStorageSlot{{
			Slot:   slotHash,
			Offset: cursor.offset,
			Bytes:  byteLen,
			Label:  cursor.path,
			Type:   typeInfo.Label,
		}}, nil
	}
}

func (r *storageResolver) slotsOccupied(typeID string) (*big.Int, error) {
	if cached, ok := r.cache[typeID]; ok {
		return new(big.Int).Set(cached), nil
	}

	typeInfo, err := r.typeInfo(typeID)
	if err != nil {
		return nil, err
	}

	var slots *big.Int
	switch {
	case typeInfo.Encoding == "mapping", typeInfo.Encoding == "dynamic_array", typeInfo.Encoding == "bytes":
		slots = big.NewInt(1)
	case isStructType(typeInfo):
		maxSlots := big.NewInt(0)
		for _, member := range typeInfo.Members {
			memberOffset, err := parseStorageSlot(member.Slot)
			if err != nil {
				return nil, fmt.Errorf("parse member slot for %s.%s: %w", typeInfo.Label, member.Label, err)
			}
			memberSlots, err := r.slotsOccupied(member.Type)
			if err != nil {
				return nil, err
			}
			end := new(big.Int).Add(memberOffset, memberSlots)
			if end.Cmp(maxSlots) > 0 {
				maxSlots = end
			}
		}
		if maxSlots.Sign() == 0 {
			maxSlots = big.NewInt(1)
		}
		slots = maxSlots
	case isStaticArrayType(typeID, typeInfo):
		length, ok := staticArrayLength(typeID)
		if !ok {
			return nil, fmt.Errorf("could not determine static array length for %s", typeInfo.Label)
		}
		baseType, err := r.typeInfo(typeInfo.Base)
		if err != nil {
			return nil, err
		}
		if canPackArrayElements(typeInfo.Base, baseType) {
			elementBytes, err := decimalStringUint64(baseType.NumberOfBytes)
			if err != nil {
				return nil, fmt.Errorf("parse array element size for %s: %w", baseType.Label, err)
			}
			perSlot := big.NewInt(int64(32 / elementBytes))
			slots = new(big.Int).Div(new(big.Int).Add(new(big.Int).Set(length), new(big.Int).Sub(perSlot, big.NewInt(1))), perSlot)
		} else {
			baseSlots, err := r.slotsOccupied(typeInfo.Base)
			if err != nil {
				return nil, err
			}
			slots = new(big.Int).Mul(length, baseSlots)
		}
	default:
		slots = big.NewInt(1)
	}

	r.cache[typeID] = new(big.Int).Set(slots)
	return new(big.Int).Set(slots), nil
}

func (r *storageResolver) typeInfo(typeID string) (StorageLayoutType, error) {
	typeInfo, ok := r.layout.Types[typeID]
	if !ok {
		return StorageLayoutType{}, fmt.Errorf("unknown storage type %q", typeID)
	}
	return typeInfo, nil
}

type storageQueryKind int

const (
	storageQueryMember storageQueryKind = iota + 1
	storageQuerySubscript
)

type storageQuery struct {
	root      string
	steps     []storageQueryStep
	wordIndex *big.Int
}

type storageQueryStep struct {
	kind  storageQueryKind
	value string
}

func parseStorageQuery(query string) (storageQuery, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return storageQuery{}, fmt.Errorf("storage query is empty")
	}

	root, pos, err := readQueryIdentifier(query, 0)
	if err != nil {
		return storageQuery{}, err
	}
	out := storageQuery{root: root}
	for pos < len(query) {
		switch query[pos] {
		case '.':
			name, next, err := readQueryIdentifier(query, pos+1)
			if err != nil {
				return storageQuery{}, err
			}
			out.steps = append(out.steps, storageQueryStep{kind: storageQueryMember, value: name})
			pos = next
		case '[':
			end := strings.IndexByte(query[pos+1:], ']')
			if end < 0 {
				return storageQuery{}, fmt.Errorf("unterminated [ in storage query")
			}
			token := strings.TrimSpace(query[pos+1 : pos+1+end])
			if token == "" {
				return storageQuery{}, fmt.Errorf("empty [] selector in storage query")
			}
			out.steps = append(out.steps, storageQueryStep{kind: storageQuerySubscript, value: token})
			pos += end + 2
		case '@':
			if !strings.HasPrefix(query[pos:], "@word(") {
				return storageQuery{}, fmt.Errorf("unsupported suffix in storage query")
			}
			end := strings.IndexByte(query[pos+6:], ')')
			if end < 0 {
				return storageQuery{}, fmt.Errorf("unterminated @word(...) suffix in storage query")
			}
			wordIndex, err := parseQueryInteger(query[pos+6 : pos+6+end])
			if err != nil {
				return storageQuery{}, fmt.Errorf("parse @word index: %w", err)
			}
			out.wordIndex = wordIndex
			pos += end + 7
			if pos != len(query) {
				return storageQuery{}, fmt.Errorf("@word(...) must be the final query suffix")
			}
		default:
			return storageQuery{}, fmt.Errorf("unexpected character %q in storage query", query[pos])
		}
	}
	return out, nil
}

func readQueryIdentifier(query string, start int) (string, int, error) {
	if start >= len(query) {
		return "", 0, fmt.Errorf("expected identifier in storage query")
	}
	r, width := utf8.DecodeRuneInString(query[start:])
	if r == utf8.RuneError && width == 1 {
		return "", 0, fmt.Errorf("invalid identifier in storage query")
	}
	if r != '_' && !unicode.IsLetter(r) {
		return "", 0, fmt.Errorf("expected identifier in storage query")
	}
	pos := start + width
	for pos < len(query) {
		r, width = utf8.DecodeRuneInString(query[pos:])
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			pos += width
			continue
		}
		break
	}
	return query[start:pos], pos, nil
}

func findStorageLayoutEntry(entries []StorageLayoutEntry, label string) (StorageLayoutEntry, bool) {
	for _, entry := range entries {
		if entry.Label == label {
			return entry, true
		}
	}
	return StorageLayoutEntry{}, false
}

func parseStorageSlot(raw string) (*big.Int, error) {
	value := strings.TrimSpace(raw)
	slot, ok := new(big.Int).SetString(value, 0)
	if !ok {
		return nil, fmt.Errorf("invalid storage slot %q", raw)
	}
	if slot.Sign() < 0 {
		return nil, fmt.Errorf("storage slot must be non-negative")
	}
	return slot, nil
}

func hashStorageSlot(slot *big.Int) *big.Int {
	b, _ := slotBigIntToBytes32(slot)
	return new(big.Int).SetBytes(crypto.Keccak256(b))
}

func slotBigIntToHash(slot *big.Int) (common.Hash, error) {
	b, err := slotBigIntToBytes32(slot)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(b), nil
}

func slotBigIntToBytes32(slot *big.Int) ([]byte, error) {
	if slot == nil {
		return nil, fmt.Errorf("storage slot is nil")
	}
	if slot.Sign() < 0 {
		return nil, fmt.Errorf("storage slot must be non-negative")
	}
	if slot.BitLen() > 256 {
		return nil, fmt.Errorf("storage slot exceeds 256 bits")
	}
	out := make([]byte, 32)
	raw := slot.Bytes()
	copy(out[32-len(raw):], raw)
	return out, nil
}

func isStructType(typeInfo StorageLayoutType) bool {
	return len(typeInfo.Members) > 0
}

func isStaticArrayType(typeID string, typeInfo StorageLayoutType) bool {
	_, ok := staticArrayLength(typeID)
	return ok && typeInfo.Encoding == "inplace" && typeInfo.Base != ""
}

func staticArrayLength(typeID string) (*big.Int, bool) {
	if !strings.HasPrefix(typeID, "t_array(") || !strings.HasSuffix(typeID, "_storage") {
		return nil, false
	}
	trimmed := strings.TrimSuffix(typeID, "_storage")
	lastParen := strings.LastIndex(trimmed, ")")
	if lastParen < 0 || lastParen == len(trimmed)-1 {
		return nil, false
	}
	suffix := trimmed[lastParen+1:]
	if suffix == "dyn" {
		return nil, false
	}
	length, ok := new(big.Int).SetString(suffix, 10)
	if !ok {
		return nil, false
	}
	return length, true
}

func canPackArrayElements(typeID string, typeInfo StorageLayoutType) bool {
	if typeInfo.Encoding != "inplace" || isStructType(typeInfo) || isStaticArrayType(typeID, typeInfo) {
		return false
	}
	byteLen, err := decimalStringUint64(typeInfo.NumberOfBytes)
	if err != nil {
		return false
	}
	return byteLen > 0 && byteLen <= 16
}

func decimalStringUint64(raw string) (uint64, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func parseQueryInteger(raw string) (*big.Int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, fmt.Errorf("empty integer")
	}

	sign := 1
	switch {
	case strings.HasPrefix(value, "+"):
		value = value[1:]
	case strings.HasPrefix(value, "-"):
		sign = -1
		value = value[1:]
	}
	if value == "" {
		return nil, fmt.Errorf("empty integer")
	}

	base := 10
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		base = 16
		value = value[2:]
	}
	if value == "" {
		return nil, fmt.Errorf("empty integer")
	}

	out, ok := new(big.Int).SetString(value, base)
	if !ok {
		return nil, fmt.Errorf("invalid integer")
	}
	if sign < 0 {
		out.Neg(out)
	}
	if out.Sign() < 0 {
		return nil, fmt.Errorf("integer must be non-negative")
	}
	return out, nil
}

func encodeMappingKey(typeID string, typeInfo StorageLayoutType, raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	switch {
	case typeInfo.Label == "bool":
		switch raw {
		case "true":
			return leftPadBytes([]byte{1}), nil
		case "false":
			return make([]byte, 32), nil
		default:
			return nil, fmt.Errorf("expected bool literal")
		}
	case typeInfo.Label == "string":
		value, err := parseStringLiteral(raw)
		if err != nil {
			return nil, err
		}
		return []byte(value), nil
	case typeInfo.Label == "bytes":
		if !strings.HasPrefix(raw, "0x") && !strings.HasPrefix(raw, "0X") {
			return nil, fmt.Errorf("expected hex bytes literal")
		}
		return common.FromHex(raw), nil
	case strings.HasPrefix(typeInfo.Label, "bytes"):
		size, err := fixedBytesSize(typeInfo.Label)
		if err != nil {
			return nil, err
		}
		value := common.FromHex(raw)
		if len(value) != size {
			return nil, fmt.Errorf("expected %d-byte hex literal", size)
		}
		return rightPadBytes(value), nil
	case typeInfo.Label == "address" || strings.HasPrefix(typeInfo.Label, "contract "):
		if !common.IsHexAddress(raw) {
			return nil, fmt.Errorf("expected address literal")
		}
		return leftPadBytes(common.HexToAddress(raw).Bytes()), nil
	case strings.HasPrefix(typeInfo.Label, "uint"):
		value, err := parseSignedIntegerLiteral(raw)
		if err != nil {
			return nil, err
		}
		if value.Sign() < 0 {
			return nil, fmt.Errorf("uint key must be non-negative")
		}
		byteLen, err := decimalStringUint64(typeInfo.NumberOfBytes)
		if err != nil {
			return nil, err
		}
		if value.BitLen() > int(byteLen*8) {
			return nil, fmt.Errorf("value exceeds %s", typeInfo.Label)
		}
		return leftPadBytes(value.Bytes()), nil
	case strings.HasPrefix(typeInfo.Label, "int"):
		value, err := parseSignedIntegerLiteral(raw)
		if err != nil {
			return nil, err
		}
		byteLen, err := decimalStringUint64(typeInfo.NumberOfBytes)
		if err != nil {
			return nil, err
		}
		return encodeSignedIntKey(value, byteLen)
	default:
		return nil, fmt.Errorf("unsupported mapping key type %s (%s)", typeInfo.Label, typeID)
	}
}

func parseStringLiteral(raw string) (string, error) {
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", fmt.Errorf("invalid quoted string")
		}
		return value, nil
	}
	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		return raw[1 : len(raw)-1], nil
	}
	return raw, nil
}

func parseSignedIntegerLiteral(raw string) (*big.Int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty integer")
	}
	sign := 1
	switch {
	case strings.HasPrefix(raw, "+"):
		raw = raw[1:]
	case strings.HasPrefix(raw, "-"):
		sign = -1
		raw = raw[1:]
	}
	if raw == "" {
		return nil, fmt.Errorf("empty integer")
	}
	base := 10
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		base = 16
		raw = raw[2:]
	}
	value, ok := new(big.Int).SetString(raw, base)
	if !ok {
		return nil, fmt.Errorf("invalid integer")
	}
	if sign < 0 {
		value.Neg(value)
	}
	return value, nil
}

func encodeSignedIntKey(value *big.Int, byteLen uint64) ([]byte, error) {
	if byteLen == 0 || byteLen > 32 {
		return nil, fmt.Errorf("invalid signed integer width")
	}
	bits := byteLen * 8
	lowerBound := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), uint(bits-1)))
	upperBound := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(bits-1)), big.NewInt(1))
	if value.Cmp(lowerBound) < 0 || value.Cmp(upperBound) > 0 {
		return nil, fmt.Errorf("value exceeds int%d", bits)
	}
	if value.Sign() >= 0 {
		return leftPadBytes(value.Bytes()), nil
	}
	modulus := new(big.Int).Lsh(big.NewInt(1), 256)
	encoded := new(big.Int).Add(modulus, value)
	return leftPadBytes(encoded.Bytes()), nil
}

func fixedBytesSize(label string) (int, error) {
	if label == "bytes" {
		return 0, fmt.Errorf("dynamic bytes is not a fixed-bytes type")
	}
	size, err := strconv.Atoi(strings.TrimPrefix(label, "bytes"))
	if err != nil || size < 1 || size > 32 {
		return 0, fmt.Errorf("unsupported fixed bytes type %s", label)
	}
	return size, nil
}

func leftPadBytes(in []byte) []byte {
	out := make([]byte, 32)
	copy(out[32-len(in):], in)
	return out
}

func rightPadBytes(in []byte) []byte {
	out := make([]byte, 32)
	copy(out, in)
	return out
}
