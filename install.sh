#!/bin/bash

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║        gopick Installation Script      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo

if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

GOPATH=$(go env GOPATH)
GOBIN=$(go env GOBIN)

if [ -z "$GOBIN" ]; then
    GOBIN="$GOPATH/bin"
fi

echo -e "${GREEN}✓${NC} Go detected: $(go version)"
echo -e "${GREEN}✓${NC} GOPATH: $GOPATH"
echo -e "${GREEN}✓${NC} GOBIN: $GOBIN"
echo

echo -e "${YELLOW}Installing gopick...${NC}"
go install github.com/MdSadiqMd/gopick@latest

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} gopick installed successfully to $GOBIN"
else
    echo -e "${RED}✗${NC} Installation failed"
    exit 1
fi

if [[ ":$PATH:" != *":$GOBIN:"* ]]; then
    echo
    echo -e "${YELLOW}Note: $GOBIN is not in your PATH${NC}"
    echo "Adding to PATH configuration..."
    
    SHELL_NAME=$(basename "$SHELL")
    RC_FILE=""
    
    case "$SHELL_NAME" in
        bash)
            if [ -f "$HOME/.bashrc" ]; then
                RC_FILE="$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                RC_FILE="$HOME/.bash_profile"
            fi
            ;;
        zsh)
            if [ -f "$HOME/.zshrc" ]; then
                RC_FILE="$HOME/.zshrc"
            elif [ -f "$HOME/.zprofile" ]; then
                RC_FILE="$HOME/.zprofile"
            fi
            ;;
        fish)
            RC_FILE="$HOME/.config/fish/config.fish"
            ;;
        *)
            echo -e "${YELLOW}Unknown shell: $SHELL_NAME${NC}"
            echo "Please add the following line to your shell configuration file manually:"
            echo "  export PATH=\"$GOBIN:\$PATH\""
            ;;
    esac
    
    if [ -n "$RC_FILE" ]; then
        if ! grep -q "export PATH.*$GOBIN" "$RC_FILE" 2>/dev/null; then
            echo "" >> "$RC_FILE"
            echo "# Added by gopick installer" >> "$RC_FILE"
            echo "export PATH=\"$GOBIN:\$PATH\"" >> "$RC_FILE"
            echo -e "${GREEN}✓${NC} Added PATH configuration to $RC_FILE"
            echo
            echo -e "${YELLOW}Action required:${NC}"
            echo "  Run: source $RC_FILE"
            echo "  Or restart your terminal"
        else
            echo -e "${GREEN}✓${NC} PATH already configured in $RC_FILE"
        fi
    fi
else
    echo -e "${GREEN}✓${NC} $GOBIN is already in PATH"
fi

echo
if command -v gopick &> /dev/null; then
    echo -e "${GREEN}✓${NC} gopick is ready to use!"
    echo
    echo "Try running: gopick"
else
    echo -e "${YELLOW}Note:${NC} gopick installed but not yet available in current session"
    echo "Please run: source your shell configuration file or restart terminal"
fi

echo
echo -e "${BLUE}Setting up configuration...${NC}"
mkdir -p "$HOME/.config/gopick"
echo -e "${GREEN}✓${NC} Configuration directory created"

echo
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Installation Complete! 🎉          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo
echo "Quick start:"
echo "  1. Type 'gopick' to launch"
echo "  2. Search for any Go package"
echo "  3. Press [h] for help"
echo
echo "Happy coding! 🚀"
