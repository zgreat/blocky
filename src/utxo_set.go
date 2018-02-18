package main

import (
	"encoding/hex"
	"log"

	"github.com/boltdb/bolt"
)

const utxoBucket = "chainstate"

type UTXOSet struct {
	bc *Blockchain
}

func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	var unspentUTXO = make(map[string][]int)
	db := u.bc.db
	acc := 0

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && acc < amount {
					acc += out.Value
					unspentUTXO[txID] = append(unspentUTXO[txID], outIdx)
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return acc, unspentUTXO
}

func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TXOutput {
	var utxos []TXOutput
	db := u.bc.db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					utxos = append(utxos, out)
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return utxos
}

func (u UTXOSet) Reindex() {
	db := u.bc.db
	bucketName := []byte(utxoBucket)

	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}

		_, err = tx.CreateBucket(bucketName)
		if err != nil {
			log.Panic(err)
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
	utxos := u.bc.FindUTXO()
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		for txID, outs := range utxos {
			txid, err := hex.DecodeString(txID)
			err = b.Put(txid, SerializeOutputs(outs))
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

/*
func (u UTXOSet) Update(block *Block) {
	db := u.bc.db
	bucketName := []byte(utxoBucket)

	err := db.Update(func(txBolt *bolt.Tx) error {
		bucket := txBolt.Bucket(bucketName)

		for _, tx := range block.Transactions {
			if !tx.IsCoinbase() {
				for _, inTx := range tx.Vin {
					updatedOutputs := TXOutputs{}
					byteOutputs := bucket.Get(inTx.Txid)
					oldOutputs := DeserializeOutputs(byteOutputs)

					for outIdx, outTx := range oldOutputs.Outputs { // pretty weired
						if outIdx != inTx.Vout {
							updatedOutputs.Outputs = append(updatedOutputs.Outputs, outTx)
						}
					}

					if len(updatedOutputs.Outputs) == 0 {
						err := bucket.Delete(inTx.Txid)
						if err != nil {
							log.Panic(err)
						}
					} else {
						err := bucket.Put(inTx.Txid, SerializeOutputs(updatedOutputs))
						if err != nil {
							log.Panic(err)
						}
					}

				}
			}
			newOutputs := TXOutputs{}
			for _, outTx := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, outTx)
			}
			err := bucket.Put(tx.ID, SerializeOutputs(newOutputs))
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}
*/

func (u UTXOSet) Update(block *Block) {
	db := u.bc.db

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false {
				for _, vin := range tx.Vin {
					updatedOuts := TXOutputs{}
					outsBytes := b.Get(vin.Txid)
					outs := DeserializeOutputs(outsBytes)

					for outIdx, out := range outs.Outputs {
						if outIdx != vin.Vout {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						err := b.Delete(vin.Txid)
						if err != nil {
							log.Panic(err)
						}
					} else {
						err := b.Put(vin.Txid, SerializeOutputs(updatedOuts))
						if err != nil {
							log.Panic(err)
						}
					}

				}
			}

			newOutputs := TXOutputs{}
			for _, out := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			err := b.Put(tx.ID, SerializeOutputs(newOutputs))
			if err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}