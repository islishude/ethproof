// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract ProofDemo {
    uint256 public value;

    event ValueUpdated(address indexed caller, bytes32 indexed marker, uint256 value);

    function setValue(uint256 newValue, bytes32 marker) external {
        value = newValue;
        emit ValueUpdated(msg.sender, marker, newValue);
    }
}
