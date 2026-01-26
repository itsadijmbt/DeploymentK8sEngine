package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

type TrivyScan struct {
	Result []struct {
		Target          string `json:Target`
		Vulnerabilities []struct {
			VulnerabilityId  string `json:VulnerabilityID`
			PkgName          string `json:PkgName`
			InstalledVersion string `json:InstalledVersion`
			Severity         string `json:Severity`
			Title            string `json:Title`
		} `json:Vulnerabilities`
	} `json:Result`
}

type Summary struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Unknown  int
	Total    int
}

func getImageUrl(image string) (string, error) {

	ecr := os.Getenv("ECR_REPO")

	if ecr == " " {
		return " ", fmt.Errorf("failed to get image")
	}

	imageUrl := ecr + ":" + image
	return imageUrl, nil
}

func (d *Daemon) scanImage(imageName string) (*Summary, error) {

	imageUrl, err := getImageUrl(imageName)
	summary := &Summary{}

	if err != nil {
		return summary, err
	}

	start := time.Now()
	cmd := exec.Command(
		"trivy",
		"image",
		imageUrl,
		"--format", "json",
		"--quiet", // suprress output
	)
	output, err := cmd.Output()
	if err != nil {
		return summary, fmt.Errorf("failed to scan image: %v", err)
	}

	var result TrivyScan
	err = json.Unmarshal(output, &result)

	for _, r := range result.Result {
		for _, v := range r.Vulnerabilities {
			switch v.Severity {

			case "CRITICAL":
				summary.Critical++
				summary.Total++

			case "HIGH":
				summary.High++
				summary.Total++

			case "MEDIUM":
				summary.Medium++
				summary.Total++

			case "LOW":
				summary.Low++
				summary.Total++

			default:
				summary.Unknown++

			}
		}
	}
	duration := time.Since(start)
	log.Printf(" Scan complete in %v: CRITICAL=%d, HIGH=%d, MEDIUM=%d, LOW=%d",
		duration, summary.Critical, summary.High, summary.Medium, summary.Low)

	return safetyOfImage(summary)

}

func safetyOfImage(summary *Summary) (*Summary, error) {

	if summary.Critical > 0 {
		return summary, fmt.Errorf("Critical Issues Detected\n")
	}
	if summary.High > 0 {
		return summary, fmt.Errorf("High Issues Detected\n")
	}

	return summary, nil
}
