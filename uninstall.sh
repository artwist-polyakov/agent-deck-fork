#!/bin/bash
#
# Agent Deck Uninstaller
# https://github.com/asheshgoplani/agent-deck
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/asheshgoplani/agent-deck/main/uninstall.sh | bash
#
# Options:
#   --keep-data         Keep ~/.agent-deck/ (sessions, config, logs)
#   --keep-tmux-config  Keep tmux configuration
#   --non-interactive   Skip all prompts (removes everything)
#   --dry-run           Show what would be removed without removing
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
DIM='\033[2m'
NC='\033[0m' # No Color

# Defaults
KEEP_DATA=false
KEEP_TMUX_CONFIG=false
NON_INTERACTIVE=false
DRY_RUN=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --keep-data)
            KEEP_DATA=true
            shift
            ;;
        --keep-tmux-config)
            KEEP_TMUX_CONFIG=true
            shift
            ;;
        --non-interactive)
            NON_INTERACTIVE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            echo "Agent Deck Uninstaller"
            echo ""
            echo "Usage: uninstall.sh [options]"
            echo ""
            echo "Options:"
            echo "  --keep-data         Keep ~/.agent-deck/ (sessions, config, logs)"
            echo "  --keep-tmux-config  Keep tmux configuration in ~/.tmux.conf"
            echo "  --non-interactive   Skip all prompts (removes everything)"
            echo "  --dry-run           Show what would be removed without removing"
            echo "  -h, --help          Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║       Agent Deck Uninstaller           ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "${YELLOW}DRY RUN MODE - Nothing will be removed${NC}"
    echo ""
fi

# Track what we find
FOUND_ITEMS=()
HOMEBREW_INSTALLED=false

# Check for Homebrew installation
if command -v brew &> /dev/null && brew list agent-deck &> /dev/null 2>&1; then
    HOMEBREW_INSTALLED=true
    FOUND_ITEMS+=("homebrew")
    echo -e "Found: ${GREEN}Homebrew installation${NC}"
fi

# Check common binary locations
BINARY_LOCATIONS=(
    "$HOME/.local/bin/agent-deck"
    "/usr/local/bin/agent-deck"
    "$HOME/bin/agent-deck"
)

for loc in "${BINARY_LOCATIONS[@]}"; do
    if [[ -f "$loc" ]] || [[ -L "$loc" ]]; then
        FOUND_ITEMS+=("binary:$loc")
        if [[ -L "$loc" ]]; then
            TARGET=$(readlink "$loc" 2>/dev/null || echo "unknown")
            echo -e "Found: ${GREEN}Binary (symlink)${NC} at $loc"
            echo -e "       ${DIM}-> $TARGET${NC}"
        else
            echo -e "Found: ${GREEN}Binary${NC} at $loc"
        fi
    fi
done

# Check for data directory
DATA_DIR="$HOME/.agent-deck"
if [[ -d "$DATA_DIR" ]]; then
    FOUND_ITEMS+=("data")

    # Count sessions across profiles
    SESSION_COUNT=0
    PROFILE_COUNT=0
    if [[ -d "$DATA_DIR/profiles" ]]; then
        for profile_dir in "$DATA_DIR/profiles"/*/; do
            if [[ -f "${profile_dir}state.db" ]]; then
                PROFILE_COUNT=$((PROFILE_COUNT + 1))
                count=$(sqlite3 "${profile_dir}state.db" "SELECT COUNT(*) FROM instances;" 2>/dev/null || echo 0)
                SESSION_COUNT=$((SESSION_COUNT + count))
            elif [[ -f "${profile_dir}sessions.json" ]]; then
                # Legacy fallback for pre-v0.11.0 profiles
                PROFILE_COUNT=$((PROFILE_COUNT + 1))
                count=$(grep -o '"id"' "${profile_dir}sessions.json" 2>/dev/null | wc -l | tr -d ' ')
                SESSION_COUNT=$((SESSION_COUNT + count))
            fi
        done
    fi

    # Get total size
    DATA_SIZE=$(du -sh "$DATA_DIR" 2>/dev/null | cut -f1)

    echo -e "Found: ${GREEN}Data directory${NC} at $DATA_DIR"
    echo -e "       ${DIM}$PROFILE_COUNT profiles, $SESSION_COUNT sessions, $DATA_SIZE total${NC}"
fi

# Check for tmux config
TMUX_CONF="$HOME/.tmux.conf"
if [[ -f "$TMUX_CONF" ]] && grep -q "# agent-deck configuration" "$TMUX_CONF" 2>/dev/null; then
    FOUND_ITEMS+=("tmux")
    echo -e "Found: ${GREEN}tmux configuration${NC} in $TMUX_CONF"
fi

echo ""

# Nothing found?
if [[ ${#FOUND_ITEMS[@]} -eq 0 ]]; then
    echo -e "${YELLOW}Agent Deck does not appear to be installed.${NC}"
    echo ""
    echo "Checked locations:"
    for loc in "${BINARY_LOCATIONS[@]}"; do
        echo "  - $loc"
    done
    echo "  - $DATA_DIR"
    echo "  - $TMUX_CONF (for agent-deck config)"
    exit 0
fi

# Summary of what will be removed
echo -e "${BLUE}The following will be removed:${NC}"
echo ""

for item in "${FOUND_ITEMS[@]}"; do
    case "$item" in
        homebrew)
            echo -e "  ${RED}•${NC} Homebrew package: agent-deck"
            ;;
        binary:*)
            loc="${item#binary:}"
            echo -e "  ${RED}•${NC} Binary: $loc"
            ;;
        data)
            if [[ "$KEEP_DATA" == "true" ]]; then
                echo -e "  ${GREEN}•${NC} Data directory: $DATA_DIR ${YELLOW}(keeping)${NC}"
            else
                echo -e "  ${RED}•${NC} Data directory: $DATA_DIR"
                echo -e "    ${DIM}Including: sessions, logs, config${NC}"
            fi
            ;;
        tmux)
            if [[ "$KEEP_TMUX_CONFIG" == "true" ]]; then
                echo -e "  ${GREEN}•${NC} tmux config: ~/.tmux.conf ${YELLOW}(keeping)${NC}"
            else
                echo -e "  ${RED}•${NC} tmux config block in ~/.tmux.conf"
            fi
            ;;
    esac
done

echo ""

# Confirm unless non-interactive
if [[ "$NON_INTERACTIVE" != "true" && "$DRY_RUN" != "true" ]]; then
    read -p "Proceed with uninstall? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Uninstall cancelled."
        exit 0
    fi
    echo ""
fi

# Dry run stops here
if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "${YELLOW}Dry run complete. No changes made.${NC}"
    exit 0
fi

# Perform uninstallation
echo -e "${BLUE}Uninstalling...${NC}"
echo ""

# 1. Homebrew
if [[ "$HOMEBREW_INSTALLED" == "true" ]]; then
    echo -e "Removing Homebrew package..."
    brew uninstall agent-deck
    echo -e "${GREEN}✓${NC} Homebrew package removed"
fi

# 2. Binary files
for item in "${FOUND_ITEMS[@]}"; do
    if [[ "$item" == binary:* ]]; then
        loc="${item#binary:}"
        echo -e "Removing binary at $loc..."

        # Check if we need sudo
        if [[ ! -w "$(dirname "$loc")" ]]; then
            echo -e "${YELLOW}Requires sudo to remove $loc${NC}"
            sudo rm -f "$loc"
        else
            rm -f "$loc"
        fi
        echo -e "${GREEN}✓${NC} Binary removed: $loc"
    fi
done

# 3. tmux config
if [[ " ${FOUND_ITEMS[*]} " =~ " tmux " ]] && [[ "$KEEP_TMUX_CONFIG" != "true" ]]; then
    echo -e "Removing tmux configuration..."

    # Create backup
    cp "$TMUX_CONF" "$TMUX_CONF.bak.agentdeck-uninstall"

    # Remove the agent-deck config block (between markers)
    # Using sed to delete from start marker to end marker
    if [[ "$(uname)" == "Darwin" ]]; then
        # macOS sed requires different syntax
        sed -i '' '/# agent-deck configuration/,/# End agent-deck configuration/d' "$TMUX_CONF"
    else
        sed -i '/# agent-deck configuration/,/# End agent-deck configuration/d' "$TMUX_CONF"
    fi

    # Remove any trailing empty lines at end of file
    if [[ "$(uname)" == "Darwin" ]]; then
        sed -i '' -e :a -e '/^\n*$/{$d;N;ba' -e '}' "$TMUX_CONF" 2>/dev/null || true
    else
        sed -i -e :a -e '/^\n*$/{$d;N;ba' -e '}' "$TMUX_CONF" 2>/dev/null || true
    fi

    echo -e "${GREEN}✓${NC} tmux configuration removed (backup: ~/.tmux.conf.bak.agentdeck-uninstall)"
fi

# 4. Data directory
if [[ " ${FOUND_ITEMS[*]} " =~ " data " ]] && [[ "$KEEP_DATA" != "true" ]]; then
    echo -e "Removing data directory..."

    # Offer backup for non-interactive mode
    if [[ "$NON_INTERACTIVE" != "true" ]]; then
        read -p "Create backup of data before removing? [Y/n] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            BACKUP_FILE="$HOME/agent-deck-backup-$(date +%Y%m%d-%H%M%S).tar.gz"
            echo -e "Creating backup at $BACKUP_FILE..."
            tar -czf "$BACKUP_FILE" -C "$HOME" .agent-deck
            echo -e "${GREEN}✓${NC} Backup created: $BACKUP_FILE"
        fi
    fi

    rm -rf "$DATA_DIR"
    echo -e "${GREEN}✓${NC} Data directory removed: $DATA_DIR"
fi

echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Uninstall complete!                ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""

if [[ "$KEEP_DATA" == "true" ]]; then
    echo -e "${YELLOW}Note:${NC} Data directory preserved at $DATA_DIR"
    echo "      Remove manually with: rm -rf ~/.agent-deck"
fi

if [[ "$KEEP_TMUX_CONFIG" == "true" ]]; then
    echo -e "${YELLOW}Note:${NC} tmux config preserved in ~/.tmux.conf"
    echo "      Remove the '# agent-deck configuration' block manually if desired"
fi

echo ""
echo "Thank you for using Agent Deck!"
echo "Feedback: https://github.com/asheshgoplani/agent-deck/issues"
