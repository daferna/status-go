package mailservers

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/protocol/transport"
)

type Mailserver struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Custom         bool   `json:"custom"`
	Address        string `json:"address"`
	Password       string `json:"password,omitempty"`
	Fleet          string `json:"fleet"`
	Version        uint   `json:"version"`
	FailedRequests uint   `json:"-"`
}

func (m Mailserver) Enode() (*enode.Node, error) {
	return enode.ParseV4(m.Address)
}

func (m Mailserver) IDBytes() ([]byte, error) {
	if m.Version == 2 {
		id, err := peer.Decode(m.UniqueID())
		if err != nil {
			return nil, err
		}
		return []byte(id.Pretty()), err
	}

	node, err := enode.ParseV4(m.Address)
	if err != nil {
		return nil, err
	}
	return node.ID().Bytes(), nil
}

func (m Mailserver) PeerID() (peer.ID, error) {
	if m.Version != 2 {
		return "", errors.New("not available")
	}

	pID, err := peer.Decode(m.UniqueID())
	if err != nil {
		return "", err
	}

	return pID, nil
}

func (m Mailserver) UniqueID() string {
	if m.Version == 2 {
		s := strings.Split(m.Address, "/")
		return s[len(s)-1]
	}
	return m.Address
}

func (m Mailserver) nullablePassword() (val sql.NullString) {
	if m.Password != "" {
		val.String = m.Password
		val.Valid = true
	}
	return
}

type MailserverRequestGap struct {
	ID     string `json:"id"`
	ChatID string `json:"chatId"`
	From   uint64 `json:"from"`
	To     uint64 `json:"to"`
}

type MailserverTopic struct {
	Topic       string   `json:"topic"`
	Discovery   bool     `json:"discovery?"`
	Negotiated  bool     `json:"negotiated?"`
	ChatIDs     []string `json:"chat-ids"`
	LastRequest int      `json:"last-request"` // default is 1
}

type ChatRequestRange struct {
	ChatID            string `json:"chat-id"`
	LowestRequestFrom int    `json:"lowest-request-from"`
	HighestRequestTo  int    `json:"highest-request-to"`
}

// sqlStringSlice helps to serialize a slice of strings into a single column using JSON serialization.
type sqlStringSlice []string

// Scan implements the Scanner interface.
func (ss *sqlStringSlice) Scan(value interface{}) error {
	if value == nil {
		*ss = nil
		return nil
	}
	src, ok := value.([]byte)
	if !ok {
		return errors.New("invalid value type, expected byte slice")
	}
	return json.Unmarshal(src, ss)
}

// Value implements the driver Valuer interface.
func (ss sqlStringSlice) Value() (driver.Value, error) {
	return json.Marshal(ss)
}

// Database sql wrapper for operations with mailserver objects.
type Database struct {
	db *sql.DB
}

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

func (d *Database) Add(mailserver Mailserver) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO mailservers(
			id,
			name,
			address,
			password,
			fleet
		) VALUES (?, ?, ?, ?, ?)`,
		mailserver.ID,
		mailserver.Name,
		mailserver.Address,
		mailserver.nullablePassword(),
		mailserver.Fleet,
	)
	return err
}

func (d *Database) Mailservers() ([]Mailserver, error) {
	var result []Mailserver

	rows, err := d.db.Query(`SELECT id, name, address, password, fleet FROM mailservers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			m        Mailserver
			password sql.NullString
		)
		if err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.Address,
			&password,
			&m.Fleet,
		); err != nil {
			return nil, err
		}
		m.Custom = true
		if password.Valid {
			m.Password = password.String
		}
		result = append(result, m)
	}

	return result, nil
}

func (d *Database) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM mailservers WHERE id = ?`, id)
	return err
}

func (d *Database) AddGaps(gaps []MailserverRequestGap) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	for _, gap := range gaps {

		_, err := tx.Exec(`INSERT OR REPLACE INTO mailserver_request_gaps(
				id,
				chat_id,
				gap_from,
				gap_to
			) VALUES (?, ?, ?, ?)`,
			gap.ID,
			gap.ChatID,
			gap.From,
			gap.To,
		)
		if err != nil {
			return err
		}

	}
	return nil
}

func (d *Database) RequestGaps(chatID string) ([]MailserverRequestGap, error) {
	var result []MailserverRequestGap

	rows, err := d.db.Query(`SELECT id, chat_id, gap_from, gap_to FROM mailserver_request_gaps WHERE chat_id = ?`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m MailserverRequestGap
		if err := rows.Scan(
			&m.ID,
			&m.ChatID,
			&m.From,
			&m.To,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	return result, nil
}

func (d *Database) DeleteGaps(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := fmt.Sprintf(`DELETE FROM mailserver_request_gaps WHERE id IN (%s)`, inVector) // nolint: gosec
	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	_, err := d.db.Exec(query, idsArgs...)
	return err
}

func (d *Database) DeleteGapsByChatID(chatID string) error {
	_, err := d.db.Exec(`DELETE FROM mailserver_request_gaps WHERE chat_id = ?`, chatID)
	return err
}

func (d *Database) AddTopic(topic MailserverTopic) error {

	chatIDs := sqlStringSlice(topic.ChatIDs)
	_, err := d.db.Exec(`INSERT OR REPLACE INTO mailserver_topics(
			topic,
			chat_ids,
			last_request,
			discovery,
			negotiated
		) VALUES (?, ?, ?,?,?)`,
		topic.Topic,
		chatIDs,
		topic.LastRequest,
		topic.Discovery,
		topic.Negotiated,
	)
	return err
}

func (d *Database) AddTopics(topics []MailserverTopic) (err error) {
	var tx *sql.Tx
	tx, err = d.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	for _, topic := range topics {
		chatIDs := sqlStringSlice(topic.ChatIDs)
		_, err = tx.Exec(`INSERT OR REPLACE INTO mailserver_topics(
			  topic,
			  chat_ids,
			  last_request,
			  discovery,
			  negotiated
		  ) VALUES (?, ?, ?,?,?)`,
			topic.Topic,
			chatIDs,
			topic.LastRequest,
			topic.Discovery,
			topic.Negotiated,
		)
		if err != nil {
			return
		}
	}
	return
}

func (d *Database) Topics() ([]MailserverTopic, error) {
	var result []MailserverTopic

	rows, err := d.db.Query(`SELECT topic, chat_ids, last_request,discovery,negotiated FROM mailserver_topics`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			t       MailserverTopic
			chatIDs sqlStringSlice
		)
		if err := rows.Scan(
			&t.Topic,
			&chatIDs,
			&t.LastRequest,
			&t.Discovery,
			&t.Negotiated,
		); err != nil {
			return nil, err
		}
		t.ChatIDs = chatIDs
		result = append(result, t)
	}

	return result, nil
}

func (d *Database) ResetLastRequest(topic string) error {
	_, err := d.db.Exec("UPDATE mailserver_topics SET last_request = 0 WHERE topic = ?", topic)
	return err
}

func (d *Database) DeleteTopic(topic string) error {
	_, err := d.db.Exec(`DELETE FROM mailserver_topics WHERE topic = ?`, topic)
	return err
}

// SetTopics deletes all topics excepts the one set, or upsert those if
// missing
func (d *Database) SetTopics(filters []*transport.Filter) (err error) {
	var tx *sql.Tx
	tx, err = d.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if len(filters) == 0 {
		return nil
	}

	topicsArgs := make([]interface{}, 0, len(filters))
	for _, filter := range filters {
		topicsArgs = append(topicsArgs, filter.Topic.String())
	}

	inVector := strings.Repeat("?, ", len(filters)-1) + "?"

	// Delete topics
	query := "DELETE FROM mailserver_topics WHERE topic NOT IN (" + inVector + ")" // nolint: gosec
	_, err = tx.Exec(query, topicsArgs...)

	// Default to now - 1.day
	lastRequest := (time.Now().Add(-24 * time.Hour)).Unix()
	// Insert if not existing
	for _, filter := range filters {
		// fetch
		var topic string
		err = tx.QueryRow(`SELECT topic FROM mailserver_topics WHERE topic = ?`, filter.Topic.String()).Scan(&topic)
		if err != nil && err != sql.ErrNoRows {
			return
		} else if err == sql.ErrNoRows {
			// we insert the topic
			_, err = tx.Exec(`INSERT INTO mailserver_topics(topic,last_request,discovery,negotiated) VALUES (?,?,?,?)`, filter.Topic.String(), lastRequest, filter.Discovery, filter.Negotiated)
		}
		if err != nil {
			return
		}
	}

	return
}

func (d *Database) AddChatRequestRange(req ChatRequestRange) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO mailserver_chat_request_ranges(
			chat_id,
			lowest_request_from,
			highest_request_to
		) VALUES (?, ?, ?)`,
		req.ChatID,
		req.LowestRequestFrom,
		req.HighestRequestTo,
	)
	return err
}

func (d *Database) AddChatRequestRanges(reqs []ChatRequestRange) (err error) {
	var tx *sql.Tx
	tx, err = d.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()
	for _, req := range reqs {

		_, err = tx.Exec(`INSERT OR REPLACE INTO mailserver_chat_request_ranges(
			chat_id,
			lowest_request_from,
			highest_request_to
		) VALUES (?, ?, ?)`,
			req.ChatID,
			req.LowestRequestFrom,
			req.HighestRequestTo,
		)
		if err != nil {
			return
		}
	}
	return
}

func (d *Database) ChatRequestRanges() ([]ChatRequestRange, error) {
	var result []ChatRequestRange

	rows, err := d.db.Query(`SELECT chat_id, lowest_request_from, highest_request_to FROM mailserver_chat_request_ranges`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var req ChatRequestRange
		if err := rows.Scan(
			&req.ChatID,
			&req.LowestRequestFrom,
			&req.HighestRequestTo,
		); err != nil {
			return nil, err
		}
		result = append(result, req)
	}

	return result, nil
}

func (d *Database) DeleteChatRequestRange(chatID string) error {
	_, err := d.db.Exec(`DELETE FROM mailserver_chat_request_ranges WHERE chat_id = ?`, chatID)
	return err
}
