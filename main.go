package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Account rappresenta un account Ethereum
type Account struct {
	gorm.Model
	PrivateKey string `gorm:"unique"`
	PublicKey  string
	Address    string
	Balance    float64
}

func main() {
	db := initDatabase()

	for {
		privateKey, err := generatePrivateKey()
		if err != nil {
			fmt.Println("error generating private key:", err)
			continue
		}

		address, publicKeyBytes, err := getAddressAndPublicKey(privateKey)
		if err != nil {
			fmt.Println("error generating address and public key:", err)
			continue
		}

		balanceInEther, err := getAccountBalance(address)
		if err != nil {
			fmt.Println("error getting account balance:", err)
			continue
		}

		if balanceInEther > 0 {
			account := Account{
				PrivateKey: hexutil.Encode(crypto.FromECDSA(privateKey)),
				PublicKey:  hexutil.Encode(publicKeyBytes),
				Address:    address,
				Balance:    balanceInEther,
			}
			if err := db.Create(&account).Error; err != nil {
				fmt.Println("error during create account:", err)
			}
		}
	}
}

func initDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("accounts.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Esegui migrazioni per creare la tabella degli account se non esiste già
	if err := db.AutoMigrate(&Account{}); err != nil {
		panic(err)
	}

	return db
}

func generatePrivateKey() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

func getAddressAndPublicKey(privateKey *ecdsa.PrivateKey) (string, []byte, error) {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", nil, fmt.Errorf("error generating public key")
	}
	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	return address, publicKeyBytes, nil
}

func getAccountBalance(address string) (float64, error) {
	url := "https://weathered-restless-spree.quiknode.pro/67256ba45eaf985ad6528c8145071d80203bd9b0/"
	payload := map[string]interface{}{
		"method":  "eth_getBalance",
		"params":  []interface{}{address, "latest"},
		"id":      1,
		"jsonrpc": "2.0",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	weiBalance := result["result"].(string)
	balanceInWei, err := strconv.ParseInt(weiBalance, 0, 64)
	if err != nil {
		return 0, err
	}
	balanceInEther := float64(balanceInWei) / math.Pow10(18)
	return balanceInEther, nil
}
