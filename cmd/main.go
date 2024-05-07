package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"github.com/gorilla/mux"

	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
	sql "gorm.io/driver/mysql"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Account rappresenta un account Ethereum
type Account struct {
	PrivateKey string
	PublicKey  string
	Address    string
	Balance    float64
	gorm.Model
}

type Environment struct {
	DBUsername string `env:"DB_USERNAME,required"`
	DBPassword string `env:"DB_PASSWORD,required"`
	DBHost     string `env:"DB_HOST,required"`
	DBPort     int    `env:"DB_PORT,required"`
	DBSchema   string `env:"DB_SCHEMA,required"`
	ServerPort int    `env:"SERVER_PORT,required"`
}

func main() {

	var err error
	var e Environment
	if err = env.Parse(&e); err != nil {
		logrus.WithError(err).Panic("cannot configure environment variables")
	}
	db := initDatabase(databaseConfig{
		Username: e.DBUsername,
		Password: e.DBPassword,
		Address:  e.DBHost,
		Port:     e.DBPort,
		Name:     e.DBSchema,
	})

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err = w.Write([]byte(`"message":"running"`)); err != nil {
			return
		}
	})

	go process(db)

	if err = http.ListenAndServe(fmt.Sprintf(":%d", e.ServerPort), router); err != nil {
		panic(err)
	}
}

type databaseConfig struct {
	Username string
	Password string
	Address  string
	Port     int
	Name     string
}

func process(db *gorm.DB) {
	var err error
	urls := []string{
		"https://cloudflare-eth.com",
		"https://rpc.flashbots.net/",
		"https://rpc.ankr.com/eth",
		"https://eth-mainnet.public.blastapi.io",
		"https://api.securerpc.com/v1",
		"https://1rpc.io/eth",
		"https://ethereum.publicnode.com",
		"https://rpc.payload.de",
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
		"https://openapi.bitstack.com/v1/wNFxbiJyQsSeLrX8RRCHi7NpRxrlErZk/DjShIqLishPCTB9HiMkPHXjUM9CNM9Na/ETH/mainnet",
		"https://endpoints.omniatech.io/v1/eth/mainnet/public",
		"https://eth-mainnet.nodereal.io/v1/1659dfb40aa24bbb8153a677b98064d7",
	}

	resultChannel := make(chan Account)
	go func() {
		for account := range resultChannel {
			if account.Balance > 0 {
				logrus.WithFields(
					logrus.Fields{
						"private_key": account.PrivateKey,
						"public_key":  account.PublicKey,
						"address":     account.Address,
						"balance":     account.Balance,
					}).Infof("found account")
				if err = db.Create(&account).Error; err != nil {
					logrus.WithError(err).Errorf("error during create account")
				}
			}
		}
	}()

	var mu sync.Mutex
	usedUrls := make(map[string]bool)
	markUsed := func(url string) {
		mu.Lock()
		defer mu.Unlock()
		usedUrls[url] = true
	}
	markUnused := func(url string) {
		mu.Lock()
		defer mu.Unlock()
		usedUrls[url] = false
	}
	isUsed := func(url string) bool {
		mu.Lock()
		defer mu.Unlock()
		return usedUrls[url]
	}
	var semaphore = make(chan struct{}, len(urls))
	var wg sync.WaitGroup
	var success int64
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Printf("success: %e\n", float64(success))
			}
		}
	}()

	for {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore
		go func() {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()
			var errGoRoutine error
			var privateKey *ecdsa.PrivateKey
			if privateKey, errGoRoutine = generatePrivateKey(); errGoRoutine != nil {
				logrus.WithError(err).Errorf("error generating private key")
				return
			}
			var address string
			var publicKeyBytes []byte
			if address, publicKeyBytes, errGoRoutine = getAddressAndPublicKey(privateKey); errGoRoutine != nil {
				logrus.WithError(errGoRoutine).Errorf("error generating address and public key")
				return
			}
			var balanceInEther float64
			for {
				for _, url := range urls {
					if !isUsed(url) {
						markUsed(url)
						balanceInEther, err = getAccountBalance(url, address)
						markUnused(url)
						if err == nil {
							break
						}
					}
				}
				if err == nil {
					success++
					break
				}
			}
			resultChannel <- Account{
				PrivateKey: hexutil.Encode(crypto.FromECDSA(privateKey)),
				PublicKey:  hexutil.Encode(publicKeyBytes),
				Address:    address,
				Balance:    balanceInEther,
			}
		}()
	}
}

func initDatabase(dbConfig databaseConfig) *gorm.DB {
	configDB := mysql.Config{
		User:                 dbConfig.Username,
		Passwd:               dbConfig.Password,
		Addr:                 fmt.Sprintf("%s:%d", dbConfig.Address, dbConfig.Port),
		Net:                  "tcp",
		DBName:               dbConfig.Name,
		Loc:                  time.UTC,
		ParseTime:            true,
		AllowNativePasswords: true,
	}

	connectionString := configDB.FormatDSN()

	var db *gorm.DB
	var err error
	if db, err = gorm.Open(sql.Open(connectionString), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}); err != nil {
		panic(err)
	}

	if err = db.AutoMigrate(&Account{}); err != nil {
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
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		//logrus.WithFields(logrus.Fields{"address": address, "url": url}).Warnf("got %d status code", resp.StatusCode)
		time.Sleep(time.Second * 30)
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
