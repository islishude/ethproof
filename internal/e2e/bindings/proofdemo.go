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

// ProofDemoMetaData contains all meta data concerning the ProofDemo contract.
var ProofDemoMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"setValue\",\"inputs\":[{\"name\":\"newValue\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"marker\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"value\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"ValueUpdated\",\"inputs\":[{\"name\":\"caller\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"marker\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
	Bin: "0x6080604052348015600e575f5ffd5b506101d68061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c80633fa4f2451461003857806390686ce114610056575b5f5ffd5b610040610072565b60405161004d91906100e8565b60405180910390f35b610070600480360381019061006b9190610162565b610077565b005b5f5481565b815f81905550803373ffffffffffffffffffffffffffffffffffffffff167f30b70f4f4e3f9a7e61df8ccb304723c5ec51f959a9fab821eff10d6b00eba5b4846040516100c491906100e8565b60405180910390a35050565b5f819050919050565b6100e2816100d0565b82525050565b5f6020820190506100fb5f8301846100d9565b92915050565b5f5ffd5b61010e816100d0565b8114610118575f5ffd5b50565b5f8135905061012981610105565b92915050565b5f819050919050565b6101418161012f565b811461014b575f5ffd5b50565b5f8135905061015c81610138565b92915050565b5f5f6040838503121561017857610177610101565b5b5f6101858582860161011b565b92505060206101968582860161014e565b915050925092905056fea26469706673582212205874e58bf77b9ad6ff558e238a87284ab3fa93338776b02b10dc21cf3509c51164736f6c634300081c0033",
}

// ProofDemoABI is the input ABI used to generate the binding from.
// Deprecated: Use ProofDemoMetaData.ABI instead.
var ProofDemoABI = ProofDemoMetaData.ABI

// ProofDemoBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ProofDemoMetaData.Bin instead.
var ProofDemoBin = ProofDemoMetaData.Bin

// DeployProofDemo deploys a new Ethereum contract, binding an instance of ProofDemo to it.
func DeployProofDemo(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ProofDemo, error) {
	parsed, err := ProofDemoMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ProofDemoBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ProofDemo{ProofDemoCaller: ProofDemoCaller{contract: contract}, ProofDemoTransactor: ProofDemoTransactor{contract: contract}, ProofDemoFilterer: ProofDemoFilterer{contract: contract}}, nil
}

// ProofDemo is an auto generated Go binding around an Ethereum contract.
type ProofDemo struct {
	ProofDemoCaller     // Read-only binding to the contract
	ProofDemoTransactor // Write-only binding to the contract
	ProofDemoFilterer   // Log filterer for contract events
}

// ProofDemoCaller is an auto generated read-only Go binding around an Ethereum contract.
type ProofDemoCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProofDemoTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ProofDemoTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProofDemoFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ProofDemoFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProofDemoSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ProofDemoSession struct {
	Contract     *ProofDemo        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ProofDemoCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ProofDemoCallerSession struct {
	Contract *ProofDemoCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// ProofDemoTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ProofDemoTransactorSession struct {
	Contract     *ProofDemoTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ProofDemoRaw is an auto generated low-level Go binding around an Ethereum contract.
type ProofDemoRaw struct {
	Contract *ProofDemo // Generic contract binding to access the raw methods on
}

// ProofDemoCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ProofDemoCallerRaw struct {
	Contract *ProofDemoCaller // Generic read-only contract binding to access the raw methods on
}

// ProofDemoTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ProofDemoTransactorRaw struct {
	Contract *ProofDemoTransactor // Generic write-only contract binding to access the raw methods on
}

// NewProofDemo creates a new instance of ProofDemo, bound to a specific deployed contract.
func NewProofDemo(address common.Address, backend bind.ContractBackend) (*ProofDemo, error) {
	contract, err := bindProofDemo(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ProofDemo{ProofDemoCaller: ProofDemoCaller{contract: contract}, ProofDemoTransactor: ProofDemoTransactor{contract: contract}, ProofDemoFilterer: ProofDemoFilterer{contract: contract}}, nil
}

// NewProofDemoCaller creates a new read-only instance of ProofDemo, bound to a specific deployed contract.
func NewProofDemoCaller(address common.Address, caller bind.ContractCaller) (*ProofDemoCaller, error) {
	contract, err := bindProofDemo(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ProofDemoCaller{contract: contract}, nil
}

// NewProofDemoTransactor creates a new write-only instance of ProofDemo, bound to a specific deployed contract.
func NewProofDemoTransactor(address common.Address, transactor bind.ContractTransactor) (*ProofDemoTransactor, error) {
	contract, err := bindProofDemo(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ProofDemoTransactor{contract: contract}, nil
}

// NewProofDemoFilterer creates a new log filterer instance of ProofDemo, bound to a specific deployed contract.
func NewProofDemoFilterer(address common.Address, filterer bind.ContractFilterer) (*ProofDemoFilterer, error) {
	contract, err := bindProofDemo(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ProofDemoFilterer{contract: contract}, nil
}

// bindProofDemo binds a generic wrapper to an already deployed contract.
func bindProofDemo(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ProofDemoMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProofDemo *ProofDemoRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProofDemo.Contract.ProofDemoCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProofDemo *ProofDemoRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProofDemo.Contract.ProofDemoTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProofDemo *ProofDemoRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProofDemo.Contract.ProofDemoTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProofDemo *ProofDemoCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProofDemo.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProofDemo *ProofDemoTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProofDemo.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProofDemo *ProofDemoTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProofDemo.Contract.contract.Transact(opts, method, params...)
}

// Value is a free data retrieval call binding the contract method 0x3fa4f245.
//
// Solidity: function value() view returns(uint256)
func (_ProofDemo *ProofDemoCaller) Value(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ProofDemo.contract.Call(opts, &out, "value")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Value is a free data retrieval call binding the contract method 0x3fa4f245.
//
// Solidity: function value() view returns(uint256)
func (_ProofDemo *ProofDemoSession) Value() (*big.Int, error) {
	return _ProofDemo.Contract.Value(&_ProofDemo.CallOpts)
}

// Value is a free data retrieval call binding the contract method 0x3fa4f245.
//
// Solidity: function value() view returns(uint256)
func (_ProofDemo *ProofDemoCallerSession) Value() (*big.Int, error) {
	return _ProofDemo.Contract.Value(&_ProofDemo.CallOpts)
}

// SetValue is a paid mutator transaction binding the contract method 0x90686ce1.
//
// Solidity: function setValue(uint256 newValue, bytes32 marker) returns()
func (_ProofDemo *ProofDemoTransactor) SetValue(opts *bind.TransactOpts, newValue *big.Int, marker [32]byte) (*types.Transaction, error) {
	return _ProofDemo.contract.Transact(opts, "setValue", newValue, marker)
}

// SetValue is a paid mutator transaction binding the contract method 0x90686ce1.
//
// Solidity: function setValue(uint256 newValue, bytes32 marker) returns()
func (_ProofDemo *ProofDemoSession) SetValue(newValue *big.Int, marker [32]byte) (*types.Transaction, error) {
	return _ProofDemo.Contract.SetValue(&_ProofDemo.TransactOpts, newValue, marker)
}

// SetValue is a paid mutator transaction binding the contract method 0x90686ce1.
//
// Solidity: function setValue(uint256 newValue, bytes32 marker) returns()
func (_ProofDemo *ProofDemoTransactorSession) SetValue(newValue *big.Int, marker [32]byte) (*types.Transaction, error) {
	return _ProofDemo.Contract.SetValue(&_ProofDemo.TransactOpts, newValue, marker)
}

// ProofDemoValueUpdatedIterator is returned from FilterValueUpdated and is used to iterate over the raw logs and unpacked data for ValueUpdated events raised by the ProofDemo contract.
type ProofDemoValueUpdatedIterator struct {
	Event *ProofDemoValueUpdated // Event containing the contract specifics and raw log

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
func (it *ProofDemoValueUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProofDemoValueUpdated)
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
		it.Event = new(ProofDemoValueUpdated)
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
func (it *ProofDemoValueUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProofDemoValueUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProofDemoValueUpdated represents a ValueUpdated event raised by the ProofDemo contract.
type ProofDemoValueUpdated struct {
	Caller common.Address
	Marker [32]byte
	Value  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterValueUpdated is a free log retrieval operation binding the contract event 0x30b70f4f4e3f9a7e61df8ccb304723c5ec51f959a9fab821eff10d6b00eba5b4.
//
// Solidity: event ValueUpdated(address indexed caller, bytes32 indexed marker, uint256 value)
func (_ProofDemo *ProofDemoFilterer) FilterValueUpdated(opts *bind.FilterOpts, caller []common.Address, marker [][32]byte) (*ProofDemoValueUpdatedIterator, error) {

	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}
	var markerRule []interface{}
	for _, markerItem := range marker {
		markerRule = append(markerRule, markerItem)
	}

	logs, sub, err := _ProofDemo.contract.FilterLogs(opts, "ValueUpdated", callerRule, markerRule)
	if err != nil {
		return nil, err
	}
	return &ProofDemoValueUpdatedIterator{contract: _ProofDemo.contract, event: "ValueUpdated", logs: logs, sub: sub}, nil
}

// WatchValueUpdated is a free log subscription operation binding the contract event 0x30b70f4f4e3f9a7e61df8ccb304723c5ec51f959a9fab821eff10d6b00eba5b4.
//
// Solidity: event ValueUpdated(address indexed caller, bytes32 indexed marker, uint256 value)
func (_ProofDemo *ProofDemoFilterer) WatchValueUpdated(opts *bind.WatchOpts, sink chan<- *ProofDemoValueUpdated, caller []common.Address, marker [][32]byte) (event.Subscription, error) {

	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}
	var markerRule []interface{}
	for _, markerItem := range marker {
		markerRule = append(markerRule, markerItem)
	}

	logs, sub, err := _ProofDemo.contract.WatchLogs(opts, "ValueUpdated", callerRule, markerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProofDemoValueUpdated)
				if err := _ProofDemo.contract.UnpackLog(event, "ValueUpdated", log); err != nil {
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

// ParseValueUpdated is a log parse operation binding the contract event 0x30b70f4f4e3f9a7e61df8ccb304723c5ec51f959a9fab821eff10d6b00eba5b4.
//
// Solidity: event ValueUpdated(address indexed caller, bytes32 indexed marker, uint256 value)
func (_ProofDemo *ProofDemoFilterer) ParseValueUpdated(log types.Log) (*ProofDemoValueUpdated, error) {
	event := new(ProofDemoValueUpdated)
	if err := _ProofDemo.contract.UnpackLog(event, "ValueUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
