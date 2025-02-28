package wallet

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type savedAddressMeta struct {
	Removed     bool
	UpdateClock uint64 // wall clock used to deconflict concurrent updates
}

type SavedAddress struct {
	Address common.Address `json:"address"`
	// TODO: Add Emoji
	// Emoji	string		   `json:"emoji"`
	Name            string `json:"name"`
	Favourite       bool   `json:"favourite"`
	ChainShortNames string `json:"chainShortNames"` // used with address only, not with ENSName
	ENSName         string `json:"ens"`
	IsTest          bool   `json:"isTest"`
	savedAddressMeta
}

func (s *SavedAddress) ID() string {
	return fmt.Sprintf("%s-%s-%t", s.Address.Hex(), s.ENSName, s.IsTest)
}

type SavedAddressesManager struct {
	db *sql.DB
}

func NewSavedAddressesManager(db *sql.DB) *SavedAddressesManager {
	return &SavedAddressesManager{db: db}
}

const rawQueryColumnsOrder = "address, name, favourite, removed, update_clock, chain_short_names, ens_name, is_test"

// getSavedAddressesFromDBRows retrieves all data based on SELECT Query using rawQueryColumnsOrder
func getSavedAddressesFromDBRows(rows *sql.Rows) ([]SavedAddress, error) {
	var addresses []SavedAddress
	for rows.Next() {
		sa := SavedAddress{}
		// based on rawQueryColumnsOrder
		err := rows.Scan(&sa.Address, &sa.Name, &sa.Favourite, &sa.Removed, &sa.UpdateClock, &sa.ChainShortNames, &sa.ENSName, &sa.IsTest)
		if err != nil {
			return nil, err
		}

		addresses = append(addresses, sa)
	}

	return addresses, nil
}

func (sam *SavedAddressesManager) getSavedAddresses(condition string) ([]SavedAddress, error) {
	var whereCondition string
	if condition != "" {
		whereCondition = fmt.Sprintf("WHERE %s", condition)
	}

	rows, err := sam.db.Query(fmt.Sprintf("SELECT %s FROM saved_addresses %s", rawQueryColumnsOrder, whereCondition))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	addresses, err := getSavedAddressesFromDBRows(rows)
	return addresses, err
}

func (sam *SavedAddressesManager) GetSavedAddresses() ([]SavedAddress, error) {
	return sam.getSavedAddresses("removed != 1")
}

// GetRawSavedAddresses provides access to the soft-delete and sync metadata
func (sam *SavedAddressesManager) GetRawSavedAddresses() ([]SavedAddress, error) {
	return sam.getSavedAddresses("")
}

func (sam *SavedAddressesManager) upsertSavedAddress(sa SavedAddress, tx *sql.Tx) error {
	sqlStatement := "INSERT OR REPLACE INTO saved_addresses (address, name, favourite, removed, update_clock, chain_short_names, ens_name, is_test) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	var err error
	var insert *sql.Stmt
	if tx != nil {
		insert, err = tx.Prepare(sqlStatement)
	} else {
		insert, err = sam.db.Prepare(sqlStatement)
	}
	if err != nil {
		return err
	}
	defer insert.Close()
	_, err = insert.Exec(sa.Address, sa.Name, sa.Favourite, sa.Removed, sa.UpdateClock, sa.ChainShortNames, sa.ENSName, sa.IsTest)
	return err
}

func (sam *SavedAddressesManager) UpdateMetadataAndUpsertSavedAddress(sa SavedAddress) (updatedClock uint64, err error) {
	sa.UpdateClock = uint64(time.Now().Unix())
	err = sam.upsertSavedAddress(sa, nil)
	if err != nil {
		return 0, err
	}
	return sa.UpdateClock, nil
}

func (sam *SavedAddressesManager) startTransactionAndCheckIfNewerChange(address common.Address, ens string, isTest bool, updateClock uint64) (newer bool, tx *sql.Tx, err error) {
	tx, err = sam.db.Begin()
	if err != nil {
		return false, nil, err
	}
	row := tx.QueryRow("SELECT update_clock FROM saved_addresses WHERE address = ? AND is_test = ? AND ens_name = ?", address, isTest, ens)
	if err != nil {
		return false, tx, err
	}

	var dbUpdateClock uint64
	err = row.Scan(&dbUpdateClock)
	if err != nil {
		return err == sql.ErrNoRows, tx, err
	}
	return dbUpdateClock < updateClock, tx, nil
}

func (sam *SavedAddressesManager) AddSavedAddressIfNewerUpdate(sa SavedAddress, updateClock uint64) (insertedOrUpdated bool, err error) {
	newer, tx, err := sam.startTransactionAndCheckIfNewerChange(sa.Address, sa.ENSName, sa.IsTest, updateClock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()
	if !newer {
		return false, err
	}

	sa.UpdateClock = updateClock
	err = sam.upsertSavedAddress(sa, tx)
	if err != nil {
		return false, err
	}

	return true, err
}

func (sam *SavedAddressesManager) DeleteSavedAddress(address common.Address, ens string, isTest bool, updateClock uint64) (deleted bool, err error) {
	if err != nil {
		return false, err
	}
	newer, tx, err := sam.startTransactionAndCheckIfNewerChange(address, ens, isTest, updateClock)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()
	if !newer {
		return false, err
	}

	update, err := tx.Prepare(`UPDATE saved_addresses SET removed = 1, update_clock = ? WHERE address = ? AND is_test = ? AND ens_name = ?`)
	if err != nil {
		return false, err
	}
	defer update.Close()
	res, err := update.Exec(updateClock, address, isTest, ens)
	if err != nil {
		return false, err
	}

	nRows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return nRows > 0, nil
}

func (sam *SavedAddressesManager) DeleteSoftRemovedSavedAddresses(threshold uint64) error {
	_, err := sam.db.Exec(`DELETE FROM saved_addresses WHERE removed = 1 AND update_clock < ?`, threshold)
	return err
}
