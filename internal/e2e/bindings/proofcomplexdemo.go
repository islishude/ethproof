// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// ProofComplexDemoMetaData contains all meta data concerning the ProofComplexDemo contract.
var ProofComplexDemoMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"applyUpdate\",\"inputs\":[{\"name\":\"balanceValue\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"positionId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"historyValue\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"quantity\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"lastPrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nextNote\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"nextPayload\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"marker\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balances\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"historyAt\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"historyLength\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"noteText\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"payloadData\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"positionOf\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"positionId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"quantity\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"lastPrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"seedHistory\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"values\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ComplexStateUpdated\",\"inputs\":[{\"name\":\"caller\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"positionId\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"marker\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"balance\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"historyValue\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"quantity\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"lastPrice\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
	Bin: "0x6080604052348015600e575f5ffd5b506107648061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610085575f3560e01c8063c38896b411610058578063c38896b41461010d578063dc984dd414610120578063e5a96cd314610128578063fc721ad81461013b575f5ffd5b806327e235e3146100895780637c9b075f146100bb5780638f4eb9d7146100d0578063a3fd0d99146100e5575b5f5ffd5b6100a861009736600461040d565b5f6020819052908152604090205481565b6040519081526020015b60405180910390f35b6100ce6100c936600461046b565b610163565b005b6100d861022f565b6040516100b29190610543565b6100f86100f3366004610555565b6102bf565b604080519283526020830191909152016100b2565b6100a861011b366004610555565b6102f1565b6100d861032b565b6100ce61013636600461057d565b61033a565b6100a861014936600461040d565b6001600160a01b03165f9081526001602052604090205490565b335f818152602081815260408083208e9055600180835281842080548083018255908552838520018d9055815180830183528c81528084018c8152958552600284528285208f8652909352922090518155915191015560036101c6858783610696565b5060046101d4838583610696565b50604080518b8152602081018a90529081018890526060810187905281908a9033907f3f7e1690876a1dc29d310a315cac888fbddc6e61e53ca7ddbc6b1849ec0bbf099060800160405180910390a450505050505050505050565b60606003805461023e90610612565b80601f016020809104026020016040519081016040528092919081815260200182805461026a90610612565b80156102b55780601f1061028c576101008083540402835291602001916102b5565b820191905f5260205f20905b81548152906001019060200180831161029857829003601f168201915b5050505050905090565b6001600160a01b0382165f908152600260209081526040808320848452909152902080546001909101545b9250929050565b6001600160a01b0382165f90815260016020526040812080548390811061031a5761031a610750565b905f5260205f200154905092915050565b60606004805461023e90610612565b6001600160a01b0383165f90815260016020526040812061035a916103bc565b5f5b818110156103b6576001600160a01b0384165f90815260016020526040902083838381811061038d5761038d610750565b8354600180820186555f958652602095869020929095029390930135920191909155500161035c565b50505050565b5080545f8255905f5260205f20908101906103d791906103da565b50565b5b808211156103ee575f81556001016103db565b5090565b80356001600160a01b0381168114610408575f5ffd5b919050565b5f6020828403121561041d575f5ffd5b610426826103f2565b9392505050565b5f5f83601f84011261043d575f5ffd5b50813567ffffffffffffffff811115610454575f5ffd5b6020830191508360208285010111156102ea575f5ffd5b5f5f5f5f5f5f5f5f5f5f6101008b8d031215610485575f5ffd5b8a35995060208b0135985060408b0135975060608b0135965060808b0135955060a08b013567ffffffffffffffff8111156104be575f5ffd5b6104ca8d828e0161042d565b90965094505060c08b013567ffffffffffffffff8111156104e9575f5ffd5b6104f58d828e0161042d565b9150809450508092505060e08b013590509295989b9194979a5092959850565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b602081525f6104266020830184610515565b5f5f60408385031215610566575f5ffd5b61056f836103f2565b946020939093013593505050565b5f5f5f6040848603121561058f575f5ffd5b610598846103f2565b9250602084013567ffffffffffffffff8111156105b3575f5ffd5b8401601f810186136105c3575f5ffd5b803567ffffffffffffffff8111156105d9575f5ffd5b8660208260051b84010111156105ed575f5ffd5b939660209190910195509293505050565b634e487b7160e01b5f52604160045260245ffd5b600181811c9082168061062657607f821691505b60208210810361064457634e487b7160e01b5f52602260045260245ffd5b50919050565b601f82111561069157805f5260205f20601f840160051c8101602085101561066f5750805b601f840160051c820191505b8181101561068e575f815560010161067b565b50505b505050565b67ffffffffffffffff8311156106ae576106ae6105fe565b6106c2836106bc8354610612565b8361064a565b5f601f8411600181146106f3575f85156106dc5750838201355b5f19600387901b1c1916600186901b17835561068e565b5f83815260208120601f198716915b828110156107225786850135825560209485019460019092019101610702565b508682101561073e575f1960f88860031b161c19848701351681555b505060018560011b0183555050505050565b634e487b7160e01b5f52603260045260245ffd",
}

// ProofComplexDemoABI is the input ABI used to generate the binding from.
// Deprecated: Use ProofComplexDemoMetaData.ABI instead.
var ProofComplexDemoABI = ProofComplexDemoMetaData.ABI

// ProofComplexDemoBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ProofComplexDemoMetaData.Bin instead.
var ProofComplexDemoBin = ProofComplexDemoMetaData.Bin

// DeployProofComplexDemo deploys a new Ethereum contract, binding an instance of ProofComplexDemo to it.
func DeployProofComplexDemo(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ProofComplexDemo, error) {
	parsed, err := ProofComplexDemoMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ProofComplexDemoBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ProofComplexDemo{ProofComplexDemoCaller: ProofComplexDemoCaller{contract: contract}, ProofComplexDemoTransactor: ProofComplexDemoTransactor{contract: contract}, ProofComplexDemoFilterer: ProofComplexDemoFilterer{contract: contract}}, nil
}

// ProofComplexDemo is an auto generated Go binding around an Ethereum contract.
type ProofComplexDemo struct {
	ProofComplexDemoCaller     // Read-only binding to the contract
	ProofComplexDemoTransactor // Write-only binding to the contract
	ProofComplexDemoFilterer   // Log filterer for contract events
}

// ProofComplexDemoCaller is an auto generated read-only Go binding around an Ethereum contract.
type ProofComplexDemoCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProofComplexDemoTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ProofComplexDemoTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProofComplexDemoFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ProofComplexDemoFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProofComplexDemoSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ProofComplexDemoSession struct {
	Contract     *ProofComplexDemo // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ProofComplexDemoCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ProofComplexDemoCallerSession struct {
	Contract *ProofComplexDemoCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// ProofComplexDemoTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ProofComplexDemoTransactorSession struct {
	Contract     *ProofComplexDemoTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// ProofComplexDemoRaw is an auto generated low-level Go binding around an Ethereum contract.
type ProofComplexDemoRaw struct {
	Contract *ProofComplexDemo // Generic contract binding to access the raw methods on
}

// ProofComplexDemoCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ProofComplexDemoCallerRaw struct {
	Contract *ProofComplexDemoCaller // Generic read-only contract binding to access the raw methods on
}

// ProofComplexDemoTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ProofComplexDemoTransactorRaw struct {
	Contract *ProofComplexDemoTransactor // Generic write-only contract binding to access the raw methods on
}

// NewProofComplexDemo creates a new instance of ProofComplexDemo, bound to a specific deployed contract.
func NewProofComplexDemo(address common.Address, backend bind.ContractBackend) (*ProofComplexDemo, error) {
	contract, err := bindProofComplexDemo(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ProofComplexDemo{ProofComplexDemoCaller: ProofComplexDemoCaller{contract: contract}, ProofComplexDemoTransactor: ProofComplexDemoTransactor{contract: contract}, ProofComplexDemoFilterer: ProofComplexDemoFilterer{contract: contract}}, nil
}

// NewProofComplexDemoCaller creates a new read-only instance of ProofComplexDemo, bound to a specific deployed contract.
func NewProofComplexDemoCaller(address common.Address, caller bind.ContractCaller) (*ProofComplexDemoCaller, error) {
	contract, err := bindProofComplexDemo(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ProofComplexDemoCaller{contract: contract}, nil
}

// NewProofComplexDemoTransactor creates a new write-only instance of ProofComplexDemo, bound to a specific deployed contract.
func NewProofComplexDemoTransactor(address common.Address, transactor bind.ContractTransactor) (*ProofComplexDemoTransactor, error) {
	contract, err := bindProofComplexDemo(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ProofComplexDemoTransactor{contract: contract}, nil
}

// NewProofComplexDemoFilterer creates a new log filterer instance of ProofComplexDemo, bound to a specific deployed contract.
func NewProofComplexDemoFilterer(address common.Address, filterer bind.ContractFilterer) (*ProofComplexDemoFilterer, error) {
	contract, err := bindProofComplexDemo(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ProofComplexDemoFilterer{contract: contract}, nil
}

// bindProofComplexDemo binds a generic wrapper to an already deployed contract.
func bindProofComplexDemo(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ProofComplexDemoMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProofComplexDemo *ProofComplexDemoRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProofComplexDemo.Contract.ProofComplexDemoCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProofComplexDemo *ProofComplexDemoRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.ProofComplexDemoTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProofComplexDemo *ProofComplexDemoRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.ProofComplexDemoTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProofComplexDemo *ProofComplexDemoCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProofComplexDemo.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProofComplexDemo *ProofComplexDemoTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProofComplexDemo *ProofComplexDemoTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.contract.Transact(opts, method, params...)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoCaller) Balances(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "balances", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _ProofComplexDemo.Contract.Balances(&_ProofComplexDemo.CallOpts, arg0)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _ProofComplexDemo.Contract.Balances(&_ProofComplexDemo.CallOpts, arg0)
}

// HistoryAt is a free data retrieval call binding the contract method 0xc38896b4.
//
// Solidity: function historyAt(address user, uint256 index) view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoCaller) HistoryAt(opts *bind.CallOpts, user common.Address, index *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "historyAt", user, index)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HistoryAt is a free data retrieval call binding the contract method 0xc38896b4.
//
// Solidity: function historyAt(address user, uint256 index) view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoSession) HistoryAt(user common.Address, index *big.Int) (*big.Int, error) {
	return _ProofComplexDemo.Contract.HistoryAt(&_ProofComplexDemo.CallOpts, user, index)
}

// HistoryAt is a free data retrieval call binding the contract method 0xc38896b4.
//
// Solidity: function historyAt(address user, uint256 index) view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) HistoryAt(user common.Address, index *big.Int) (*big.Int, error) {
	return _ProofComplexDemo.Contract.HistoryAt(&_ProofComplexDemo.CallOpts, user, index)
}

// HistoryLength is a free data retrieval call binding the contract method 0xfc721ad8.
//
// Solidity: function historyLength(address user) view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoCaller) HistoryLength(opts *bind.CallOpts, user common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "historyLength", user)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HistoryLength is a free data retrieval call binding the contract method 0xfc721ad8.
//
// Solidity: function historyLength(address user) view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoSession) HistoryLength(user common.Address) (*big.Int, error) {
	return _ProofComplexDemo.Contract.HistoryLength(&_ProofComplexDemo.CallOpts, user)
}

// HistoryLength is a free data retrieval call binding the contract method 0xfc721ad8.
//
// Solidity: function historyLength(address user) view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) HistoryLength(user common.Address) (*big.Int, error) {
	return _ProofComplexDemo.Contract.HistoryLength(&_ProofComplexDemo.CallOpts, user)
}

// NoteText is a free data retrieval call binding the contract method 0x8f4eb9d7.
//
// Solidity: function noteText() view returns(string)
func (_ProofComplexDemo *ProofComplexDemoCaller) NoteText(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "noteText")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// NoteText is a free data retrieval call binding the contract method 0x8f4eb9d7.
//
// Solidity: function noteText() view returns(string)
func (_ProofComplexDemo *ProofComplexDemoSession) NoteText() (string, error) {
	return _ProofComplexDemo.Contract.NoteText(&_ProofComplexDemo.CallOpts)
}

// NoteText is a free data retrieval call binding the contract method 0x8f4eb9d7.
//
// Solidity: function noteText() view returns(string)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) NoteText() (string, error) {
	return _ProofComplexDemo.Contract.NoteText(&_ProofComplexDemo.CallOpts)
}

// PayloadData is a free data retrieval call binding the contract method 0xdc984dd4.
//
// Solidity: function payloadData() view returns(bytes)
func (_ProofComplexDemo *ProofComplexDemoCaller) PayloadData(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "payloadData")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// PayloadData is a free data retrieval call binding the contract method 0xdc984dd4.
//
// Solidity: function payloadData() view returns(bytes)
func (_ProofComplexDemo *ProofComplexDemoSession) PayloadData() ([]byte, error) {
	return _ProofComplexDemo.Contract.PayloadData(&_ProofComplexDemo.CallOpts)
}

// PayloadData is a free data retrieval call binding the contract method 0xdc984dd4.
//
// Solidity: function payloadData() view returns(bytes)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) PayloadData() ([]byte, error) {
	return _ProofComplexDemo.Contract.PayloadData(&_ProofComplexDemo.CallOpts)
}

// PositionOf is a free data retrieval call binding the contract method 0xa3fd0d99.
//
// Solidity: function positionOf(address user, uint256 positionId) view returns(uint256 quantity, uint256 lastPrice)
func (_ProofComplexDemo *ProofComplexDemoCaller) PositionOf(opts *bind.CallOpts, user common.Address, positionId *big.Int) (struct {
	Quantity  *big.Int
	LastPrice *big.Int
}, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "positionOf", user, positionId)

	outstruct := new(struct {
		Quantity  *big.Int
		LastPrice *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Quantity = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.LastPrice = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// PositionOf is a free data retrieval call binding the contract method 0xa3fd0d99.
//
// Solidity: function positionOf(address user, uint256 positionId) view returns(uint256 quantity, uint256 lastPrice)
func (_ProofComplexDemo *ProofComplexDemoSession) PositionOf(user common.Address, positionId *big.Int) (struct {
	Quantity  *big.Int
	LastPrice *big.Int
}, error) {
	return _ProofComplexDemo.Contract.PositionOf(&_ProofComplexDemo.CallOpts, user, positionId)
}

// PositionOf is a free data retrieval call binding the contract method 0xa3fd0d99.
//
// Solidity: function positionOf(address user, uint256 positionId) view returns(uint256 quantity, uint256 lastPrice)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) PositionOf(user common.Address, positionId *big.Int) (struct {
	Quantity  *big.Int
	LastPrice *big.Int
}, error) {
	return _ProofComplexDemo.Contract.PositionOf(&_ProofComplexDemo.CallOpts, user, positionId)
}

// ApplyUpdate is a paid mutator transaction binding the contract method 0x7c9b075f.
//
// Solidity: function applyUpdate(uint256 balanceValue, uint256 positionId, uint256 historyValue, uint256 quantity, uint256 lastPrice, string nextNote, bytes nextPayload, bytes32 marker) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactor) ApplyUpdate(opts *bind.TransactOpts, balanceValue *big.Int, positionId *big.Int, historyValue *big.Int, quantity *big.Int, lastPrice *big.Int, nextNote string, nextPayload []byte, marker [32]byte) (*types.Transaction, error) {
	return _ProofComplexDemo.contract.Transact(opts, "applyUpdate", balanceValue, positionId, historyValue, quantity, lastPrice, nextNote, nextPayload, marker)
}

// ApplyUpdate is a paid mutator transaction binding the contract method 0x7c9b075f.
//
// Solidity: function applyUpdate(uint256 balanceValue, uint256 positionId, uint256 historyValue, uint256 quantity, uint256 lastPrice, string nextNote, bytes nextPayload, bytes32 marker) returns()
func (_ProofComplexDemo *ProofComplexDemoSession) ApplyUpdate(balanceValue *big.Int, positionId *big.Int, historyValue *big.Int, quantity *big.Int, lastPrice *big.Int, nextNote string, nextPayload []byte, marker [32]byte) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.ApplyUpdate(&_ProofComplexDemo.TransactOpts, balanceValue, positionId, historyValue, quantity, lastPrice, nextNote, nextPayload, marker)
}

// ApplyUpdate is a paid mutator transaction binding the contract method 0x7c9b075f.
//
// Solidity: function applyUpdate(uint256 balanceValue, uint256 positionId, uint256 historyValue, uint256 quantity, uint256 lastPrice, string nextNote, bytes nextPayload, bytes32 marker) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactorSession) ApplyUpdate(balanceValue *big.Int, positionId *big.Int, historyValue *big.Int, quantity *big.Int, lastPrice *big.Int, nextNote string, nextPayload []byte, marker [32]byte) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.ApplyUpdate(&_ProofComplexDemo.TransactOpts, balanceValue, positionId, historyValue, quantity, lastPrice, nextNote, nextPayload, marker)
}

// SeedHistory is a paid mutator transaction binding the contract method 0xe5a96cd3.
//
// Solidity: function seedHistory(address user, uint256[] values) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactor) SeedHistory(opts *bind.TransactOpts, user common.Address, values []*big.Int) (*types.Transaction, error) {
	return _ProofComplexDemo.contract.Transact(opts, "seedHistory", user, values)
}

// SeedHistory is a paid mutator transaction binding the contract method 0xe5a96cd3.
//
// Solidity: function seedHistory(address user, uint256[] values) returns()
func (_ProofComplexDemo *ProofComplexDemoSession) SeedHistory(user common.Address, values []*big.Int) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SeedHistory(&_ProofComplexDemo.TransactOpts, user, values)
}

// SeedHistory is a paid mutator transaction binding the contract method 0xe5a96cd3.
//
// Solidity: function seedHistory(address user, uint256[] values) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactorSession) SeedHistory(user common.Address, values []*big.Int) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SeedHistory(&_ProofComplexDemo.TransactOpts, user, values)
}

// ProofComplexDemoComplexStateUpdatedIterator is returned from FilterComplexStateUpdated and is used to iterate over the raw logs and unpacked data for ComplexStateUpdated events raised by the ProofComplexDemo contract.
type ProofComplexDemoComplexStateUpdatedIterator struct {
	Event *ProofComplexDemoComplexStateUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ProofComplexDemoComplexStateUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProofComplexDemoComplexStateUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ProofComplexDemoComplexStateUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ProofComplexDemoComplexStateUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProofComplexDemoComplexStateUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProofComplexDemoComplexStateUpdated represents a ComplexStateUpdated event raised by the ProofComplexDemo contract.
type ProofComplexDemoComplexStateUpdated struct {
	Caller       common.Address
	PositionId   *big.Int
	Marker       [32]byte
	Balance      *big.Int
	HistoryValue *big.Int
	Quantity     *big.Int
	LastPrice    *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterComplexStateUpdated is a free log retrieval operation binding the contract event 0x3f7e1690876a1dc29d310a315cac888fbddc6e61e53ca7ddbc6b1849ec0bbf09.
//
// Solidity: event ComplexStateUpdated(address indexed caller, uint256 indexed positionId, bytes32 indexed marker, uint256 balance, uint256 historyValue, uint256 quantity, uint256 lastPrice)
func (_ProofComplexDemo *ProofComplexDemoFilterer) FilterComplexStateUpdated(opts *bind.FilterOpts, caller []common.Address, positionId []*big.Int, marker [][32]byte) (*ProofComplexDemoComplexStateUpdatedIterator, error) {

	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}
	var positionIdRule []interface{}
	for _, positionIdItem := range positionId {
		positionIdRule = append(positionIdRule, positionIdItem)
	}
	var markerRule []interface{}
	for _, markerItem := range marker {
		markerRule = append(markerRule, markerItem)
	}

	logs, sub, err := _ProofComplexDemo.contract.FilterLogs(opts, "ComplexStateUpdated", callerRule, positionIdRule, markerRule)
	if err != nil {
		return nil, err
	}
	return &ProofComplexDemoComplexStateUpdatedIterator{contract: _ProofComplexDemo.contract, event: "ComplexStateUpdated", logs: logs, sub: sub}, nil
}

// WatchComplexStateUpdated is a free log subscription operation binding the contract event 0x3f7e1690876a1dc29d310a315cac888fbddc6e61e53ca7ddbc6b1849ec0bbf09.
//
// Solidity: event ComplexStateUpdated(address indexed caller, uint256 indexed positionId, bytes32 indexed marker, uint256 balance, uint256 historyValue, uint256 quantity, uint256 lastPrice)
func (_ProofComplexDemo *ProofComplexDemoFilterer) WatchComplexStateUpdated(opts *bind.WatchOpts, sink chan<- *ProofComplexDemoComplexStateUpdated, caller []common.Address, positionId []*big.Int, marker [][32]byte) (event.Subscription, error) {

	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}
	var positionIdRule []interface{}
	for _, positionIdItem := range positionId {
		positionIdRule = append(positionIdRule, positionIdItem)
	}
	var markerRule []interface{}
	for _, markerItem := range marker {
		markerRule = append(markerRule, markerItem)
	}

	logs, sub, err := _ProofComplexDemo.contract.WatchLogs(opts, "ComplexStateUpdated", callerRule, positionIdRule, markerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProofComplexDemoComplexStateUpdated)
				if err := _ProofComplexDemo.contract.UnpackLog(event, "ComplexStateUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseComplexStateUpdated is a log parse operation binding the contract event 0x3f7e1690876a1dc29d310a315cac888fbddc6e61e53ca7ddbc6b1849ec0bbf09.
//
// Solidity: event ComplexStateUpdated(address indexed caller, uint256 indexed positionId, bytes32 indexed marker, uint256 balance, uint256 historyValue, uint256 quantity, uint256 lastPrice)
func (_ProofComplexDemo *ProofComplexDemoFilterer) ParseComplexStateUpdated(log types.Log) (*ProofComplexDemoComplexStateUpdated, error) {
	event := new(ProofComplexDemoComplexStateUpdated)
	if err := _ProofComplexDemo.contract.UnpackLog(event, "ComplexStateUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
