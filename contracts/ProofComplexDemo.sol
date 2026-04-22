// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract ProofComplexDemo {
    struct Position {
        uint256 quantity;
        uint256 lastPrice;
    }

    mapping(address => uint256) public balances;
    mapping(address => uint256[]) private history;
    mapping(address => mapping(uint256 => Position)) private positions;
    string private note;
    bytes private payload;

    event ComplexStateUpdated(
        address indexed caller,
        uint256 indexed positionId,
        bytes32 indexed marker,
        uint256 balance,
        uint256 historyValue,
        uint256 quantity,
        uint256 lastPrice
    );

    function seedHistory(address user, uint256[] calldata values) external {
        delete history[user];
        for (uint256 i = 0; i < values.length; ++i) {
            history[user].push(values[i]);
        }
    }

    function applyUpdate(
        uint256 balanceValue,
        uint256 positionId,
        uint256 historyValue,
        uint256 quantity,
        uint256 lastPrice,
        string calldata nextNote,
        bytes calldata nextPayload,
        bytes32 marker
    ) external {
        balances[msg.sender] = balanceValue;
        history[msg.sender].push(historyValue);
        positions[msg.sender][positionId] = Position({quantity: quantity, lastPrice: lastPrice});
        note = nextNote;
        payload = nextPayload;

        emit ComplexStateUpdated(msg.sender, positionId, marker, balanceValue, historyValue, quantity, lastPrice);
    }

    function historyLength(address user) external view returns (uint256) {
        return history[user].length;
    }

    function historyAt(address user, uint256 index) external view returns (uint256) {
        return history[user][index];
    }

    function positionOf(address user, uint256 positionId) external view returns (uint256 quantity, uint256 lastPrice) {
        Position storage position = positions[user][positionId];
        return (position.quantity, position.lastPrice);
    }

    function noteText() external view returns (string memory) {
        return note;
    }

    function payloadData() external view returns (bytes memory) {
        return payload;
    }
}
