package alchemyws

import "encoding/json"

// MinedTxOptions defines the parameters for the subscription request.
type MinedTxOptions struct {
	Addresses      []AddressFilter `json:"addresses,omitempty"`
	IncludeRemoved bool            `json:"includeRemoved,omitempty"`
	HashesOnly     bool            `json:"hashesOnly,omitempty"`
}

// AddressFilter can filter transactions by sender and receiver addresses.
type AddressFilter struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// MinedTxEvent wraps the transaction and removal status.
type MinedTxEvent struct {
	Removed     bool        `json:"removed"`
	Transaction Transaction `json:"transaction,omitempty"`
	Hash        string      `json:"hash,omitempty"`
}

// PendingTxOptions defines parameters for subscribing to pending transactions
type PendingTxOptions struct {
	FromAddress []string `json:"fromAddress,omitempty"`
	ToAddress   []string `json:"toAddress,omitempty"`
	HashesOnly  bool     `json:"hashesOnly,omitempty"`
}

// PendingTxEvent represents either a full pending transaction or a hash
// depending on the HashesOnly flag in subscription.
type PendingTxEvent struct {
	Transaction Transaction `json:"transaction,omitempty"`
	Hash        string      `json:"hash,omitempty"`
}

// Transaction is the generic Ethereum transaction format for mined or pending
// When pending, many fields may be null.
type Transaction struct {
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	From             string `json:"from"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	Hash             string `json:"hash"`
	Input            string `json:"input"`
	Nonce            string `json:"nonce"`
	To               string `json:"to"`
	TransactionIndex string `json:"transactionIndex"`
	Value            string `json:"value"`
	V                string `json:"v"`
	R                string `json:"r"`
	S                string `json:"s"`
	Type             string `json:"type"`
}

// NewHeadEvent represents a new block header received via the newHeads subscription.
type NewHeadEvent struct {
	Number           string `json:"number"`
	ParentHash       string `json:"parentHash"`
	Nonce            string `json:"nonce"`
	Sha3Uncles       string `json:"sha3Uncles"`
	LogsBloom        string `json:"logsBloom"`
	TransactionsRoot string `json:"transactionsRoot"`
	StateRoot        string `json:"stateRoot"`
	ReceiptsRoot     string `json:"receiptsRoot"`
	Miner            string `json:"miner"`
	Difficulty       string `json:"difficulty"`
	ExtraData        string `json:"extraData"`
	GasLimit         string `json:"gasLimit"`
	GasUsed          string `json:"gasUsed"`
	Timestamp        string `json:"timestamp"`
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type subscriptionMessage struct {
	Subscription string          `json:"subscription"`
	Result       json.RawMessage `json:"result"`
}

type rpcEnvelope struct {
	ID     int             `json:"id,omitempty"`
	Result string          `json:"result,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}
