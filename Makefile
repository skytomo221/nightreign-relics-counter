.PHONY: build run test clean package clean_exe clean_dist

SHELL := cmd.exe
.SHELLFLAGS := /C

APP_NAME := nightreign-relics-counter

MAIN_FILE := .\main.go

DIST_DIR := dist

TESSERACT_DIR ?= .\Tesseract-OCR

build: clean_exe
	go build -o "$(APP_NAME).exe" -ldflags="-s -w" "$(MAIN_FILE)"

run:
	go run $(MAIN_FILE)

test:
	go test ./...

package: clean_dist build
	if not exist "$(DIST_DIR)" mkdir "$(DIST_DIR)"
	if not exist "$(DIST_DIR)\data" mkdir "$(DIST_DIR)\data"
	if not exist "$(DIST_DIR)\Tesseract-OCR" mkdir "$(DIST_DIR)\Tesseract-OCR"
	copy "$(APP_NAME).exe" "$(DIST_DIR)\"
	copy "README.md" "$(DIST_DIR)\"
	copy "relics.tsv" "$(DIST_DIR)\"
	xcopy "$(TESSERACT_DIR)" "$(DIST_DIR)\Tesseract-OCR\" /E /I /Y >NUL
	if exist "LICENSE.txt" copy "LICENSE.txt" "$(DIST_DIR)\LICENSE.txt"

clean_exe:
	if exist "$(APP_NAME).exe" del "$(APP_NAME).exe"

clean: clean_exe clean_dist

clean_dist:
	if exist "$(DIST_DIR)" rmdir /s /q "$(DIST_DIR)"
