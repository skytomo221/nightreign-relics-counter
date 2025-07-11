.PHONY: build run test clean package clean_dist

APP_NAME := nightreign-relics-counter

MAIN_FILE := .\main.go

DIST_DIR := dist

APP_LANG_DATA_DIR := .\tessdata

build: clean_exe
	go build -o "$(APP_NAME).exe" -ldflags="-s -w" "$(MAIN_FILE)"

run:
	go run $(MAIN_FILE)

test:
	go test ./...

package: clean_dist build
	if not exist "$(DIST_DIR)" mkdir "$(DIST_DIR)"
	if not exist "$(DIST_DIR)\tessdata" mkdir "$(DIST_DIR)\tessdata"
	copy "$(APP_NAME).exe" "$(DIST_DIR)\"
	xcopy "C:\Program Files\Tesseract-OCR\bin\*.dll" "$(DIST_DIR)\" /Y >NUL
	xcopy "$(APP_LANG_DATA_DIR)" "$(DIST_DIR)\tessdata\" /E /I /Y >NUL
	if exist "LICENSE.txt" copy "LICENSE.txt" "$(DIST_DIR)\LICENSE.txt"

clean_exe:
	if exist "$(APP_NAME).exe" del "$(APP_NAME).exe"

clean: clean_exe clean_dist

clean_dist:
	if exist "$(DIST_DIR)" rmdir /s /q "$(DIST_DIR)"
