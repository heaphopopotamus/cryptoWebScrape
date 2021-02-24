package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// todo: remove list of list
type allSitesData struct {
	CoinMarketCap	[]CoinMarketCapData
	BinanceUS 	[][]BinanceusData
}

type CoinMarketCapData struct {
	UsdPair               string
	Symbol                string
	Low24hr               string
	High24hr              string
	MarketCap             string
	FullyDilutedMarketCap string
	Volume                string
	CirculatingSupply     string
	MaxSupply             string
	BtcPair               string
	EthPair               string
	PercentChange         string
}

type BinanceusData struct {
	Pair string
	Coin string
	UsdPair	string
	Change24h string
	High24h string
	Low24h string
	MarketCap string
	Volume24h string
}

func getDoc(url string) *goquery.Document {
	resp, err := http.Get(url)
	if err != nil {
		log.Println("error getting: ", url)
		log.Println(err)
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func parseCoinmarketcapCurrencyDoc(doc *goquery.Document) CoinMarketCapData {
	var currencyData CoinMarketCapData
	// get ada/usd pair price
	doc.Find("div.priceValue___11gHJ").Each(func(i int, s *goquery.Selection) {
		currencyData.UsdPair = s.Text()
	})
	// get ada symbol
	// <small class="nameSymbol___1arQV">ADA</small>
	doc.Find("small.nameSymbol___1arQV").Each(func(i int, s *goquery.Selection) {
		currencyData.Symbol = s.Text()
	})
	// get ada low 24hr
	//<span class="highLowValue___GfyK7">$0.8330</span>
	highLowFound := 0
	doc.Find("span.highLowValue___GfyK7").Each(func(i int, s *goquery.Selection) {
		switch {
		case highLowFound == 0:
			currencyData.Low24hr = s.Text()
			highLowFound++
		case highLowFound == 1:
			currencyData.High24hr = s.Text()
			highLowFound++
		case highLowFound > 1:
			log.Println("Found more than one highLowValue___GfyK7 when looking for AdaHigh24hr and AdaLow24hr")
		}
	})
	// <div class="statsValue___2iaoZ">$27,795,775,311</div>
	marketCapFound := 0
	doc.Find("div.statsValue___2iaoZ").Each(func(i int, s *goquery.Selection) {
		switch {
		case marketCapFound == 0:
			currencyData.MarketCap = s.Text()
			marketCapFound++
		case marketCapFound == 1:
			currencyData.FullyDilutedMarketCap = s.Text()
			marketCapFound++
		case marketCapFound == 2:
			currencyData.Volume = s.Text()
			marketCapFound++
		case marketCapFound == 3:
			currencyData.CirculatingSupply = s.Text()
			marketCapFound++
		}
	})
	// get percent changed
	// <span style="background-color:var(--down-color);color:#fff;padding:5px 10px;border-radius:8px;font-size:14px;font-weight:600" class="qe1dn9-0 RYkpI"><span class="icon-Caret-down"></span>2.91<!-- -->%</span>
	doc.Find("span.qe1dn9-0").Each(func(i int, s *goquery.Selection) {
		currencyData.PercentChange = s.Text()
	})
	// get max supply
	// <div class="maxSupplyValue___1nBaS">45,000,000,000</div>
	doc.Find("div.maxSupplyValue___1nBaS").Each(func(i int, s *goquery.Selection) {
		currencyData.MaxSupply = s.Text()
	})
	// get ada pairs
	// <p class="sc-10nusm4-0 bspaAT">0.00001914 BTC</p>
	doc.Find("p.bspaAT").Each(func(i int, s *goquery.Selection) {
		switch {
		case strings.Contains(s.Text(), "BTC"):
			currencyData.BtcPair = s.Text()
		case strings.Contains(s.Text(), "ETH"):
			currencyData.EthPair = s.Text()
		}
	})
	return currencyData
}

// TODO fix column 8 issue returning - instead of string val tried title attr as well
func parseBinanceusRows(s *goquery.Selection) BinanceusData {
	var currencyData BinanceusData

	// regex to clean pairs string
	reg, err := regexp.Compile("[^a-zA-Z0-9-/]+")
	if err != nil {
		log.Fatal(err)
	}
	// <div aria-colindex="1" class="ReactVirtualized__Table__rowColumn"
	s.Find("div.ReactVirtualized__Table__rowColumn").Each(func(i int, s *goquery.Selection) {
		col, _ := s.Attr("aria-colindex")
		switch {
		case col == "2":
			processedString := reg.ReplaceAllString(s.Text(), "")
			currencyData.Pair = processedString
		case col == "3":
			currencyData.Coin = s.Text()
		case col == "4":
			currencyData.UsdPair = s.Text()
		case col == "5":
			currencyData.Change24h = s.Text()
		case col == "6":
			currencyData.High24h = s.Text()
		case col == "7":
			currencyData.Low24h = s.Text()
		case col == "8":
			currencyData.MarketCap = s.Text()
		case col == "9":
			currencyData.Volume24h = s.Text()
		}
	})
	return currencyData
}

// iterate through the table and based on the href value of each row parse the columns
func parseBinanceusMarketsDoc(doc *goquery.Document) []BinanceusData {
	var allBinanceusData []BinanceusData
	// get the row within column the switch/case for the href to pull values for usd markets
	//<a aria-label="row" tabindex="0" class="ReactVirtualized__Table__row"
	doc.Find("a.ReactVirtualized__Table__row").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		switch {
		case link == "/en/trade/ADA_USD":
			allBinanceusData = append(allBinanceusData, parseBinanceusRows(s))
		case link == "/en/trade/ETH_USD":
			allBinanceusData = append(allBinanceusData, parseBinanceusRows(s))
		case link == "/en/trade/BTC_USD":
			allBinanceusData = append(allBinanceusData, parseBinanceusRows(s))
		case link == "/en/trade/BNB_USD":
			allBinanceusData = append(allBinanceusData, parseBinanceusRows(s))
		}
	})
	return allBinanceusData
}

func main() {
	var allData allSitesData
	// Collect for coin marketcap
	// add coin names to this list must match coinmarketcap
	var coinmarketcapCurrencies = []string{"cardano", "bitcoin", "ethereum", "polkadot", "binance-coin", "litecoin", "bitcoin-cash", "dogecoin", "monero", "dash"}
	for _, currency := range coinmarketcapCurrencies {
		coinMarketBaseUrl := "https://coinmarketcap.com/currencies/"
		coinMarketCurrencyURL := coinMarketBaseUrl + currency + "/"
		coinmarketcapDoc := getDoc(coinMarketCurrencyURL)
		allData.CoinMarketCap = append(allData.CoinMarketCap, parseCoinmarketcapCurrencyDoc(coinmarketcapDoc))
	}
	// Collect for binanceus
	binanceusCurrencyURL := "https://www.binance.us/en/markets/"
	binanceDoc := getDoc(binanceusCurrencyURL)
	allData.BinanceUS = append(allData.BinanceUS, parseBinanceusMarketsDoc(binanceDoc))
	// jsonify, print to stdout, write to file
	resultsJSON, err := json.MarshalIndent(allData, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Println(string(resultsJSON))
	err = ioutil.WriteFile("currencyDetails.json", resultsJSON, os.ModePerm)
	if err != nil {
		log.Println("Issue writing json out file")
		log.Println(err)
	}
}
