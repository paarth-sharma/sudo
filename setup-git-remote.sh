#!/bin/bash

# Git Remote Setup Script for SUDO Kanban
# This script helps you connect your local repository to GitHub

set -e

echo "================================================"
echo "  SUDO Kanban - GitHub Remote Setup"
echo "================================================"
echo ""

# Check if git is initialized
if [ ! -d ".git" ]; then
    echo "Error: Not a git repository. Run this script from the project root."
    exit 1
fi

# Check if remote already exists
if git remote get-url origin &> /dev/null; then
    echo "Remote 'origin' already exists:"
    git remote -v
    echo ""
    read -p "Do you want to replace it? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git remote remove origin
        echo "Removed existing remote."
    else
        echo "Keeping existing remote. Exiting."
        exit 0
    fi
fi

# Get GitHub username
echo "Enter your GitHub username:"
read -r GITHUB_USERNAME

if [ -z "$GITHUB_USERNAME" ]; then
    echo "Error: GitHub username cannot be empty."
    exit 1
fi

# Get repository name (default: sudo)
echo ""
echo "Enter your GitHub repository name (default: sudo):"
read -r REPO_NAME
REPO_NAME=${REPO_NAME:-sudo}

# Construct remote URL
REMOTE_URL="https://github.com/${GITHUB_USERNAME}/${REPO_NAME}.git"

echo ""
echo "Setting up remote:"
echo "  URL: $REMOTE_URL"
echo ""

# Add remote
git remote add origin "$REMOTE_URL"

echo "✓ Remote 'origin' added successfully!"
echo ""

# Verify remote
echo "Verifying remote configuration:"
git remote -v
echo ""

# Ask if user wants to push now
read -p "Do you want to push branches to GitHub now? (Y/n): " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Nn]$ ]]; then
    echo ""
    echo "Pushing main branch..."
    if git push -u origin main; then
        echo "✓ Main branch pushed successfully!"
    else
        echo "✗ Failed to push main branch."
        echo "  Make sure you've created the repository on GitHub first."
        exit 1
    fi

    echo ""
    echo "Pushing develop branch..."
    if git push -u origin develop; then
        echo "✓ Develop branch pushed successfully!"
    else
        echo "✗ Failed to push develop branch."
        exit 1
    fi

    echo ""
    echo "================================================"
    echo "  ✓ Repository successfully pushed to GitHub!"
    echo "================================================"
    echo ""
    echo "Next steps:"
    echo "  1. Visit: https://github.com/${GITHUB_USERNAME}/${REPO_NAME}"
    echo "  2. Configure branch protection rules (see docs/GIT_SETUP.md)"
    echo "  3. Set up Railway deployment (if needed)"
    echo "  4. Start developing on the 'develop' branch"
    echo ""
else
    echo ""
    echo "Remote configured but not pushed."
    echo "To push manually, run:"
    echo "  git push -u origin main"
    echo "  git push -u origin develop"
    echo ""
fi

echo "For detailed setup instructions, see: docs/GIT_SETUP.md"
