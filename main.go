package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
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

	urls := []string{
		"https://weathered-restless-spree.quiknode.pro/67256ba45eaf985ad6528c8145071d80203bd9b0/",
		"https://intensive-aged-theorem.quiknode.pro/8f60643fdd3a671086701484c224d97953d429d4/",
		"https://eth-mainnet.g.alchemy.com/v2/owUCVigVvnHA63o0C6mh3yrf3jxMkV7b",
		"https://cloudflare-eth.com",
		"https://rpc.flashbots.net/",
		"https://rpc.ankr.com/eth",
		"https://eth-mainnet.public.blastapi.io",
		"https://api.securerpc.com/v1",
		"https://1rpc.io/eth",
		"https://ethereum.publicnode.com",
		"https://rpc.payload.de",
		"https://llamanodes.com/",
		"https://eth.api.onfinality.io/public",
		"https://eth.merkle.io",
		"https://eth.drpc.org",
		"https://public.stackup.sh/api/v1/node/ethereum-mainnet",
		"https://eth.llamarpc.com",
		"https://ethereum.blockpi.network/v1/rpc/public",
		"https://ethereum-rpc.publicnode.com",
		"https://rpc.eth.gateway.fm",
		"https://mainnet.gateway.tenderly.co",
		"https://gateway.tenderly.co/public/mainnet",
		"https://uk.rpc.blxrbdn.com",
		"https://go.getblock.io/d9fde9abc97545f4887f56ae54f3c2c0",
		"https://singapore.rpc.blxrbdn.com",
		"https://eth.meowrpc.com",
		"https://rpc.mevblocker.io/fast",
		"https://rpc.mevblocker.io",
		"https://eth.rpc.blxrbdn.com",
		"https://virginia.rpc.blxrbdn.com",
		"https://core.gashawk.io/rpc",
		"https://api.stateless.solutions/ethereum/v1/demo",
		"https://api.tatum.io/v3/blockchain/node/ethereum-mainnet",
		"https://rpc.flashbots.net/fast",
		"https://rpc.flashbots.net",
		"https://eth.nodeconnect.org",
		"https://eth-pokt.nodies.app",
		"https://gateway.subquery.network/rpc/eth",
		"https://rpc.mevblocker.io/fullprivacy",
		"https://rpc.mevblocker.io/noreverts",
		"https://ethereum.rpc.subquery.network/public",
		"https://rpc.builder0x69.io",
		"https://rpc.blocknative.com/boost",
		"https://services.tokenview.io/vipapi/nodeservice/eth?apikey=qVHq2o6jpaakcw3lRstl",
		"https://rpc.tenderly.co/fork/c63af728-a183-4cfb-b24e-a92801463484",
		"https://eth-mainnet.g.alchemy.com/v2/demo",
		"https://api.zmok.io/mainnet/oaen6dy8ff6hju9k",
		"https://openapi.bitstack.com/v1/wNFxbiJyQsSeLrX8RRCHi7NpRxrlErZk/DjShIqLishPCTB9HiMkPHXjUM9CNM9Na/ETH/mainnet",
		"https://endpoints.omniatech.io/v1/eth/mainnet/public",
		"https://eth-mainnet.nodereal.io/v1/1659dfb40aa24bbb8153a677b98064d7",
	}

	// Canale per inviare e ricevere risultati
	resultChannel := make(chan Account)

	// Goroutine per la comunicazione con il canale principale e l'elaborazione dei risultati
	go func() {
		for account := range resultChannel {
			if account.Balance > 0 {
				fmt.Println("Account found with balance greater than 0:")
				fmt.Println("Private Key:", account.PrivateKey)
				fmt.Println("Public Key:", account.PublicKey)
				fmt.Println("Address:", account.Address)
				fmt.Println("Balance:", account.Balance)

				// Salva l'account nel database
				if err := db.Create(&account).Error; err != nil {
					fmt.Println("error during create account:", err)
				}
			}
		}
	}()

	// Mutex per la gestione thread-safe delle URL utilizzate
	var mu sync.Mutex
	usedUrls := make(map[string]bool) // Mappa per tenere traccia delle URL utilizzate

	// Funzione per segnare una URL come utilizzata in modo thread-safe
	markUsed := func(url string) {
		mu.Lock()
		defer mu.Unlock()
		usedUrls[url] = true
	}

	// Funzione per verificare se una URL è stata utilizzata in modo thread-safe
	isUsed := func(url string) bool {
		mu.Lock()
		defer mu.Unlock()
		return usedUrls[url]
	}

	// Canale per limitare il numero di goroutine attive
	semaphore := make(chan struct{}, len(urls))

	// Contatore dei successi
	var successCount int

	var wg sync.WaitGroup
	for _, url := range urls {
		semaphore <- struct{}{} // Acquisisci un semaforo per avviare una nuova goroutine
		wg.Add(1)
		go func(url string) {
			defer func() {
				<-semaphore // Rilascia il semaforo
				wg.Done()
			}()

			privateKey, err := generatePrivateKey()
			if err != nil {
				fmt.Println("error generating private key:", err)
				return
			}

			address, publicKeyBytes, err := getAddressAndPublicKey(privateKey)
			if err != nil {
				fmt.Println("error generating address and public key:", err)
				return
			}

			balanceInEther, err := getAccountBalance(url, address)
			if err != nil {
				fmt.Printf("error getting account balance from %s: %v\n", url, err)
				// Se c'è stato un errore, segna questa URL come utilizzata
				markUsed(url)
				// Cerca la prima URL libera e utilizzala
				for _, u := range urls {
					if !isUsed(u) {
						balanceInEther, err = getAccountBalance(u, address)
						if err == nil {
							url = u
							break
						}
					}
				}
			} else {
				successCount++ // Incrementa il conteggio dei successi
			}

			resultChannel <- Account{
				PrivateKey: hexutil.Encode(crypto.FromECDSA(privateKey)),
				PublicKey:  hexutil.Encode(publicKeyBytes),
				Address:    address,
				Balance:    balanceInEther,
			}
		}(url)
	}

	// Avvia una goroutine per visualizzare il conteggio dei successi
	go func() {
		for {
			fmt.Printf("Numero di successi finora: %d\n", successCount)
			time.Sleep(5 * time.Minute) // Aggiorna ogni 5 minuti
		}
	}()

	wg.Wait()            // Aspetta che tutte le goroutine siano completate
	close(resultChannel) // Chiudi il canale dei risultati per terminare la goroutine di elaborazione
}

func initDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("accounts.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
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

func getAccountBalance(url string, address string) (float64, error) {
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
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	if err = json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	var weiBalance string
	if result["result"] != nil {
		if str, ok := result["result"].(string); ok {
			weiBalance = str
		}
	}

	if weiBalance == "" {
		return 0.00, nil
	}

	balanceInWei, err := strconv.ParseInt(weiBalance, 0, 64)
	if err != nil {
		return 0, err
	}
	balanceInEther := float64(balanceInWei) / math.Pow10(18)
	return balanceInEther, nil
}
