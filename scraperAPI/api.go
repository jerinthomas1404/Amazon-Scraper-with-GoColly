package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/gorilla/mux"
)

type InnerDocument struct {
	Name         string `json:"name,omitempty"`
	ImageURL     string `json:"imageURL,omitempty"`
	Desc         string `json:"description,omitempty"`
	Price        string `json:"price,omitempty"`
	TotalReviews int    `json:"totalReviews,omitempty"`
}

type OuterDocument struct {
	URL     string        `json:"url,omitempty"`
	Product InnerDocument `json:"product,omitempty"`
}

type DBStatusObject struct {
	InsertedID    string `json:"InsertedID,omitempty"`
	MatchedCount  int    `json:"MatchedCount,omitempty"`
	ModifiedCount int    `json:"ModifiedCount,omitempty"`
}

func scraper(url string) OuterDocument {
	c := colly.NewCollector(
		colly.AllowedDomains("amazon.com", "www.amazon.com"),
	)
	c.SetRequestTimeout(120 * time.Second)

	// Callbacks as checkpoint for REQ
	// c.OnRequest(func(r *colly.Request) {
	// 	fmt.Println("Visiting", r.URL)
	// })

	// // Callbacks as checkpoint after receiving a RESP
	// c.OnResponse(func(r *colly.Response) {
	// 	fmt.Println("Response from:", r.Request.URL)
	// })

	outerDoc := OuterDocument{
		URL: url,
	}
	//go query selector - sun tong cascadia
	c.OnHTML("h1#title", func(e *colly.HTMLElement) {
		var prdName string
		prdNameEle := e.DOM.First().Find("span#productTitle").Text()
		prdName = strings.TrimSpace(prdNameEle)

		if prdName != "" {
			outerDoc.Product.Name = prdName
		} else {
			outerDoc.Product.Name = "NOT FOUND"
		}
	})

	c.OnHTML("span#acrCustomerReviewText", func(e *colly.HTMLElement) {
		var totalReviews int
		reviewCount := strings.ReplaceAll(e.DOM.First().Text(), ",", "")
		reviewCount = strings.Split(reviewCount, " ")[0]
		totalReviews, _ = strconv.Atoi(reviewCount)
		outerDoc.Product.TotalReviews = totalReviews
	})

	c.OnHTML("div#feature-bullets", func(e *colly.HTMLElement) {
		var desc string
		allDesc := e.DOM.First().Find("li")
		desc = allDesc.Text()

		if desc != "" {
			outerDoc.Product.Desc = desc
		} else {
			outerDoc.Product.Desc = "NOT FOUND!!"
		}
	})

	c.OnHTML("div#imgTagWrapperId", func(e *colly.HTMLElement) {

		var imageURL string
		imgElement := e.DOM.First().Find("img")
		imgURLS := imgElement.AttrOr("data-a-dynamic-image", "NOT FOUND!!")
		pattern, _ := regexp.Compile("https:\\/\\/.*?.jpg")
		imgs := pattern.FindAllString(imgURLS, -1)
		if len(imgs) > 1 {
			imageURL = imgs[len(imgs)-1]
		}

		if imageURL != "" {
			outerDoc.Product.ImageURL = imageURL
		} else {
			outerDoc.Product.ImageURL = "NOT FOUND!!"
		}

	})

	c.OnHTML("span.a-price.aok-align-center.reinventPricePriceToPayMargin.priceToPay", func(e *colly.HTMLElement) {
		var price string = e.DOM.Find("span").First().Text()
		if price != "" {
			outerDoc.Product.Price = price
		} else {
			outerDoc.Product.Price = "NOT FOUND!!"
		}

	})

	/*
		The below piece of code is to verify the output generated during the POC
	*/

	// c.OnScraped(func(r *colly.Response) {
	// 	fmt.Println("Finished", r.Request.URL)
	// 	js, err := json.MarshalIndent(outerDoc, "", "    ")
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Println("Printing content to STDOUT")

	// 	fmt.Println(string(js))

	// })
	c.Visit(url)

	return outerDoc
}

/*
The below handler was created for POC purpose.
*/
// func getFunc(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "Do a POST request to the API")
// }

func postFunc(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	formData := OuterDocument{}
	err := decoder.Decode(&formData)
	if err != nil {
		panic(err)
	}
	formData = scraper(formData.URL)
	productDetails, err := json.Marshal(formData)
	if err != nil {
		log.Fatal("Error while marshalling scraped data", err)
	}

	/*
		Sending the scraped data to aggregator service
	*/
	aggregatorURL := "http://aggregator:8081/aggregator"
	aggreq, err := http.NewRequest("POST", aggregatorURL, bytes.NewBuffer(productDetails))
	if err != nil {
		log.Fatal(err)
	}
	aggreq.Header.Set("content-type", "application/json")

	c := &http.Client{
		Timeout: 120 * time.Second,
	}

	aggresp, err := c.Do(aggreq)
	if err != nil {
		panic(err)
	}
	defer aggresp.Body.Close()

	var status DBStatusObject
	_ = json.NewDecoder(aggresp.Body).Decode(&status)
	if status.MatchedCount == 0 {
		fmt.Fprintf(w, "URL: %s\n Product Details inserted with ID: %s\n", formData.URL, status.InsertedID)
	} else {
		if status.ModifiedCount == 0 {
			fmt.Fprintf(w, "URL: %s\n\tSame Product exists.\n", formData.URL)
		} else {
			fmt.Fprintf(w, "URL: %s\n\tProduct updated", formData.URL)
		}
	}

}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	// router.HandleFunc("/scraper", getFunc).Methods("GET")
	router.HandleFunc("/scraper", postFunc).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", router))
}
