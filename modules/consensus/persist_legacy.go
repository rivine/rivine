package consensus

import (
	bolt "github.com/rivine/bbolt"
	"github.com/rivine/rivine/build"
	"github.com/rivine/rivine/crypto"
	"github.com/rivine/rivine/encoding"
	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/persist"
	"github.com/rivine/rivine/types"
)

// convertLegacyDatabase converts a 0.5.0 consensus database,
// to a database of the current version as defined by dbMetadata.
// It keeps the database open and returns it for further usage.
func convertLegacyDatabase(filePath string) (db *persist.BoltDatabase, err error) {
	var legacyDBMetadata = persist.Metadata{
		Header:  "Consensus Set Database",
		Version: "0.5.0",
	}
	db, err = persist.OpenDatabase(legacyDBMetadata, filePath)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if bucket := tx.Bucket(BlockMap); bucket != nil {
			if err := updateLegacyBlockMapBucket(bucket); err != nil {
				return err
			}
		}
		if bucket := tx.Bucket(CoinOutputs); bucket != nil {
			if err := updateLegacyCoinOutputBucket(bucket); err != nil {
				return err
			}
		}
		if bucket := tx.Bucket(BlockStakeOutputs); bucket != nil {
			return updateLegacyBlockstakeOutputBucket(bucket)
		}
		return nil
	})
	if err == nil {
		// set the new metadata, and save it,
		// such that next time we have the new version stored
		db.Header, db.Version = dbMetadata.Header, dbMetadata.Version
		err = db.SaveMetadata()
	}
	if err != nil {
		err := db.Close()
		if build.DEBUG && err != nil {
			panic(err)
		}
	}
	return
}

func updateLegacyBlockMapBucket(bucket *bolt.Bucket) error {
	var (
		err    error
		cursor = bucket.Cursor()
	)
	for k, v := cursor.First(); len(k) != 0; k, v = cursor.Next() {
		// try to decode the legacy format
		var legacyBlock legacyProcessedBlock
		err = encoding.Unmarshal(v, &legacyBlock)
		if err != nil {
			// ensure it is in the new format already
			var block processedBlock
			err = encoding.Unmarshal(v, &block)
			if err != nil {
				return err
			}
		}
		// it's in the legacy format, as expected, we overwrite it using the new format
		err = legacyBlock.storeAsNewFormat(bucket, k)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateLegacyCoinOutputBucket(bucket *bolt.Bucket) error {
	var (
		err    error
		cursor = bucket.Cursor()
	)
	for k, v := cursor.First(); len(k) != 0; k, v = cursor.Next() {
		// try to decode the legacy format
		var out legacyOutput
		err = encoding.Unmarshal(v, &out)
		if err != nil {
			// ensure it is in the new format already
			var co types.CoinOutput
			err = encoding.Unmarshal(v, &co)
			if err != nil {
				return err
			}
		}
		// it's in the legacy format, as expected, we overwrite it using the new format
		err = bucket.Put(k, encoding.Marshal(types.CoinOutput{
			Value: out.Value,
			Condition: types.UnlockConditionProxy{
				Condition: types.NewUnlockHashCondition(out.UnlockHash),
			},
		}))
		if err != nil {
			return err
		}
	}
	return nil
}

func updateLegacyBlockstakeOutputBucket(bucket *bolt.Bucket) error {
	var (
		err    error
		cursor = bucket.Cursor()
	)
	for k, v := cursor.First(); len(k) != 0; k, v = cursor.Next() {
		// try to decode the legacy format
		var out legacyOutput
		err = encoding.Unmarshal(v, &out)
		if err != nil {
			// ensure it is in the new format already
			var bso types.BlockStakeOutput
			err = encoding.Unmarshal(v, &bso)
			if err != nil {
				return err
			}
		}
		// it's in the legacy format, as expected, we overwrite it using the new format
		err = bucket.Put(k, encoding.Marshal(types.BlockStakeOutput{
			Value: out.Value,
			Condition: types.UnlockConditionProxy{
				Condition: types.NewUnlockHashCondition(out.UnlockHash),
			},
		}))
		if err != nil {
			return err
		}
	}
	return nil
}

// legacyProcessedBlock defines the legacy version of what used to be the processedBlock,
// as serialized in the consensus database
type (
	legacyProcessedBlock struct {
		Block       legacyBlock
		Height      types.BlockHeight
		Depth       types.Target
		ChildTarget types.Target

		DiffsGenerated         bool
		CoinOutputDiffs        []legacyCoinOutputDiff
		BlockStakeOutputDiffs  []legacyBlockStakeOutputDiff
		DelayedCoinOutputDiffs []legacyDelayedCoinOutputDiff
		TxIDDiffs              []modules.TransactionIDDiff

		ConsensusChecksum crypto.Hash
	}
	legacyBlock struct {
		ParentID     types.BlockID
		Timestamp    types.Timestamp
		POBSOutput   types.BlockStakeOutputIndexes
		MinerPayouts []types.MinerPayout
		Transactions []types.Transaction
	}
	legacyCoinOutputDiff struct {
		Direction  modules.DiffDirection
		ID         types.CoinOutputID
		CoinOutput legacyOutput
	}
	legacyBlockStakeOutputDiff struct {
		Direction        modules.DiffDirection
		ID               types.BlockStakeOutputID
		BlockStakeOutput legacyOutput
	}
	legacyDelayedCoinOutputDiff struct {
		Direction      modules.DiffDirection
		ID             types.CoinOutputID
		CoinOutput     legacyOutput
		MaturityHeight types.BlockHeight
	}
	legacyOutput struct {
		Value      types.Currency
		UnlockHash types.UnlockHash
	}
)

func (lpb *legacyProcessedBlock) storeAsNewFormat(bucket *bolt.Bucket, key []byte) error {
	block := processedBlock{
		Block: types.Block{
			ParentID:     lpb.Block.ParentID,
			Timestamp:    lpb.Block.Timestamp,
			POBSOutput:   lpb.Block.POBSOutput,
			MinerPayouts: lpb.Block.MinerPayouts,
			Transactions: lpb.Block.Transactions,
		},
		Height:      lpb.Height,
		Depth:       lpb.Depth,
		ChildTarget: lpb.ChildTarget,

		DiffsGenerated: lpb.DiffsGenerated,
		TxIDDiffs:      lpb.TxIDDiffs,

		ConsensusChecksum: lpb.ConsensusChecksum,
	}

	block.CoinOutputDiffs = make([]modules.CoinOutputDiff, len(lpb.CoinOutputDiffs))
	for i, od := range lpb.CoinOutputDiffs {
		block.CoinOutputDiffs[i] = modules.CoinOutputDiff{
			Direction: od.Direction,
			ID:        od.ID,
			CoinOutput: types.CoinOutput{
				Value: od.CoinOutput.Value,
				Condition: types.UnlockConditionProxy{
					Condition: types.NewUnlockHashCondition(od.CoinOutput.UnlockHash),
				},
			},
		}
	}

	block.BlockStakeOutputDiffs = make([]modules.BlockStakeOutputDiff, len(lpb.BlockStakeOutputDiffs))
	for i, od := range lpb.BlockStakeOutputDiffs {
		block.BlockStakeOutputDiffs[i] = modules.BlockStakeOutputDiff{
			Direction: od.Direction,
			ID:        od.ID,
			BlockStakeOutput: types.BlockStakeOutput{
				Value: od.BlockStakeOutput.Value,
				Condition: types.UnlockConditionProxy{
					Condition: types.NewUnlockHashCondition(od.BlockStakeOutput.UnlockHash),
				},
			},
		}
	}

	block.DelayedCoinOutputDiffs = make([]modules.DelayedCoinOutputDiff, len(lpb.DelayedCoinOutputDiffs))
	for i, od := range lpb.DelayedCoinOutputDiffs {
		block.DelayedCoinOutputDiffs[i] = modules.DelayedCoinOutputDiff{
			Direction: od.Direction,
			ID:        od.ID,
			CoinOutput: types.CoinOutput{
				Value: od.CoinOutput.Value,
				Condition: types.UnlockConditionProxy{
					Condition: types.NewUnlockHashCondition(od.CoinOutput.UnlockHash),
				},
			},
			MaturityHeight: od.MaturityHeight,
		}
	}

	return bucket.Put(key, encoding.Marshal(block))
}