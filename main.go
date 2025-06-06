package main

import (
	"archive/zip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

//go:embed zones.json
var zonesData []byte

type Zone struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ZonesData struct {
	Zones map[string][]Zone `json:"zones"`
}

type DownloadJob struct {
	ZoneCode  string
	StateName string
	URL       string
	FilePath  string
}

type DownloadResult struct {
	Job    DownloadJob
	Error  error
	Status string // "downloaded", "skipped", "failed"
}

func main() {
	currentYear := time.Now().Year()
	yearStr := fmt.Sprintf("%d", currentYear)

	if err := os.MkdirAll(yearStr, 0755); err != nil {
		fmt.Printf("Error creating directory %s: %v\n", yearStr, err)
		return
	}

	var zones ZonesData
	if err := json.Unmarshal(zonesData, &zones); err != nil {
		fmt.Printf("Error parsing zones.json: %v\n", err)
		return
	}

	baseURL := "https://www.e-solat.gov.my/index.php?r=esolatApi/TakwimSolat&period=year&zone="

	var jobs []DownloadJob
	for stateName, stateZones := range zones.Zones {
		for _, zone := range stateZones {
			fileName := fmt.Sprintf("%s-%s.json", zone.Value, yearStr)
			filePath := filepath.Join(yearStr, fileName)

			jobs = append(jobs, DownloadJob{
				ZoneCode:  zone.Value,
				StateName: stateName,
				URL:       baseURL + zone.Value,
				FilePath:  filePath,
			})
		}
	}

	fmt.Printf("Found %d zones to process for year %s\n", len(jobs), yearStr)
	fmt.Println("Starting concurrent download process with 20 goroutines...")

	startTime := time.Now()

	results := processJobsConcurrently(jobs, 20)

	downloaded := 0
	skipped := 0
	failed := 0
	stateResults := make(map[string][]DownloadResult)

	for _, result := range results {
		stateResults[result.Job.StateName] = append(stateResults[result.Job.StateName], result)
		switch result.Status {
		case "downloaded":
			downloaded++
		case "skipped":
			skipped++
		case "failed":
			failed++
		}
	}

	fmt.Println("\n=== Results by State ===")
	for stateName, stateResults := range stateResults {
		fmt.Printf("\n%s:\n", stateName)
		for _, result := range stateResults {
			switch result.Status {
			case "downloaded":
				fmt.Printf("  ✓ %s downloaded\n", result.Job.ZoneCode)
			case "skipped":
				fmt.Printf("  ⏭ %s skipped (exists)\n", result.Job.ZoneCode)
			case "failed":
				fmt.Printf("  ✗ %s failed: %v\n", result.Job.ZoneCode, result.Error)
			}
		}
	}

	duration := time.Since(startTime)

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total zones: %d\n", len(jobs))
	fmt.Printf("Downloaded: %d\n", downloaded)
	fmt.Printf("Skipped (already exists): %d\n", skipped)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Total time: %v\n", duration)
	fmt.Printf("All files saved in folder: %s\n", yearStr)

	// Create ZIP file of the year folder
	if err := zipYearFolder(yearStr); err != nil {
		fmt.Printf("Error creating ZIP file: %v\n", err)
	} else {
		fmt.Printf("ZIP archive created successfully: %s/%s.zip\n", yearStr, yearStr)
	}
}

func processJobsConcurrently(jobs []DownloadJob, maxWorkers int) []DownloadResult {
	jobChan := make(chan DownloadJob, len(jobs))
	resultChan := make(chan DownloadResult, len(jobs))

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go worker(jobChan, resultChan, &wg)
	}

	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []DownloadResult
	for result := range resultChan {
		results = append(results, result)
		switch result.Status {
		case "downloaded":
			fmt.Printf("✓ Downloaded: %s\n", result.Job.ZoneCode)
		case "skipped":
			fmt.Printf("⏭ Skipped: %s (already exists)\n", result.Job.ZoneCode)
		case "failed":
			fmt.Printf("✗ Failed: %s - %v\n", result.Job.ZoneCode, result.Error)
		}
	}

	return results
}

func worker(jobChan <-chan DownloadJob, resultChan chan<- DownloadResult, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for job := range jobChan {
		result := DownloadResult{Job: job}
		if _, err := os.Stat(job.FilePath); err == nil {
			result.Status = "skipped"
			resultChan <- result
			continue
		}

		if err := downloadAndProcessFile(client, job.URL, job.FilePath); err != nil {
			result.Error = err
			result.Status = "failed"
		} else {
			result.Status = "downloaded"
		}

		resultChan <- result
		time.Sleep(100 * time.Millisecond)
	}
}

func downloadAndProcessFile(client *http.Client, url, filepath string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	delete(jsonData, "serverTime")

	cleanedJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cleaned JSON: %w", err)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = out.Write(cleanedJSON)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func zipYearFolder(folder string) error {
	zipPath := filepath.Join(folder, folder+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		relPath, err := filepath.Rel(folder, path)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(writer, file)
		return err
	})

	return err
}
