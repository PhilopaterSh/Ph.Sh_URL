# Ph.Sh_URL - Reconnaissance and URL Collection Tool

<div align="center">

## 📖 About The Tool

`Ph.Sh_URL` is a powerful and concurrent open-source intelligence (OSINT) tool built with Go. It is designed for security researchers, penetration testers, and bug bounty hunters to automate the process of URL discovery. The tool systematically queries multiple online sources to gather a comprehensive list of URLs associated with one or more target domains, helping to uncover a wider attack surface.

### Connect with me:
<a href="https://www.linkedin.com/in/philopater-shenouda/"><img src="https://img.shields.io/badge/LinkedIn-blue?style=for-the-badge&logo=linkedin&logoColor=white" alt="LinkedIn"></a>
<a href="https://twitter.com/PhilopaterSh"><img src="https://img.shields.io/badge/X-black?style=for-the-badge&logo=x&logoColor=white" alt="X"></a>
<a href="https://medium.com/@PhilopaterSh"><img src="https://img.shields.io/badge/Medium-black?style=for-the-badge&logo=medium&logoColor=white" alt="Medium"></a>

</div>

## 🛠️ Technology Stack

<div align="center">

![Language](https://img.shields.io/badge/Language-Go-blue.svg)
![Go Version](https://img.shields.io/badge/Go_Version-1.25.1-blue.svg)
![Program Version](https://img.shields.io/badge/Version-1.1.8-blue.svg)
![Dependencies](https://img.shields.io/badge/Dependencies-gopkg.in/yaml.v3-brightgreen.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

</div>

##  Features

- **Multiple Sources**: Gathers data from 4 different sources to ensure broad coverage.
- **High Performance**: Utilizes Go's Goroutines technology to execute searches in parallel for different sources *within a single domain*, providing superior speed for each domain's lookup.
- **Flexible Input/Output**: Supports reading domains from a file or standard input (stdin), and allows specifying the output file.
- **Flexible Key Management**: Relies on a `config.yaml` file for API key management.
- **Automatic Setup**: The tool automatically creates its configuration file when needed.
- **Silent Mode**: The `-silent` option prints only URLs, ideal for scripting. When not in silent mode, logging output is concise, without timestamps, with simplified source-specific messages, and includes a domain counter (e.g., `Processing domain 5/100: example.com`).
- **Clean Results**: Deduplicates found URLs and saves them to a single text file. Also, performs enhanced validation and cleaning of input domains, skipping invalid ones with a warning.
- **Stateful Resumption**: The tool saves its progress and, upon restart, resumes from where it left off without overwriting previously found URLs, ensuring that results from multiple sessions are merged.

## 🚀 What's New in v1.1.8

- **Enhanced Resumption Logic**: The tool's ability to resume after an interruption has been tested and confirmed. If the process is stopped, it will now reliably continue from the last successfully processed domain upon restart.
- **Clarified Output File Behavior**: The tool's file-writing logic has been clarified. It is designed to produce a **complete and unique** list of URLs. At the end of each session (or upon interruption), it loads all previously found URLs, merges them with new findings, removes duplicates, and then **overwrites** the output file with the complete, clean list. This ensures you always have a comprehensive and de-duplicated dataset without manual cleanup.
- **Version Bump**: The official version is now `1.1.8`.

## 🎨 Colored Output

`Ph.Sh_URL` now uses colored output to make the logs more readable. The colors are used to indicate the status of the different operations:

- **Green**: Success messages.
- **Red**: Error messages.
- **Yellow**: Warning messages.
- **Blue**: Informational messages.

Here is an example of the new output:

```
[INFO] Loaded 3 domains.
[INFO] Processing domain 1/3 (33.33%): google.com
[OTX] Found 50000 URLs
[VT] Found 0 URLs
[Wayback] Found 3 URLs
[HudsonRock] Found 0 URLs
[INFO] Waiting for 20s before next domain. Estimated time remaining: 40s
[INFO] Processing domain 2/3 (66.67%): example.com
```

## 📈 Progress and Timer

To enhance user experience, `Ph.Sh_URL` now includes:

-   **Progress Percentage**: Displays the completion percentage as it processes domains.
-   **Estimated Time Remaining**: Shows an estimated time until all domains are processed.

Example of the new progress indication:
```
[INFO] Processing domain 1/3 (33.33%): google.com
...
[INFO] Waiting for 20s before next domain. Estimated time remaining: 40s
[INFO] Processing domain 2/3 (66.67%): example.com
...
```

## 🚦 Graceful Shutdown

`Ph.Sh_URL` supports graceful shutdown. If you interrupt the process (e.g., by pressing `Ctrl+C`), the tool will automatically save all the URLs found up to that point to the output file before exiting. This ensures that you don't lose any data even if you stop the scan midway.

## 💾 Logging, Resuming, and File Writing

`Ph.Sh_URL` is designed to be resilient and prevent data loss between sessions.

### Logging and Resuming
The tool logs its progress by saving the last successfully processed domain to a log file (`Ph.Sh_URL.log`). If the script is interrupted, it can be restarted and will automatically resume from the next domain in the list. Once the entire process is complete, the log file is automatically deleted.

### Output File Handling
To ensure a clean and unique final list, the tool follows a specific procedure for writing to the output file (e.g., `endpoints.txt`):
1.  **Load Existing Data**: At startup, the tool reads all URLs from the existing output file into memory.
2.  **Gather New Data**: It then processes the domains and collects new URLs.
3.  **Merge and De-duplicate**: All URLs (old and new) are merged, and duplicates are removed.
4.  **Overwrite with Complete Set**: Finally, the tool **overwrites** the output file with the complete, unique set of URLs.

This process guarantees that even if you stop and start the tool multiple times, you will not lose any data. The final output file will always contain a comprehensive, de-duplicated collection of all URLs found across all runs.

##  Data Sources

The tool relies on the following sources:

1.  **VirusTotal**: Requires an API key.
2.  **AlienVault OTX**: Works without a key, but using one is preferred for better results.
3.  **The Wayback Machine**: Works without a key.
4.  **Hudson Rock**: Requires an API key for full data. If no API key is provided, the tool will attempt to fetch data but will filter out common redacted/encrypted URL patterns.

---

##  Setup and Usage

### 1. First-time Setup

When running the tool for the first time, it will look for a configuration file. If not found, it will automatically create one in the following path:
- **On Linux:** `~/.config/Ph.Sh_URL/config.yaml`
- **On Windows:** `C:\Users\YourUser\.config\Ph.Sh_URL\config.yaml`

The tool will then stop and ask you to edit this file. Open the file and add your API keys in the designated sections.

```yaml
# Configuration file for Ph.Sh_URL
virustotal:
  - "YOUR_VT_API_KEY_1"

alienvault:
  - "YOUR_OTX_API_KEY_1"

hudsonrock:
  - "YOUR_HUDSONROCK_API_KEY_1"
```

### 2. Installation

To install the tool, ensure you have Go installed (version 1.16 or higher). Then, run the following command:

```bash
go install github.com/PhilopaterSh/Ph.Sh_url@latest
```

This command will compile the source code and place the executable (`Ph.Sh_url` or `Ph.Sh_url.exe` on Windows) in your `$GOPATH/bin` directory.

To make it globally accessible on Linux systems (like Kali Linux), you might want to move it to a common bin directory:

```bash
sudo mv "$(go env GOPATH)/bin/Ph.Sh_url" /usr/local/bin/
```

This ensures you can run `Ph.Sh_url` from any directory.

### 3. How to Use

You can use the tool in several ways:

**a) Via a file:**
```bash
Ph.Sh_url -d domains.txt -o found_urls.txt
```

**b) Via Standard Input (stdin):**
```bash
cat domains.txt | Ph.Sh_url -o all_urls.txt
```

**c) Silent Mode (to display results only):**
```bash
cat domains.txt | Ph.Sh_url -silent | tee urls.txt
```

**Usage Options:**
- `-d`: Path to the domains file (optional if using stdin).
- `-o`: Path to the output file (default: `endpoints.txt`).
- `-silent`: To enable silent mode.
- `-e`: Comma-separated list of sources to exclude (e.g., `vt,hr`). Available sources: `vt` (VirusTotal), `otx` (AlienVault OTX), `wayback` (The Wayback Machine), `hr` (Hudson Rock).

## 🔧 Troubleshooting

### Version not updating after `go install`

Due to caching in Go's module proxy, `go install` might not immediately fetch the latest tagged version. If you run `go install ...@latest` and see an older version being installed, you can bypass the proxy by using the `GOPRIVATE` environment variable:

```bash
GOPRIVATE=github.com/PhilopaterSh/Ph.Sh_url go install github.com/PhilopaterSh/Ph.Sh_url@latest
```

## 📋 Results

The tool saves the list of unique URLs in the file you specify via the `-o` option, or in `endpoints.txt` by default.

## 🔄 Updating the Tool

To update `Ph.Sh_url` to the latest version, simply run the `go install` command again:

```bash
go install github.com/PhilopaterSh/Ph.Sh_url@latest
```

This will download and compile the newest version, replacing your old executable. The tool's version, defined in `main.go`, should be incremented with each significant update or improvement.

To check the currently installed version, use the `-version` flag:

```bash
Ph.Sh_url -version
```
