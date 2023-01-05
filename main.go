package main

import(
	_ "embed"
	"fmt"
	"net/http"
	"time"
	"strings"
	"math/rand"
	"bufio"
	"os"
	"log"
	"encoding/json"
	"strconv"
	"github.com/PuerkitoBio/goquery"
)

type SearchResult struct {
	ResultRank int
	ResultURL string
	ResultTitle string
	ResultDesc string
}

var userAgents = []string {
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:56.0) Gecko/20100101 Firefox/56.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
}

// Selects random user agent from slice
func getRandomUserAgent() string {
	rand.Seed(time.Now().Unix());
	randNum := rand.Int() % len(userAgents); 
	return userAgents[randNum];
}

func getGoogleDomains() (map[string]string, error) {

	data, err := os.ReadFile("./google_domains.json");

	if err != nil {
		err := fmt.Errorf("file read error: %s", err);
		return nil, err;
	}
	
	var domains map[string]string;

	err = json.Unmarshal(data, &domains);

	if err != nil {
		return nil, err;
	}
	
	return domains, nil;
}

// Make HTTP request
func makeRequest(url string) (*http.Response, error) {
	client := &http.Client{};
	
	req, _ := http.NewRequest("GET", url, nil);

	req.Header.Set("User-Agent", getRandomUserAgent());

	res, err := client.Do(req);

	if res.StatusCode != 200 {
		err := fmt.Errorf("scaper received a non 200 status code suggesting a ban");
		return nil, err;
	}

	if err != nil {
		return nil, err
	}

	return res, nil; 
}

// Constructs valid google urls for scraping
func buildGoogleUrls(searchTerm string, countryCode string, pages int, count int, languageCode string) ([]string, error) {

	fmt.Println("Building google URLs...");

	toScrape := []string{};

	searchTerm = strings.Trim(searchTerm, " ");
	searchTerm = strings.Replace(searchTerm, " ", "+", -1);

	googleDomains, err := getGoogleDomains();

	if err != nil {
		log.Fatal(err);
	}

	domain := googleDomains[countryCode];

	if domain != "" {
		for i := 0; i < pages; i++ {
			start := i * 10;
			scrapeURL := fmt.Sprintf("%s%s&num=%d&hl=%s&start=%d&filter=0", domain, searchTerm, count, languageCode, start);
			toScrape = append(toScrape, scrapeURL);
		}
	} else {
		err := fmt.Errorf("country (%s) is currently not supported", countryCode);
 
		return nil, err;
	}

	return toScrape, nil;
}

// Scrape results from response
func googleResultParser(response *http.Response, rank int)([]SearchResult, error) {

	fmt.Println("Parsing result...");

	doc, err :=	goquery.NewDocumentFromResponse(response);

	if err != nil {
		return nil, err;
	}

	results := []SearchResult{};

	sel := doc.Find("div.g");

	rank ++;
	for i := range sel.Nodes {
		item := sel.Eq(i);
		linkTag := item.Find("a");
		link, _ := linkTag.Attr("href");
		titleTag := item.Find("h3.r");
		descriptionTag := item.Find("span.st"); 
		description := descriptionTag.Text();
		title := titleTag.Text();

		link = strings.Trim(link, " ");

		if link != "" && link != "#" && !strings.HasPrefix(link, "/") {
			result :=  SearchResult{
				rank,
				link,
				title,
				description,
			}

			results = append(results, result);
			rank ++;
		};
	}

	return results, err; 
}

// Scrape google for search term
func ScrapeGoogle(searchTerm string, countryCode string, languageCode string, pages int, count int)([]SearchResult, error) {

	fmt.Printf("Searching Google for \"%s\"...\n", searchTerm);
	
	results := []SearchResult{};

	resultCounter := 0;

	googlePages, err := buildGoogleUrls(searchTerm, countryCode, pages, count, languageCode);

	if err != nil {
		return nil, err;
	}

	for _, page := range googlePages {
		res, err := makeRequest(page);

		if err != nil {
			return nil, err;
		}

		data, err := googleResultParser(res, resultCounter);

		if err != nil {
			return nil, err;
		}

		resultCounter += len(data);

		results = append(results, data...);
		
	}

	return results, nil;
}

func main(){

	scanner := bufio.NewScanner(os.Stdin);

	var searchText string;
	var pagesToScrape int;

	fmt.Print("Please enter search term: ");
	
	for scanner.Scan() {
		if scanner.Text() != "" {
			searchText = scanner.Text();
			break;	
		}

		fmt.Print("Invalid string supplied, try again: ");
	}

	fmt.Print("How Google pages do you want to scrape? ");

	for scanner.Scan() {
		if scanner.Text() != "" {
			pages, err := strconv.Atoi(scanner.Text());

			if err == nil {
				pagesToScrape = pages;
				break;	
			}

			fmt.Print("Please supply a number, try again: ");
		}

	}

	results, err := ScrapeGoogle(searchText, "com", "en", pagesToScrape, 30);

	if err == nil {
		for _, result := range results{
			fmt.Println(result);
		}

		return;
	}
}