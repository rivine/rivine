package transactionpool

import (
	"github.com/rivine/rivine/crypto"
	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/types"

	"github.com/rivine/bbolt"
)

// purge removes all transactions from the transaction pool.
func (tp *TransactionPool) purge() {
	tp.knownObjects = make(map[ObjectID]TransactionSetID)
	tp.transactionSets = make(map[TransactionSetID][]types.Transaction)
	tp.transactionSetDiffs = make(map[TransactionSetID]modules.ConsensusChange)
	tp.transactionListSize = 0
}

// ProcessConsensusChange gets called to inform the transaction pool of changes
// to the consensus set.
func (tp *TransactionPool) ProcessConsensusChange(cc modules.ConsensusChange) {
	tp.mu.Lock()

	// Update the database of confirmed transactions.
	err := tp.db.Update(func(tx *bolt.Tx) error {
		for _, block := range cc.RevertedBlocks {
			for _, txn := range block.Transactions {
				err := tp.deleteTransaction(tx, txn.ID())
				if err != nil {
					return err
				}
			}
		}
		for _, block := range cc.AppliedBlocks {
			for _, txn := range block.Transactions {
				err := tp.addTransaction(tx, txn.ID())
				if err != nil {
					return err
				}
			}
		}
		return tp.putRecentConsensusChange(tx, cc.ID)
	})
	if err != nil {
		// TODO: Handle error
	}

	// Remove all transactions confirmed in the block from the cache
	for _, block := range cc.AppliedBlocks {
		for _, txn := range block.Transactions {
			setID := TransactionSetID(crypto.HashObject([]types.Transaction{txn}))
			tp.broadcastCache.delete(setID)
		}
	}

	// Scan the applied blocks for transactions that got accepted. This will
	// help to determine which transactions to remove from the transaction
	// pool. Having this list enables both efficiency improvements and helps to
	// clean out transactions with no dependencies, such as arbitrary data
	// transactions from the host.
	txids := make(map[types.TransactionID]struct{})
	for _, block := range cc.AppliedBlocks {
		for _, txn := range block.Transactions {
			txids[txn.ID()] = struct{}{}
		}
	}

	// TODO: Right now, transactions that were reverted to not get saved and
	// retried, because some transactions such as storage proofs might be
	// illegal, and there's no good way to preserve dependencies when illegal
	// transactions are suddenly involved.
	//
	// One potential solution is to have modules manually do resubmission if
	// something goes wrong. Another is to have the transaction pool remember
	// recent transaction sets on the off chance that they become valid again
	// due to a reorg.
	//
	// Another option is to scan through the blocks transactions one at a time
	// check if they are valid. If so, lump them in a set with the next guy.
	// When they stop being valid, you've found a guy to throw away. It's n^2
	// in the number of transactions in the block.

	// Save all of the current unconfirmed transaction sets into a list.
	var unconfirmedSets [][]types.Transaction
	for _, tSet := range tp.transactionSets {
		// Compile a new transaction set the removes all transactions duplicated
		// in the block. Though mostly handled by the dependency manager in the
		// transaction pool, this should both improve efficiency and will strip
		// out duplicate transactions with no dependencies (arbitrary data only
		// transactions)
		var newTSet []types.Transaction
		for _, txn := range tSet {
			_, exists := txids[txn.ID()]
			if !exists {
				newTSet = append(newTSet, txn)
			}
		}
		unconfirmedSets = append(unconfirmedSets, newTSet)
	}

	// Purge the transaction pool. Some of the transactions sets may be invalid
	// after the consensus change.
	tp.purge()

	// Add all of the unconfirmed transaction sets back to the transaction
	// pool. The ones that are invalid will throw an error and will not be
	// re-added.
	//
	// Accepting a transaction set requires locking the consensus set (to check
	// validity). But, ProcessConsensusChange is only called when the consensus
	// set is already locked, causing a deadlock problem. Therefore,
	// transactions are readded to the pool in a goroutine, so that this
	// function can finish and consensus can unlock. The tpool lock is held
	// however until the goroutine completes.
	//
	// Which means that no other modules can require a tpool lock when
	// processing consensus changes. Overall, the locking is pretty fragile and
	// more rules need to be put in place.
	// Accepting the set again will write the current block height in the
	// broadcast cache. So we copy the cache, clear it, and override it later
	for _, set := range unconfirmedSets {
		if err := tp.acceptTransactionSet(set); err != nil {
			// the transaction is now invalid and no longer in the pool,
			// so remove it from the cache as well
			setID := TransactionSetID(crypto.HashObject(set))
			tp.broadcastCache.delete(setID)
			continue
		}
	}

	// If we are synced, try to broadcast again
	if cc.Synced {
		currentheight := tp.consensusSet.Height()
		for _, id := range tp.broadcastCache.getTransactionsToBroadcast(currentheight) {
			go tp.gateway.Broadcast("RelayTransactionSet", tp.transactionSets[id], tp.gateway.Peers())
		}
	}

	// Inform subscribers that an update has executed.
	tp.mu.Demote()
	tp.updateSubscribersTransactions()
	tp.mu.DemotedUnlock()
}

// PurgeTransactionPool deletes all transactions from the transaction pool.
func (tp *TransactionPool) PurgeTransactionPool() {
	tp.mu.Lock()
	tp.purge()
	tp.mu.Unlock()
}
