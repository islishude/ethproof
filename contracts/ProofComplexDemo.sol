// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract ProofComplexDemo {
    struct Position {
        uint256 quantity;
        uint256 lastPrice;
    }

    struct PackedTriplet {
        uint128 a;
        uint64 b;
        uint64 c;
    }

    struct MixedTriplet {
        uint128 a;
        uint64 b;
        bytes32 c;
    }

    mapping(address => uint256) public balances;
    mapping(address => uint256[]) private history;
    mapping(address => mapping(uint256 => Position)) private positions;
    string private note;
    bytes private payload;
    uint256 public basicUint256;
    address public basicAddress;
    bytes32 public basicBytes32;
    bool public basicBool;
    PackedTriplet private packedTriplet;
    MixedTriplet private mixedTriplet;
    uint128[3] private fixedSmallArray;

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

    function setBasicSamples(
        uint256 nextBasicUint256,
        address nextBasicAddress,
        bytes32 nextBasicBytes32,
        bool nextBasicBool
    ) external {
        basicUint256 = nextBasicUint256;
        basicAddress = nextBasicAddress;
        basicBytes32 = nextBasicBytes32;
        basicBool = nextBasicBool;
    }

    function setPackedTriplet(uint128 nextPackedA, uint64 nextPackedB, uint64 nextPackedC) external {
        packedTriplet = PackedTriplet({a: nextPackedA, b: nextPackedB, c: nextPackedC});
    }

    function setMixedTriplet(uint128 nextMixedA, uint64 nextMixedB, bytes32 nextMixedC) external {
        mixedTriplet = MixedTriplet({a: nextMixedA, b: nextMixedB, c: nextMixedC});
    }

    function setFixedSmallArray(uint128 nextFixed0, uint128 nextFixed1, uint128 nextFixed2) external {
        fixedSmallArray[0] = nextFixed0;
        fixedSmallArray[1] = nextFixed1;
        fixedSmallArray[2] = nextFixed2;
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
