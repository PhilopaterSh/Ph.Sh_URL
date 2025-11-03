package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp" // Added for domain validation
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	requestDelay   = 20 * time.Second
	configDirName  = "Ph.Sh_url" // Updated config directory name
	configFileName = "config.yaml"
	retryAttempts  = 3
	retryDelay     = 5 * time.Second

	version = "1.1.6" // Tool version
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorReset  = "\033[0m"
)

// isValidDomain checks if a string is a syntactically valid domain name.
func isValidDomain(domain string) bool {
	match, _ := regexp.MatchString(`(?i)^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`, domain)
	return match
}

// cleanDomainLine removes invalid characters from a domain string.
func cleanDomainLine(line string) string {
	// This regex removes anything that is not a letter, number, dot, or hyphen.
	reg := regexp.MustCompile(`[^a-zA-Z0-9.-]`)
	cleaned := reg.ReplaceAllString(line, "")
	return strings.TrimSpace(cleaned)
}

// --- CONFIGURATION STRUCTURES ---
type Config struct {
	VirusTotalKeys []string `yaml:"virustotal"`
	AlienVaultKeys []string `yaml:"alienvault"`
	HudsonRockKeys []string `yaml:"hudsonrock"`
}

// --- API RESPONSE STRUCTURES ---
type VirusTotalResponse struct {
	UndetectedUrls [][]string `json:"undetected_urls"`
	ResponseCode   int        `json:"response_code"`
}

type AlienVaultResponse struct {
	URLList []struct {
		URL string `json:"url"`
	} `json:"url_list"`
	HasNext bool `json:"has_next"`
}

type HudsonRockResponse struct {
	Data struct {
		AllURLs []struct {
			URL string `json:"url"`
		} `json:"all_urls"`
	} `json:"data"`
}

// --- LOG MESSAGE STRUCTURE ---
type logMessage struct {
	Type    string
	Source  string
	Message string
}

// --- CONFIGURATION LOADING ---
func loadConfig(silent bool) (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", configDirName, configFileName)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if !silent {
			log.Printf("Configuration file not found at %s", configPath)
		}
		if err := createDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("could not create default config file: %w", err)
		}
		return nil, fmt.Errorf("a new configuration file has been created at %s. Please edit it to add your API keys", configPath)
	}

	if !silent {
		log.Printf("Loading configuration from %s", configPath)
	}
	data, err := os.ReadFile(configPath) // Changed ioutil.ReadFile to os.ReadFile
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	return &config, nil
}

func createDefaultConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	defaultConfig := `# Configuration file for Ph.Sh_URL
virustotal:
  - "YOUR_VT_API_KEY_1"
alienvault:
  - "YOUR_OTX_API_KEY_1"
hudsonrock:
  - "YOUR_HUDSONROCK_API_KEY_1"
`

	return os.WriteFile(path, []byte(defaultConfig), 0644) // Changed ioutil.WriteFile to os.WriteFile
}

// --- HTTP CLIENT WITH RETRY ---
func makeRequestWithRetry(req *http.Request, silent bool, logChan chan<- logMessage, source string) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < retryAttempts; i++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil {
			// Success, return response
			return resp, nil
		}

		if !silent {
			logChan <- logMessage{Type: "WARNING", Source: source, Message: fmt.Sprintf("Request failed (attempt %d/%d): %v. Retrying in %v...", i+1, retryAttempts, err, retryDelay)}
		}

		// Don't retry on client-side errors, but do on server-side or network errors.
		if resp != nil && (resp.StatusCode >= 400 && resp.StatusCode < 500) {
			if !silent {
				logChan <- logMessage{Type: "ERROR", Source: source, Message: fmt.Sprintf("Client error %d, not retrying.", resp.StatusCode)}
			}
			return resp, err // Return response as-is for client errors
		}

		time.Sleep(retryDelay)
	}
	return nil, fmt.Errorf("request failed after %d attempts: %w", retryAttempts, err)
}

// --- DATA FETCHING GOROUTINES ---

func getVirusTotalURLs(domain string, apiKeys []string, apiKeyIndex int, ch chan<- []string, logChan chan<- logMessage, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()
	if len(apiKeys) == 0 {
		return
	}
	apiKey := apiKeys[apiKeyIndex%len(apiKeys)]
	url := fmt.Sprintf("https://www.virustotal.com/vtapi/v2/domain/report?apikey=%s&domain=%s", apiKey, domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if !silent {
			logChan <- logMessage{Type: "ERROR", Source: "VT", Message: fmt.Sprintf("Failed to create request: %v", err)}
		}
		ch <- nil
		return
	}

	resp, err := makeRequestWithRetry(req, silent, logChan, "VT")
	if err != nil {
		if !silent {
			logChan <- logMessage{Type: "ERROR", Source: "VT", Message: err.Error()}
		}
		ch <- nil
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var r VirusTotalResponse
	json.Unmarshal(body, &r)
	var urls []string
	if r.ResponseCode == 1 {
		for _, item := range r.UndetectedUrls {
			if len(item) > 0 {
				urls = append(urls, item[0])
			}
		}
	}
	if !silent {
		logChan <- logMessage{Type: "SUCCESS", Source: "VT", Message: fmt.Sprintf("Found %d URLs", len(urls))}
	}
	ch <- urls
}

func getAlienVaultURLs(domain string, apiKeys []string, apiKeyIndex int, ch chan<- []string, logChan chan<- logMessage, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()
	var allUrls []string
	page := 1
	for {
		url := fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/url_list?limit=500&page=%d", domain, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			if !silent {
				logChan <- logMessage{Type: "ERROR", Source: "OTX", Message: fmt.Sprintf("Failed to create request: %v", err)}
			}
			break
		}

		if len(apiKeys) > 0 {
			apiKey := apiKeys[apiKeyIndex%len(apiKeys)]
			req.Header.Add("X-OTX-API-KEY", apiKey)
		}

		resp, err := makeRequestWithRetry(req, silent, logChan, "OTX")
		if err != nil {
			if !silent {
				logChan <- logMessage{Type: "ERROR", Source: "OTX", Message: err.Error()}
			}
			break
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var r AlienVaultResponse
		json.Unmarshal(body, &r)
		for _, item := range r.URLList {
			allUrls = append(allUrls, item.URL)
		}
		if !r.HasNext {
			break
		}
		page++
	}
	if !silent {
		logChan <- logMessage{Type: "SUCCESS", Source: "OTX", Message: fmt.Sprintf("Found %d URLs", len(allUrls))}
	}
	ch <- allUrls
}

func getWaybackURLs(domain string, ch chan<- []string, logChan chan<- logMessage, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()
	url := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=*.%s/*&output=text&fl=original&collapse=urlkey", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if !silent {
			logChan <- logMessage{Type: "ERROR", Source: "Wayback", Message: fmt.Sprintf("Failed to create request: %v", err)}
		}
		ch <- nil
		return
	}

	resp, err := makeRequestWithRetry(req, silent, logChan, "Wayback")
	if err != nil {
		if !silent {
			logChan <- logMessage{Type: "ERROR", Source: "Wayback", Message: err.Error()}
		}
		ch <- nil
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")
	// Filter out empty lines
	var urls []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			urls = append(urls, line)
		}
	}
	if !silent {
		logChan <- logMessage{Type: "SUCCESS", Source: "Wayback", Message: fmt.Sprintf("Found %d URLs", len(urls))}
	}
	ch <- urls
}

func getHudsonRockURLs(domain string, apiKeys []string, apiKeyIndex int, ch chan<- []string, logChan chan<- logMessage, wg *sync.WaitGroup, silent bool) {
	defer wg.Done()

	url := fmt.Sprintf("https://cavalier.hudsonrock.com/api/json/v2/osint-tools/search-by-domain?domain=%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if !silent {
			logChan <- logMessage{Type: "ERROR", Source: "HudsonRock", Message: fmt.Sprintf("Failed to create request: %v", err)}
		}
ch <- nil
		return
	}

	if len(apiKeys) > 0 {
		apiKey := apiKeys[apiKeyIndex%len(apiKeys)]
		req.Header.Add("key", apiKey) // Assuming the header is 'key', might need to be 'X-API-Key' or similar
	} else {
		if !silent {
			logChan <- logMessage{Type: "WARNING", Source: "HudsonRock", Message: "Processing without API key (data may be redacted)..."}
		}
	}

	resp, err := makeRequestWithRetry(req, silent, logChan, "HudsonRock")
	if err != nil {
		if !silent {
			logChan <- logMessage{Type: "ERROR", Source: "HudsonRock", Message: err.Error()}
		}
		ch <- nil
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var r HudsonRockResponse
	json.Unmarshal(body, &r)

	var urls []string
	for _, item := range r.Data.AllURLs {
		urls = append(urls, item.URL)
	}

	if !silent {
		logChan <- logMessage{Type: "SUCCESS", Source: "HudsonRock", Message: fmt.Sprintf("Found %d URLs", len(urls))}
	}
	ch <- urls
}

// --- BANNER FUNCTION ---
func showBanner() {
	fmt.Println(`
  _____   _          _____  _     
 |  __ \ | |        / ____|| |    
 | |__) || |__     | (___  | |__  
 |  ___/ | '_' \    \___ \ | '_' \ 
 | |     | | | | _  ____) || | | |
 |_|     |_| |_|(_)|_____/ |_| |_|
Built by : PhilopaterSh
# LinkedIn: https://www.linkedin.com/in/philopater-shenouda/
                              `)
}

func readLinesFromStdin() ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func readLinesFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeLinesToFile(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

// --- MAIN LOGIC ---
func main() {
	log.SetFlags(0) // Remove default log flags (including timestamp)

	outputFile := flag.String("o", "endpoints.txt", "Path to the output file.")
	inputFile := flag.String("d", "", "Path to file with domains. If empty, reads from stdin.")
	silent := flag.Bool("silent", false, "Run in silent mode, only outputting URLs to stdout.")
	excludeSources := flag.String("e", "", "Comma-separated list of sources to exclude (vt, otx, wayback, hr).")
	versionFlag := flag.Bool("version", false, "Print the version of the tool.")
	flag.Parse()

	if *versionFlag {
		fmt.Println("Ph.Sh_url version:", version)
		return
	}

	if !*silent {
		showBanner()
	}

	config, err := loadConfig(*silent)
	if err != nil {
		log.Fatalf(colorRed+"[FATAL] %v"+colorReset, err)
	}

	var domains []string
	if *inputFile != "" {
		domains, err = readLinesFromFile(*inputFile)
		if err != nil {
			log.Fatalf(colorRed+"[FATAL] Failed to read domains from file: %v"+colorReset, err)
		}
	} else {
		if !*silent {
			log.Println(colorBlue + "[INFO] Reading domains from stdin..." + colorReset)
		}
		domains, err = readLinesFromStdin()
		if err != nil {
			log.Fatalf(colorRed+"[FATAL] Failed to read domains from stdin: %v"+colorReset, err)
		}
	}

	if len(domains) == 0 {
		log.Fatal(colorRed + "[FATAL] No domains provided for scanning." + colorReset)
	}

	if !*silent {
		log.Printf(colorBlue+"[INFO] Loaded %d domains." + colorReset, len(domains))
	}

	excludedMap := make(map[string]bool)
	if *excludeSources != "" {
		for _, src := range strings.Split(*excludeSources, ",") {
			excludedMap[strings.TrimSpace(src)] = true
		}
	}

	finalUrlSet := make(map[string]struct{})
	apiKeyIndex := 0
	var failedDomains []string // New slice for failed domains

	// --- SIGNAL HANDLING ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan // Wait for a signal
		if !*silent {
			log.Println(colorYellow + "\n[WARNING] Interrupt signal received. Saving results..." + colorReset)
		}

		var finalUrls []string
		for u := range finalUrlSet {
			finalUrls = append(finalUrls, u)
		}

		if err := writeLinesToFile(*outputFile, finalUrls); err != nil {
			log.Fatalf(colorRed+"[FATAL] Failed to write URLs on interrupt: %v"+colorReset, err)
		}

		if len(failedDomains) > 0 {
			if !*silent {
				log.Printf(colorYellow+"[WARNING] %d domains failed to process and were saved to failed_domains.txt"+colorReset, len(failedDomains))
			}
			if err := writeLinesToFile("failed_domains.txt", failedDomains); err != nil {
				log.Fatalf(colorRed+"[FATAL] Failed to write failed domains to file: %v"+colorReset, err)
			}
		}

		if !*silent {
			log.Printf(colorGreen+"[SUCCESS] Results saved to %s"+colorReset, *outputFile)
		}
		os.Exit(0)
	}()

	for i, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		// Clean the domain line
		domain = cleanDomainLine(domain)
		if domain == "" { // Check again if cleaning made it empty
			if !*silent {
				log.Printf(colorYellow + "[WARNING] Skipping domain after cleaning resulted in empty string." + colorReset)
			}
			continue
		}
		// Validate domain format
		if !isValidDomain(domain) {
			if !*silent {
				log.Printf(colorYellow+"[WARNING] Skipping invalid domain format: %s"+colorReset, domain)
			}
			continue
		}

		if !*silent {
			percentage := float64(i+1) / float64(len(domains)) * 100
			log.Printf(colorBlue+"[INFO] Processing domain %d/%d (%.2f%%): %s"+colorReset, i+1, len(domains), percentage, domain)
		}
		urlChannel := make(chan []string, 4)
		logChan := make(chan logMessage, 4)
		var wg sync.WaitGroup
		var logWg sync.WaitGroup
		var domainSuccess bool // Flag to track domain success

		logWg.Add(1)
		go func() {
			defer logWg.Done()
			for msg := range logChan {
				var color string
				switch msg.Type {
				case "SUCCESS":
					color = colorGreen
				case "ERROR":
					color = colorRed
				case "WARNING":
					color = colorYellow
				default:
					color = colorReset
				}
				log.Printf("%s[%s]%s %s", color, msg.Source, colorReset, msg.Message)
			}
		}()

		runCount := 0
		if !excludedMap["vt"] {
			runCount++
			wg.Add(1)
			go getVirusTotalURLs(domain, config.VirusTotalKeys, apiKeyIndex, urlChannel, logChan, &wg, *silent)
		}
		if !excludedMap["otx"] {
			runCount++
			wg.Add(1)
			go getAlienVaultURLs(domain, config.AlienVaultKeys, apiKeyIndex, urlChannel, logChan, &wg, *silent)
		}
		if !excludedMap["wayback"] {
			runCount++
			wg.Add(1)
			go getWaybackURLs(domain, urlChannel, logChan, &wg, *silent)
		}
		if !excludedMap["hr"] {
			runCount++
			wg.Add(1)
			go getHudsonRockURLs(domain, config.HudsonRockKeys, apiKeyIndex, urlChannel, logChan, &wg, *silent)
		}

		if runCount == 0 {
			if !*silent {
				log.Printf(colorBlue+"[INFO] No sources selected for domain: %s"+colorReset, domain)
			}
			close(urlChannel)
			close(logChan)
		} else {
			wg.Wait()
			close(urlChannel)
			close(logChan)
		}

		logWg.Wait()
		for urls := range urlChannel {
			if urls != nil {
				domainSuccess = true
			}
			for _, u := range urls {
				finalUrlSet[u] = struct{}{}
			}
		}

		if !domainSuccess && runCount > 0 {
			failedDomains = append(failedDomains, domain)
		}

		apiKeyIndex++

		if i < len(domains)-1 {
			if !*silent {
				remaining := time.Duration(len(domains)-(i+1)) * requestDelay
				log.Printf(colorBlue+"[INFO] Waiting for %v before next domain. Estimated time remaining: %v"+colorReset, requestDelay, remaining)
				time.Sleep(requestDelay)
			} else {
				time.Sleep(requestDelay)
			}
		}
	}

	var finalUrls []string
	for u := range finalUrlSet {
		finalUrls = append(finalUrls, u)
	}

	if len(failedDomains) > 0 {
		if !*silent {
			log.Printf(colorYellow+"[WARNING] %d domains failed to process and were saved to failed_domains.txt"+colorReset, len(failedDomains))
		}
		if err := writeLinesToFile("failed_domains.txt", failedDomains); err != nil {
			log.Fatalf(colorRed+"[FATAL] Failed to write failed domains to file: %v"+colorReset, err)
		}
	}

	if *silent {
		for _, u := range finalUrls {
			fmt.Println(u)
		}
	} else {
		if err := writeLinesToFile(*outputFile, finalUrls); err != nil {
			log.Fatalf(colorRed+"[FATAL] Failed to write URLs: %v"+colorReset, err)
		}
		log.Printf(colorGreen+"[SUCCESS] All done! Found %d unique URLs. Results saved to %s"+colorReset, len(finalUrls), *outputFile)
	}
}

