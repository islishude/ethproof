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

// ERC7201CustomLayoutDemoMetaData contains all meta data concerning the ERC7201CustomLayoutDemo contract.
var ERC7201CustomLayoutDemoMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"initialX\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"initialY\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"}]",
	Bin: "0x608034609257601f60ae38819003918201601f19168301916001600160401b03831184841017609657808492604094855283398101031260925760208151910151907f9f96f1285fecaf7ff903f9bcb9e24bb62cf61a391840765cfc133c40cd812700557f9f96f1285fecaf7ff903f9bcb9e24bb62cf61a391840765cfc133c40cd812701556040516003908160ab8239f35b5f80fd5b634e487b7160e01b5f52604160045260245ffdfe5f80fd",
}

// ERC7201CustomLayoutDemoABI is the input ABI used to generate the binding from.
// Deprecated: Use ERC7201CustomLayoutDemoMetaData.ABI instead.
var ERC7201CustomLayoutDemoABI = ERC7201CustomLayoutDemoMetaData.ABI

// ERC7201CustomLayoutDemoBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ERC7201CustomLayoutDemoMetaData.Bin instead.
var ERC7201CustomLayoutDemoBin = ERC7201CustomLayoutDemoMetaData.Bin

// DeployERC7201CustomLayoutDemo deploys a new Ethereum contract, binding an instance of ERC7201CustomLayoutDemo to it.
func DeployERC7201CustomLayoutDemo(auth *bind.TransactOpts, backend bind.ContractBackend, initialX *big.Int, initialY *big.Int) (common.Address, *types.Transaction, *ERC7201CustomLayoutDemo, error) {
	parsed, err := ERC7201CustomLayoutDemoMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ERC7201CustomLayoutDemoBin), backend, initialX, initialY)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ERC7201CustomLayoutDemo{ERC7201CustomLayoutDemoCaller: ERC7201CustomLayoutDemoCaller{contract: contract}, ERC7201CustomLayoutDemoTransactor: ERC7201CustomLayoutDemoTransactor{contract: contract}, ERC7201CustomLayoutDemoFilterer: ERC7201CustomLayoutDemoFilterer{contract: contract}}, nil
}

// ERC7201CustomLayoutDemo is an auto generated Go binding around an Ethereum contract.
type ERC7201CustomLayoutDemo struct {
	ERC7201CustomLayoutDemoCaller     // Read-only binding to the contract
	ERC7201CustomLayoutDemoTransactor // Write-only binding to the contract
	ERC7201CustomLayoutDemoFilterer   // Log filterer for contract events
}

// ERC7201CustomLayoutDemoCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC7201CustomLayoutDemoCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC7201CustomLayoutDemoTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC7201CustomLayoutDemoTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC7201CustomLayoutDemoFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC7201CustomLayoutDemoFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC7201CustomLayoutDemoSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC7201CustomLayoutDemoSession struct {
	Contract     *ERC7201CustomLayoutDemo // Generic contract binding to set the session for
	CallOpts     bind.CallOpts            // Call options to use throughout this session
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ERC7201CustomLayoutDemoCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC7201CustomLayoutDemoCallerSession struct {
	Contract *ERC7201CustomLayoutDemoCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                  // Call options to use throughout this session
}

// ERC7201CustomLayoutDemoTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC7201CustomLayoutDemoTransactorSession struct {
	Contract     *ERC7201CustomLayoutDemoTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                  // Transaction auth options to use throughout this session
}

// ERC7201CustomLayoutDemoRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC7201CustomLayoutDemoRaw struct {
	Contract *ERC7201CustomLayoutDemo // Generic contract binding to access the raw methods on
}

// ERC7201CustomLayoutDemoCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC7201CustomLayoutDemoCallerRaw struct {
	Contract *ERC7201CustomLayoutDemoCaller // Generic read-only contract binding to access the raw methods on
}

// ERC7201CustomLayoutDemoTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC7201CustomLayoutDemoTransactorRaw struct {
	Contract *ERC7201CustomLayoutDemoTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC7201CustomLayoutDemo creates a new instance of ERC7201CustomLayoutDemo, bound to a specific deployed contract.
func NewERC7201CustomLayoutDemo(address common.Address, backend bind.ContractBackend) (*ERC7201CustomLayoutDemo, error) {
	contract, err := bindERC7201CustomLayoutDemo(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC7201CustomLayoutDemo{ERC7201CustomLayoutDemoCaller: ERC7201CustomLayoutDemoCaller{contract: contract}, ERC7201CustomLayoutDemoTransactor: ERC7201CustomLayoutDemoTransactor{contract: contract}, ERC7201CustomLayoutDemoFilterer: ERC7201CustomLayoutDemoFilterer{contract: contract}}, nil
}

// NewERC7201CustomLayoutDemoCaller creates a new read-only instance of ERC7201CustomLayoutDemo, bound to a specific deployed contract.
func NewERC7201CustomLayoutDemoCaller(address common.Address, caller bind.ContractCaller) (*ERC7201CustomLayoutDemoCaller, error) {
	contract, err := bindERC7201CustomLayoutDemo(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC7201CustomLayoutDemoCaller{contract: contract}, nil
}

// NewERC7201CustomLayoutDemoTransactor creates a new write-only instance of ERC7201CustomLayoutDemo, bound to a specific deployed contract.
func NewERC7201CustomLayoutDemoTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC7201CustomLayoutDemoTransactor, error) {
	contract, err := bindERC7201CustomLayoutDemo(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC7201CustomLayoutDemoTransactor{contract: contract}, nil
}

// NewERC7201CustomLayoutDemoFilterer creates a new log filterer instance of ERC7201CustomLayoutDemo, bound to a specific deployed contract.
func NewERC7201CustomLayoutDemoFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC7201CustomLayoutDemoFilterer, error) {
	contract, err := bindERC7201CustomLayoutDemo(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC7201CustomLayoutDemoFilterer{contract: contract}, nil
}

// bindERC7201CustomLayoutDemo binds a generic wrapper to an already deployed contract.
func bindERC7201CustomLayoutDemo(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ERC7201CustomLayoutDemoMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC7201CustomLayoutDemo *ERC7201CustomLayoutDemoRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC7201CustomLayoutDemo.Contract.ERC7201CustomLayoutDemoCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC7201CustomLayoutDemo *ERC7201CustomLayoutDemoRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC7201CustomLayoutDemo.Contract.ERC7201CustomLayoutDemoTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC7201CustomLayoutDemo *ERC7201CustomLayoutDemoRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC7201CustomLayoutDemo.Contract.ERC7201CustomLayoutDemoTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC7201CustomLayoutDemo *ERC7201CustomLayoutDemoCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC7201CustomLayoutDemo.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC7201CustomLayoutDemo *ERC7201CustomLayoutDemoTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC7201CustomLayoutDemo.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC7201CustomLayoutDemo *ERC7201CustomLayoutDemoTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC7201CustomLayoutDemo.Contract.contract.Transact(opts, method, params...)
}
