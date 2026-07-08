package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/PhilopaterSh/Ph.Sh_url/internal/config"
	"github.com/PhilopaterSh/Ph.Sh_url/internal/domain"
	"github.com/PhilopaterSh/Ph.Sh_url/internal/logging"
	"github.com/PhilopaterSh/Ph.Sh_url/internal/sources"
)

const (
	requestDelay = 20 * time.Second
	logFileName  = "Ph.Sh_URL.log"

	version = "1.2.0" // Tool version
)

// --- BANNER FUNCTION ---
func showBanner() {
	fmt.Println(`
  _____   _           ____  _
 |  __ \ | |         / ___|| |
 | |__) || |__      | (___ | |__
 |  ___/ | '_' \    \___  \| '_' \
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

	cfg, err := config.LoadConfig(*silent)
	if err != nil {
		log.Fatalf(logging.ColorRed+"[FATAL] %v"+logging.ColorReset, err)
	}

	var domains []string
	if *inputFile != "" {
		domains, err = readLinesFromFile(*inputFile)
		if err != nil {
			log.Fatalf(logging.ColorRed+"[FATAL] Failed to read domains from file: %v"+logging.ColorReset, err)
		}
	} else {
		if !*silent {
			log.Println(logging.ColorBlue + "[INFO] Reading domains from stdin..." + logging.ColorReset)
		}
		domains, err = readLinesFromStdin()
		if err != nil {
			log.Fatalf(logging.ColorRed+"[FATAL] Failed to read domains from stdin: %v"+logging.ColorReset, err)
		}
	}

	if len(domains) == 0 {
		log.Fatal(logging.ColorRed + "[FATAL] No domains provided for scanning." + logging.ColorReset)
	}

	// Read the log file to find the last processed domain
	if _, err := os.Stat(logFileName); err == nil {
		logData, err := os.ReadFile(logFileName)
		if err == nil {
			lastProcessedDomain := strings.TrimSpace(string(logData))
			if lastProcessedDomain != "" {
				for i, d := range domains {
					if d == lastProcessedDomain {
						if !*silent {
							log.Printf(logging.ColorBlue+"[INFO] Resuming from domain: %s"+logging.ColorReset, d)
						}
						domains = domains[i:]
						break
					}
				}
			}
		}
	}

	if !*silent {
		log.Printf(logging.ColorBlue+"[INFO] Loaded %d domains."+logging.ColorReset, len(domains))
	}

	excludedMap := make(map[string]bool)
	if *excludeSources != "" {
		for _, src := range strings.Split(*excludeSources, ",") {
			excludedMap[strings.TrimSpace(src)] = true
		}
	}

	finalUrlSet := make(map[string]struct{})
	if _, err := os.Stat(*outputFile); err == nil {
		existingLines, err := readLinesFromFile(*outputFile)
		if err != nil {
			if !*silent {
				log.Printf(logging.ColorYellow+"[WARNING] Could not read existing output file: %v"+logging.ColorReset, err)
			}
		} else {
			for _, line := range existingLines {
				finalUrlSet[line] = struct{}{}
			}
			if !*silent {
				log.Printf(logging.ColorBlue+"[INFO] Loaded %d existing URLs from %s"+logging.ColorReset, len(existingLines), *outputFile)
			}
		}
	}
	var failedDomains []string // New slice for failed domains
	var resultsMu sync.Mutex   // Guards finalUrlSet and failedDomains, shared with the main loop below

	// --- SIGNAL HANDLING ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan // Wait for a signal
		if !*silent {
			log.Println(logging.ColorYellow + "\n[WARNING] Interrupt signal received. Saving results..." + logging.ColorReset)
		}

		resultsMu.Lock()
		var finalUrls []string
		for u := range finalUrlSet {
			finalUrls = append(finalUrls, u)
		}
		failedCopy := append([]string(nil), failedDomains...)
		resultsMu.Unlock()

		if err := writeLinesToFile(*outputFile, finalUrls); err != nil {
			log.Fatalf(logging.ColorRed+"[FATAL] Failed to write URLs on interrupt: %v"+logging.ColorReset, err)
		}

		if len(failedCopy) > 0 {
			if !*silent {
				log.Printf(logging.ColorYellow+"[WARNING] %d domains failed to process and were saved to failed_domains.txt"+logging.ColorReset, len(failedCopy))
			}
			if err := writeLinesToFile("failed_domains.txt", failedCopy); err != nil {
				log.Fatalf(logging.ColorRed+"[FATAL] Failed to write failed domains to file: %v"+logging.ColorReset, err)
			}
		}

		if !*silent {
			log.Printf(logging.ColorGreen+"[SUCCESS] Results saved to %s"+logging.ColorReset, *outputFile)
		}
		os.Exit(0)
	}()

	apiKeyIndex := 0
	for i, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		// Clean the domain line
		d = domain.CleanDomainLine(d)
		if d == "" { // Check again if cleaning made it empty
			if !*silent {
				log.Printf(logging.ColorYellow + "[WARNING] Skipping domain after cleaning resulted in empty string." + logging.ColorReset)
			}
			continue
		}
		// Validate domain format
		if !domain.IsValidDomain(d) {
			if !*silent {
				log.Printf(logging.ColorYellow+"[WARNING] Skipping invalid domain format: %s"+logging.ColorReset, d)
			}
			continue
		}

		if !*silent {
			percentage := float64(i+1) / float64(len(domains)) * 100
			log.Printf(logging.ColorBlue+"[INFO] Processing domain %d/%d (%.2f%%): %s"+logging.ColorReset, i+1, len(domains), percentage, d)
		}
		urlChannel := make(chan []string, 4)
		logChan := make(chan logging.Message, 4)
		var wg sync.WaitGroup
		var logWg sync.WaitGroup
		var domainSuccess bool // Flag to track domain success

		logWg.Add(1)
		go func() {
			defer logWg.Done()
			for msg := range logChan {
				var color string
				switch msg.Type {
				case "INFO":
					color = logging.ColorBlue
				case "WARNING":
					color = logging.ColorYellow
				case "ERROR":
					color = logging.ColorRed
				case "FATAL":
					color = logging.ColorRed
				case "SUCCESS":
					color = logging.ColorGreen
				default:
					color = logging.ColorReset
				}
				log.Printf("%s[%s]%s %s", color, msg.Source, logging.ColorReset, msg.Message)
			}
		}()

		runCount := 0
		if !excludedMap["vt"] {
			runCount++
			wg.Add(1)
			go sources.FetchVirusTotalURLs(d, cfg.VirusTotalKeys, apiKeyIndex, urlChannel, logChan, &wg, *silent)
		}
		if !excludedMap["otx"] {
			runCount++
			wg.Add(1)
			go sources.FetchAlienVaultURLs(d, cfg.AlienVaultKeys, apiKeyIndex, urlChannel, logChan, &wg, *silent)
		}
		if !excludedMap["wayback"] {
			runCount++
			wg.Add(1)
			go sources.FetchWaybackURLs(d, urlChannel, logChan, &wg, *silent)
		}
		if !excludedMap["hr"] {
			runCount++
			wg.Add(1)
			go sources.FetchHudsonRockURLs(d, cfg.HudsonRockKeys, apiKeyIndex, urlChannel, logChan, &wg, *silent)
		}

		if runCount == 0 {
			if !*silent {
				log.Printf(logging.ColorBlue+"[INFO] No sources selected for domain: %s"+logging.ColorReset, d)
			}
			close(urlChannel)
			close(logChan)
		} else {
			wg.Wait()
			close(urlChannel)
			close(logChan)
		}

		logWg.Wait()
		resultsMu.Lock()
		for urls := range urlChannel {
			if urls != nil {
				domainSuccess = true
			}
			for _, u := range urls {
				finalUrlSet[u] = struct{}{}
			}
		}

		if !domainSuccess && runCount > 0 {
			failedDomains = append(failedDomains, d)
		}
		resultsMu.Unlock()

		if domainSuccess {
			err := os.WriteFile(logFileName, []byte(d), 0644)
			if err != nil {
				if !*silent {
					log.Printf(logging.ColorRed+"[ERROR] Failed to write to log file: %v"+logging.ColorReset, err)
				}
			}
		}

		apiKeyIndex++

		if i < len(domains)-1 {
			time.Sleep(requestDelay)
		}

	}
	resultsMu.Lock()
	var finalUrls []string
	for u := range finalUrlSet {
		finalUrls = append(finalUrls, u)
	}
	failedCopy := append([]string(nil), failedDomains...)
	resultsMu.Unlock()

	if len(failedCopy) > 0 {
		if !*silent {
			log.Printf(logging.ColorYellow+"[WARNING] %d domains failed to process and were saved to failed_domains.txt"+logging.ColorReset, len(failedCopy))
		}
		if err := writeLinesToFile("failed_domains.txt", failedCopy); err != nil {
			log.Fatalf(logging.ColorRed+"[FATAL] Failed to write failed domains to file: %v"+logging.ColorReset, err)
		}
	}

	if *silent {
		for _, u := range finalUrls {
			fmt.Println(u)
		}
	} else {
		if err := writeLinesToFile(*outputFile, finalUrls); err != nil {
			log.Fatalf(logging.ColorRed+"[FATAL] Failed to write URLs: %v"+logging.ColorReset, err)
		}
		log.Printf(logging.ColorGreen+"[SUCCESS] All done! Found %d unique URLs. Results saved to %s"+logging.ColorReset, len(finalUrls), *outputFile)
	}

}
