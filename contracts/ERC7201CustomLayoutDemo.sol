// SPDX-License-Identifier: MIT
pragma solidity ^0.8.35;

contract ERC7201CustomLayoutDemo layout at erc7201("openzeppelin.storage.ERC7201CustomLayoutDemo") {
    uint256 x;
    uint256 y;

    constructor(uint256 initialX, uint256 initialY) {
        x = initialX;
        y = initialY;
    }
}
