Proof Of BlockStake
===================

This document is meant to provide a good high level overview of the BlockStake
algorithm.

General
-------


protocol
--------

![POBSprotocoloverview](https://rawgit.com/rivine/rivine/master/doc/POBSoverview.svg)

The hash function used is a 32-byte BLAKE2b hash. To compare the hash with the difficulty it is interpreted as a big-endian unsigned integer.

Stakemodifier
-------------


Transaction fees
----------------

For every transaction, a fee (exact amount to be determined) is charged.

Every Block starts with 3 fixed transactions:

All Fees from all transactions in the created block goes to:
25% to the BCN who generate the block
75% to address 0 (zero)
From address 0 , a certain percentage goes to BCN who generate the block
From address 1 , a fixed amount (ex 1BDG) goes to BCN who generate the block

Address 0 is filled by each transaction fee and distributed over the next few
block creators. Address 1 can be artificially filled by the foundation to start
up the system.

Since everything needs to be Backed, no new Digital Value can be created.
But there is still an incentive to mine because:

You get a certain amount of the fee of the transactions in the block.

If there are no transactions, you still get value from the fixed 2e transaction
which are fees from transactions in the previous blocks. And the BCN get also
value from the 3 fixed transaction from address 1.

If somebody makes a transaction error or does a test and send an amount to
address 0, that amount will not get lost but will be distributed over the BCNs.

### Maturity of collected transaction fees

Coins received through the collection of transaction fees during block creation can't be spent until the block has 144 confirmations. Transactions that try to spend a block creation fee output before this will be rejected.

The reason for this is that sometimes the block chain forks, blocks that were valid become invalid, and the block creation reward in those blocks is lost. That's just an unavoidable part of how blockchains works, and it can sometimes happen even when there is no one attacking the network. If there was no maturation time, then whenever a fork happened, everyone who received coins that were collected on an unlucky fork (possibly through many intermediaries) would have their coins disappear, even without any sort of double-spend or other attack. On long forks, thousands of people could find coins disappearing from their wallets, even though there is no one actually attacking them and they had no reason to be suspicious of the money they were receiving. For example, without a maturation time, a block creator might deposit 25 coins into an EWallet, and if I withdraw money from a completely unrelated account on the same EWallet, my withdrawn money might just disappear if there is a fork and I'm unlucky enough to withdraw coins that have been "tainted" by the block creator's now-invalid coins. Due to the way this sort of taint tends to "infect" transactions, far more than 25 coins per block would be affected. Each invalidated block could cause transactions collectively worth hundreds of coins to be reversed. The maturation time makes it impossible for anyone to lose coins by accident like this as long as a fork doesn't last longer than 144 blocks.

Difficulty
----------

The hash in the POBS protocol results in a 256 bit integer so there are 2^256 possible combinations. If we want 1 block to be created every 10 minutes on average, this means that 1 hash should match every 10\*60 seconds.
The chance of having a match is also multiplied by the number of blockstakes you have available so for the starting difficulty this means it should be divided by the total number of blockstakes in the system.

Difficulty is adjusted every 50 blocks to compensate for the fact that not every blockstake available is always participating in the POBS protocol.

![POBSprotocoldifficulty](https://rawgit.com/rivine/rivine/master/doc/POBSdifficulty.svg)