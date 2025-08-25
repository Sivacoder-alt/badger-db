package models

type Transaction struct {
	ID        string   `json:"id"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Amount    uint64   `json:"amount"`
	Parents   []string `json:"parents"` 
	Timestamp int64    `json:"timestamp"`
}

type AccountState struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}
