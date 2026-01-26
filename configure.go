package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

type osType struct {
	linux   bool
	windows bool
	mac     bool
}

func getOsType() osType {
	os := runtime.GOOS

	if os == "windows" {
		return osType{windows: true, linux: false, mac: false}
	} else if os == "darwin" {
		return osType{windows: false, linux: false, mac: true}
	} else {
		return osType{windows: false, linux: true, mac: false}
	}
}

func checkTrivyInstalled() bool {
	cmd := exec.Command("trivy", "--version")
	err := cmd.Run()
	return err == nil
}

func ConfigureSystem() error {
	log.Println(" Checking system configuration...")

	if checkTrivyInstalled() {
		cmd := exec.Command("trivy", "--version")
		output, _ := cmd.Output()
		log.Printf("trivy already installed: %s", strings.TrimSpace(string(output)))
		return nil
	}

	log.Println("  Trivy not found, attempting installation...")

	os := getOsType()

	if os.windows {
		return installWindows()
	} else if os.mac {
		return installMac()
	} else if os.linux {
		return installLinux()
	}

	return fmt.Errorf("unsupported operating system")
}

func installWindows() error {
	log.Println("Windows detected")

	if checkChocolateyInstalled() {
		log.Println(" Chocolatey found, installing Trivy...")

		cmd := exec.Command("powershell", "-Command", "choco", "install", "trivy", "-y")
		output, err := cmd.CombinedOutput()

		if err != nil {
			log.Printf(" Chocolatey install failed: %v", err)
			log.Printf("Output: %s", string(output))
			return printWindowsManualInstructions()
		}

		log.Println(" Trivy installed via Chocolatey")
		return verifyInstallation()
	}

	if checkScoopInstalled() {
		log.Println(" Scoop found, installing Trivy...")

		cmd := exec.Command("powershell", "-Command", "scoop", "install", "trivy")
		output, err := cmd.CombinedOutput()

		if err != nil {
			log.Printf(" Scoop install failed: %v", err)
			log.Printf("Output: %s", string(output))
			return printWindowsManualInstructions()
		}
		log.Println(" Trivy installed via Scoop")
		return verifyInstallation()
	}

	return printWindowsManualInstructions()
}

func installMac() error {
	log.Println(" macOS detected")

	if !checkHomebrewInstalled() {
		return printMacManualInstructions()
	}

	log.Println(" Homebrew found, installing Trivy...")

	cmd := exec.Command("brew", "install", "trivy")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf(" Homebrew install failed: %v", err)
		log.Printf("Output: %s", string(output))
		return printMacManualInstructions()
	}

	log.Println(" Trivy installed via Homebrew")
	return verifyInstallation()
}

func installLinux() error {
	log.Println(" Linux detected")

	log.Println(" Installing Trivy via install script...")

	cmd := exec.Command("sh", "-c",
		"curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin")

	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("❌ Install script failed: %v", err)
		log.Printf("Output: %s", string(output))

		log.Println(" Retrying with sudo...")
		cmdSudo := exec.Command("sh", "-c",
			"curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sudo sh -s -- -b /usr/local/bin")

		outputSudo, errSudo := cmdSudo.CombinedOutput()

		if errSudo != nil {
			log.Printf(" Sudo install failed: %v", errSudo)
			log.Printf("Output: %s", string(outputSudo))
			return printLinuxManualInstructions()
		}
	}

	log.Println(" Trivy installed via install script")
	return verifyInstallation()
}

func checkChocolateyInstalled() bool {
	cmd := exec.Command("powershell", "-Command", "Get-Command", "choco", "-ErrorAction", "SilentlyContinue")
	err := cmd.Run()
	return err == nil
}

func checkScoopInstalled() bool {
	cmd := exec.Command("powershell", "-Command", "Get-Command", "scoop", "-ErrorAction", "SilentlyContinue")
	err := cmd.Run()
	return err == nil
}

func checkHomebrewInstalled() bool {
	cmd := exec.Command("which", "brew")
	err := cmd.Run()
	return err == nil
}

func verifyInstallation() error {
	if checkTrivyInstalled() {
		cmd := exec.Command("trivy", "--version")
		output, _ := cmd.Output()
		log.Printf(" Installation verified: %s", strings.TrimSpace(string(output)))
		return nil
	}

	return fmt.Errorf("installation completed but Trivy still not found in PATH - restart terminal")
}

func printWindowsManualInstructions() error {
	instructions := `
╔════════════════════════════════════════════════════════════════╗
║                   MANUAL INSTALLATION REQUIRED                 ║
╚════════════════════════════════════════════════════════════════╝

Trivy is not installed and automatic installation failed.

OPTION 1: Install Chocolatey (Recommended)
------------------------------------------
1. Open PowerShell as Administrator
2. Run:
   Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

3. Then install Trivy:
   choco install trivy -y

OPTION 2: Install Scoop
-----------------------
1. Open PowerShell
2. Run:
   Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
   irm get.scoop.sh | iex

3. Then install Trivy:
   scoop install trivy

OPTION 3: Manual Download
-------------------------
1. Go to: https://github.com/aquasecurity/trivy/releases
2. Download: trivy_<version>_windows-64bit.zip
3. Extract to: C:\Program Files\trivy\
4. Add to PATH:
   - Windows Settings → System → About
   - Advanced system settings → Environment Variables
   - System variables → Path → Edit → New
   - Add: C:\Program Files\trivy

After installation, restart this application.
`
	log.Println(instructions)
	return fmt.Errorf("trivy installation required")
}

func printMacManualInstructions() error {
	instructions := `
╔════════════════════════════════════════════════════════════════╗
║                   MANUAL INSTALLATION REQUIRED                 ║
╚════════════════════════════════════════════════════════════════╝

Trivy is not installed and automatic installation failed.

OPTION 1: Install Homebrew (Recommended)
----------------------------------------
1. Open Terminal
2. Install Homebrew:
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

3. Then install Trivy:
   brew install trivy

OPTION 2: Install via Script
----------------------------
1. Open Terminal
2. Run:
   curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

After installation, restart this application.
`
	log.Println(instructions)
	return fmt.Errorf("trivy installation required")
}

func printLinuxManualInstructions() error {
	instructions := `
╔════════════════════════════════════════════════════════════════╗
║                   MANUAL INSTALLATION REQUIRED                 ║
╚════════════════════════════════════════════════════════════════╝

Trivy is not installed and automatic installation failed.

OPTION 1: Install via Script (All Distros)
------------------------------------------
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sudo sh -s -- -b /usr/local/bin

OPTION 2: Ubuntu/Debian
-----------------------
sudo apt-get install wget apt-transport-https gnupg lsb-release
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | gpg --dearmor | sudo tee /usr/share/keyrings/trivy.gpg > /dev/null
echo "deb [signed-by=/usr/share/keyrings/trivy.gpg] https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
sudo apt-get update
sudo apt-get install trivy

OPTION 3: RHEL/CentOS/Fedora
----------------------------
sudo tee /etc/yum.repos.d/trivy.repo << EOF
[trivy]
name=Trivy repository
baseurl=https://aquasecurity.github.io/trivy-repo/rpm/releases/\$basearch/
gpgcheck=1
enabled=1
gpgkey=https://aquasecurity.github.io/trivy-repo/rpm/public.key
EOF
sudo yum -y update
sudo yum -y install trivy

After installation, restart this application.
`
	log.Println(instructions)
	return fmt.Errorf("trivy installation required")
}
