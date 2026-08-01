// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

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
	_ = time.Tick
	_ = context.Background
)

// ProofComplexDemoMetaData contains all meta data concerning the ProofComplexDemo contract.
var ProofComplexDemoMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"applyUpdate\",\"inputs\":[{\"name\":\"balanceValue\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"positionId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"historyValue\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"quantity\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"lastPrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nextNote\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"nextPayload\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"marker\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balances\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"basicAddress\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"basicBool\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"basicBytes32\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"basicUint256\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"historyAt\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"historyLength\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"noteText\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"payloadData\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"positionOf\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"positionId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"quantity\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"lastPrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"seedHistory\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"values\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setBasicSamples\",\"inputs\":[{\"name\":\"nextBasicUint256\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nextBasicAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"nextBasicBytes32\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nextBasicBool\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setFixedSmallArray\",\"inputs\":[{\"name\":\"nextFixed0\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"nextFixed1\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"nextFixed2\",\"type\":\"uint128\",\"internalType\":\"uint128\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setMixedTriplet\",\"inputs\":[{\"name\":\"nextMixedA\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"nextMixedB\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"nextMixedC\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setPackedTriplet\",\"inputs\":[{\"name\":\"nextPackedA\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"nextPackedB\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"nextPackedC\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ComplexStateUpdated\",\"inputs\":[{\"name\":\"caller\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"positionId\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"marker\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"balance\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"historyValue\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"quantity\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"lastPrice\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
	Bin: "0x60808060405234601557610c36908161001a8239f35b5f80fdfe6080806040526004361015610012575f80fd5b5f3560e01c90816327e235e314610a7b575080633ef803d014610a045780634844fd1f146109e25780635dbf3ec4146109bc5780637c9b075f146106195780638f4eb9d7146105745780639871a22f146104fa578063a3fd0d99146104ad578063b4a14edc14610490578063c38896b414610447578063c5081c3e1461038b578063c6ba22981461036e578063d5d24110146102b0578063dc984dd4146101ce578063e5a96cd3146101085763fc721ad8146100cc575f80fd5b34610104576020366003190112610104576001600160a01b036100ed610aae565b165f526001602052602060405f2054604051908152f35b5f80fd5b3461010457604036600319011261010457610121610aae565b6024359067ffffffffffffffff821161010457366023830112156101045781600401359067ffffffffffffffff8211610104573660248360051b85010111610104576001600160a01b0316805f52600160205260405f2080545f8255806101b8575b50505f5b828110156101b657600190825f52816020526101b060405f2060248360051b8801013590610b6c565b01610187565b005b6101c7915f5260205f20610c1b565b8380610183565b34610104575f366003190112610104576040515f6004546101ee81610be3565b808452906001811690811561028c575060011461022e575b61022a8361021681850382610bc1565b604051918291602083526020830190610b08565b0390f35b60045f9081527f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b939250905b80821061027257509091508101602001610216610206565b91926001816020925483858801015201910190929161025a565b60ff191660208086019190915291151560051b840190910191506102169050610206565b34610104576060366003190112610104576102c9610ac4565b6102d1610b2c565b604435918260406001600160801b038151936102ec85610ba5565b169283815267ffffffffffffffff8516602082015201526fffffffffffffffffffffffffffffffff19600a541617600a557fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff77ffffffffffffffff0000000000000000000000000000000080600a549360801b1616911617600a55600b555f80f35b34610104575f366003190112610104576020600554604051908152f35b34610104576060366003190112610104576103a4610ac4565b6103ac610b2c565b6044359167ffffffffffffffff83168084036101045760406001600160801b038151936103d885610ba5565b1680845267ffffffffffffffff851660208501529201526001600160801b031660809190911b77ffffffffffffffff00000000000000000000000000000000161760c09190911b7fffffffffffffffff0000000000000000000000000000000000000000000000001617600955005b34610104576040366003190112610104576001600160a01b03610468610aae565b165f526001602052602061048160243560405f20610b43565b90549060031b1c604051908152f35b34610104575f366003190112610104576020600754604051908152f35b34610104576040366003190112610104576001600160a01b036104ce610aae565b165f52600260205260405f206024355f526020526040805f206001815491015482519182526020820152f35b34610104576080366003190112610104576024356001600160a01b0381168091036101045760643590811515809203610104576004356005557fffffffffffffffffffffffff0000000000000000000000000000000000000000600654161760065560443560075560ff8019600854169116176008555f80f35b34610104575f366003190112610104576040515f60035461059481610be3565b808452906001811690811561028c57506001146105bb5761022a8361021681850382610bc1565b60035f9081527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b939250905b8082106105ff57509091508101602001610216610206565b9192600181602092548385880101520191019092916105e7565b34610104576101003660031901126101045760043560243560843560643560443560a43567ffffffffffffffff81116101045761065a903690600401610ada565b9060c43567ffffffffffffffff81116101045761067b903690600401610ada565b929091335f525f6020528860405f2055335f5260016020526106a08560405f20610b6c565b6040516040810181811067ffffffffffffffff8211176108c257604052868152600160208201898152335f52600260205260405f208b5f5260205260405f20925183555191015567ffffffffffffffff82116108c257610701600354610be3565b601f8111610964575b505f90601f83116001146108e15761073992915f91836108d6575b50508160011b915f199060031b1c19161790565b6003555b67ffffffffffffffff82116108c257610757600454610be3565b601f8111610861575b505f90601f83116001146107de5761078e92915f91836107d35750508160011b915f199060031b1c19161790565b6004555b60405194855260208501526040840152606083015260e435917f3f7e1690876a1dc29d310a315cac888fbddc6e61e53ca7ddbc6b1849ec0bbf0960803392a4005b013590508880610725565b601f1983169160045f527f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b925f5b8181106108495750908460019594939210610830575b505050811b01600455610792565b01355f19600384901b60f8161c19169055878080610822565b9193602060018192878701358155019501920161080c565b828111156107605760045f526108b4907f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b90601f850160051c90602086106108ba575b601f82910160051c039101610c1b565b87610760565b5f91506108a4565b634e487b7160e01b5f52604160045260245ffd5b013590508a80610725565b601f1983169160035f527fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b925f5b81811061094c5750908460019594939210610933575b505050811b0160035561073d565b01355f19600384901b60f8161c19169055898080610925565b9193602060018192878701358155019501920161090f565b8281111561070a5760035f526109b6907fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b90601f850160051c90602086106108ba57601f82910160051c039101610c1b565b8961070a565b34610104575f3660031901126101045760206001600160a01b0360065416604051908152f35b34610104575f36600319011261010457602060ff600854166040519015158152f35b3461010457606036600319011261010457610a1d610ac4565b6024356001600160801b038116810361010457604435906001600160801b03821682036101045760801b6fffffffffffffffffffffffffffffffff199081166001600160801b0393841617600c55600d805490911691909216179055005b34610104576020366003190112610104576020906001600160a01b03610a9f610aae565b165f525f825260405f20548152f35b600435906001600160a01b038216820361010457565b600435906001600160801b038216820361010457565b9181601f840112156101045782359167ffffffffffffffff8311610104576020838186019501011161010457565b805180835260209291819084018484015e5f828201840152601f01601f1916010190565b6024359067ffffffffffffffff8216820361010457565b8054821015610b58575f5260205f2001905f90565b634e487b7160e01b5f52603260045260245ffd5b8054680100000000000000008110156108c257610b8e91600182018155610b43565b819291549060031b91821b915f19901b1916179055565b6060810190811067ffffffffffffffff8211176108c257604052565b90601f8019910116810190811067ffffffffffffffff8211176108c257604052565b90600182811c92168015610c11575b6020831014610bfd57565b634e487b7160e01b5f52602260045260245ffd5b91607f1691610bf2565b5f5b828110610c2957505050565b5f82820155600101610c1d56",
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

// BasicAddress is a free data retrieval call binding the contract method 0x5dbf3ec4.
//
// Solidity: function basicAddress() view returns(address)
func (_ProofComplexDemo *ProofComplexDemoCaller) BasicAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "basicAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BasicAddress is a free data retrieval call binding the contract method 0x5dbf3ec4.
//
// Solidity: function basicAddress() view returns(address)
func (_ProofComplexDemo *ProofComplexDemoSession) BasicAddress() (common.Address, error) {
	return _ProofComplexDemo.Contract.BasicAddress(&_ProofComplexDemo.CallOpts)
}

// BasicAddress is a free data retrieval call binding the contract method 0x5dbf3ec4.
//
// Solidity: function basicAddress() view returns(address)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) BasicAddress() (common.Address, error) {
	return _ProofComplexDemo.Contract.BasicAddress(&_ProofComplexDemo.CallOpts)
}

// BasicBool is a free data retrieval call binding the contract method 0x4844fd1f.
//
// Solidity: function basicBool() view returns(bool)
func (_ProofComplexDemo *ProofComplexDemoCaller) BasicBool(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "basicBool")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// BasicBool is a free data retrieval call binding the contract method 0x4844fd1f.
//
// Solidity: function basicBool() view returns(bool)
func (_ProofComplexDemo *ProofComplexDemoSession) BasicBool() (bool, error) {
	return _ProofComplexDemo.Contract.BasicBool(&_ProofComplexDemo.CallOpts)
}

// BasicBool is a free data retrieval call binding the contract method 0x4844fd1f.
//
// Solidity: function basicBool() view returns(bool)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) BasicBool() (bool, error) {
	return _ProofComplexDemo.Contract.BasicBool(&_ProofComplexDemo.CallOpts)
}

// BasicBytes32 is a free data retrieval call binding the contract method 0xb4a14edc.
//
// Solidity: function basicBytes32() view returns(bytes32)
func (_ProofComplexDemo *ProofComplexDemoCaller) BasicBytes32(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "basicBytes32")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// BasicBytes32 is a free data retrieval call binding the contract method 0xb4a14edc.
//
// Solidity: function basicBytes32() view returns(bytes32)
func (_ProofComplexDemo *ProofComplexDemoSession) BasicBytes32() ([32]byte, error) {
	return _ProofComplexDemo.Contract.BasicBytes32(&_ProofComplexDemo.CallOpts)
}

// BasicBytes32 is a free data retrieval call binding the contract method 0xb4a14edc.
//
// Solidity: function basicBytes32() view returns(bytes32)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) BasicBytes32() ([32]byte, error) {
	return _ProofComplexDemo.Contract.BasicBytes32(&_ProofComplexDemo.CallOpts)
}

// BasicUint256 is a free data retrieval call binding the contract method 0xc6ba2298.
//
// Solidity: function basicUint256() view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoCaller) BasicUint256(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ProofComplexDemo.contract.Call(opts, &out, "basicUint256")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BasicUint256 is a free data retrieval call binding the contract method 0xc6ba2298.
//
// Solidity: function basicUint256() view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoSession) BasicUint256() (*big.Int, error) {
	return _ProofComplexDemo.Contract.BasicUint256(&_ProofComplexDemo.CallOpts)
}

// BasicUint256 is a free data retrieval call binding the contract method 0xc6ba2298.
//
// Solidity: function basicUint256() view returns(uint256)
func (_ProofComplexDemo *ProofComplexDemoCallerSession) BasicUint256() (*big.Int, error) {
	return _ProofComplexDemo.Contract.BasicUint256(&_ProofComplexDemo.CallOpts)
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

// SetBasicSamples is a paid mutator transaction binding the contract method 0x9871a22f.
//
// Solidity: function setBasicSamples(uint256 nextBasicUint256, address nextBasicAddress, bytes32 nextBasicBytes32, bool nextBasicBool) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactor) SetBasicSamples(opts *bind.TransactOpts, nextBasicUint256 *big.Int, nextBasicAddress common.Address, nextBasicBytes32 [32]byte, nextBasicBool bool) (*types.Transaction, error) {
	return _ProofComplexDemo.contract.Transact(opts, "setBasicSamples", nextBasicUint256, nextBasicAddress, nextBasicBytes32, nextBasicBool)
}

// SetBasicSamples is a paid mutator transaction binding the contract method 0x9871a22f.
//
// Solidity: function setBasicSamples(uint256 nextBasicUint256, address nextBasicAddress, bytes32 nextBasicBytes32, bool nextBasicBool) returns()
func (_ProofComplexDemo *ProofComplexDemoSession) SetBasicSamples(nextBasicUint256 *big.Int, nextBasicAddress common.Address, nextBasicBytes32 [32]byte, nextBasicBool bool) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SetBasicSamples(&_ProofComplexDemo.TransactOpts, nextBasicUint256, nextBasicAddress, nextBasicBytes32, nextBasicBool)
}

// SetBasicSamples is a paid mutator transaction binding the contract method 0x9871a22f.
//
// Solidity: function setBasicSamples(uint256 nextBasicUint256, address nextBasicAddress, bytes32 nextBasicBytes32, bool nextBasicBool) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactorSession) SetBasicSamples(nextBasicUint256 *big.Int, nextBasicAddress common.Address, nextBasicBytes32 [32]byte, nextBasicBool bool) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SetBasicSamples(&_ProofComplexDemo.TransactOpts, nextBasicUint256, nextBasicAddress, nextBasicBytes32, nextBasicBool)
}

// SetFixedSmallArray is a paid mutator transaction binding the contract method 0x3ef803d0.
//
// Solidity: function setFixedSmallArray(uint128 nextFixed0, uint128 nextFixed1, uint128 nextFixed2) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactor) SetFixedSmallArray(opts *bind.TransactOpts, nextFixed0 *big.Int, nextFixed1 *big.Int, nextFixed2 *big.Int) (*types.Transaction, error) {
	return _ProofComplexDemo.contract.Transact(opts, "setFixedSmallArray", nextFixed0, nextFixed1, nextFixed2)
}

// SetFixedSmallArray is a paid mutator transaction binding the contract method 0x3ef803d0.
//
// Solidity: function setFixedSmallArray(uint128 nextFixed0, uint128 nextFixed1, uint128 nextFixed2) returns()
func (_ProofComplexDemo *ProofComplexDemoSession) SetFixedSmallArray(nextFixed0 *big.Int, nextFixed1 *big.Int, nextFixed2 *big.Int) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SetFixedSmallArray(&_ProofComplexDemo.TransactOpts, nextFixed0, nextFixed1, nextFixed2)
}

// SetFixedSmallArray is a paid mutator transaction binding the contract method 0x3ef803d0.
//
// Solidity: function setFixedSmallArray(uint128 nextFixed0, uint128 nextFixed1, uint128 nextFixed2) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactorSession) SetFixedSmallArray(nextFixed0 *big.Int, nextFixed1 *big.Int, nextFixed2 *big.Int) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SetFixedSmallArray(&_ProofComplexDemo.TransactOpts, nextFixed0, nextFixed1, nextFixed2)
}

// SetMixedTriplet is a paid mutator transaction binding the contract method 0xd5d24110.
//
// Solidity: function setMixedTriplet(uint128 nextMixedA, uint64 nextMixedB, bytes32 nextMixedC) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactor) SetMixedTriplet(opts *bind.TransactOpts, nextMixedA *big.Int, nextMixedB uint64, nextMixedC [32]byte) (*types.Transaction, error) {
	return _ProofComplexDemo.contract.Transact(opts, "setMixedTriplet", nextMixedA, nextMixedB, nextMixedC)
}

// SetMixedTriplet is a paid mutator transaction binding the contract method 0xd5d24110.
//
// Solidity: function setMixedTriplet(uint128 nextMixedA, uint64 nextMixedB, bytes32 nextMixedC) returns()
func (_ProofComplexDemo *ProofComplexDemoSession) SetMixedTriplet(nextMixedA *big.Int, nextMixedB uint64, nextMixedC [32]byte) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SetMixedTriplet(&_ProofComplexDemo.TransactOpts, nextMixedA, nextMixedB, nextMixedC)
}

// SetMixedTriplet is a paid mutator transaction binding the contract method 0xd5d24110.
//
// Solidity: function setMixedTriplet(uint128 nextMixedA, uint64 nextMixedB, bytes32 nextMixedC) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactorSession) SetMixedTriplet(nextMixedA *big.Int, nextMixedB uint64, nextMixedC [32]byte) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SetMixedTriplet(&_ProofComplexDemo.TransactOpts, nextMixedA, nextMixedB, nextMixedC)
}

// SetPackedTriplet is a paid mutator transaction binding the contract method 0xc5081c3e.
//
// Solidity: function setPackedTriplet(uint128 nextPackedA, uint64 nextPackedB, uint64 nextPackedC) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactor) SetPackedTriplet(opts *bind.TransactOpts, nextPackedA *big.Int, nextPackedB uint64, nextPackedC uint64) (*types.Transaction, error) {
	return _ProofComplexDemo.contract.Transact(opts, "setPackedTriplet", nextPackedA, nextPackedB, nextPackedC)
}

// SetPackedTriplet is a paid mutator transaction binding the contract method 0xc5081c3e.
//
// Solidity: function setPackedTriplet(uint128 nextPackedA, uint64 nextPackedB, uint64 nextPackedC) returns()
func (_ProofComplexDemo *ProofComplexDemoSession) SetPackedTriplet(nextPackedA *big.Int, nextPackedB uint64, nextPackedC uint64) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SetPackedTriplet(&_ProofComplexDemo.TransactOpts, nextPackedA, nextPackedB, nextPackedC)
}

// SetPackedTriplet is a paid mutator transaction binding the contract method 0xc5081c3e.
//
// Solidity: function setPackedTriplet(uint128 nextPackedA, uint64 nextPackedB, uint64 nextPackedC) returns()
func (_ProofComplexDemo *ProofComplexDemoTransactorSession) SetPackedTriplet(nextPackedA *big.Int, nextPackedB uint64, nextPackedC uint64) (*types.Transaction, error) {
	return _ProofComplexDemo.Contract.SetPackedTriplet(&_ProofComplexDemo.TransactOpts, nextPackedA, nextPackedB, nextPackedC)
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
