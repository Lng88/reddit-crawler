.PHONY: setup
setup:
	@echo "Checking for Homebrew..."
	@which brew > /dev/null || (echo "Installing Homebrew..." && /bin/bash -c "$$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)")
	@echo "Homebrew is installed"
	@echo "Installing dependencies with Homebrew..."
	brew install asdf
	asdf plugin add golang
	@echo "Installing GoLang..."
	asdf install golang 1.22.1
	asdf local golang 1.22.1

.PHONY: run
run:
	go mod download
	go run main.go config.go