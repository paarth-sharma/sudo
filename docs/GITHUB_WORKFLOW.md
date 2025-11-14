# GitHub Workflow Guide for SUDO Kanban

Complete guide for managing your GitHub repository with production deployments to Railway.

---

## 📋 Table of Contents

- [Repository Structure](#repository-structure)
- [Branching Strategy](#branching-strategy)
- [Initial Setup](#initial-setup)
- [Daily Workflow](#daily-workflow)
- [Railway Deployment](#railway-deployment)
- [Branch Protection Rules](#branch-protection-rules)
- [Git Commands Cheat Sheet](#git-commands-cheat-sheet)
- [Troubleshooting](#troubleshooting)

---

## Repository Structure

```
main (production)     ← Railway deploys from here
  └── develop         ← Your active development branch
      └── feature/*   ← Short-lived feature branches (optional)
```

### Branch Purposes

| Branch | Purpose | Auto-deploys? |
|--------|---------|---------------|
| `main` | Production-ready code | ✅ Railway Auto-deploys |
| `develop` | Integration & testing | ❌ Manual testing |
| `feature/*` | Individual features (optional) | ❌ Local only |

---

## Branching Strategy

We use **GitHub Flow with a Develop Branch** - a simplified strategy perfect for solo developers.

### Why This Strategy?

✅ **Simple** - Only 2-3 branches to manage
✅ **Safe** - Production is protected
✅ **Flexible** - Easy to hotfix production
✅ **CI/CD Ready** - Automatic Railway deployments
✅ **Solo-friendly** - No complex merge conflicts

### The Flow

```
1. Work on develop branch
2. Test thoroughly locally
3. Merge to main when ready
4. Railway auto-deploys
```

---

## Initial Setup

### 1. Initialize Git Repository

```bash
# Navigate to your project
cd /c/Users/PaarthSharma/.vscode/sudo

# Initialize git (if not already done)
git init

# Add all files
git add .

# Create initial commit
git commit -m "Initial commit: SUDO Kanban v1.0

- Email-based OTP authentication
- Multi-board kanban interface
- Real-time WebSocket collaboration
- Nested boards support
- Dark/Light mode toggle
- Profile management
- Account deletion feature
- Military-grade encryption (AES-256-GCM)
- Production-ready self-hosting setup
- Comprehensive documentation"
```

### 2. Create GitHub Repository

```bash
# Create repo on GitHub.com:
# 1. Go to https://github.com/new
# 2. Name: "sudo" or "sudo-kanban"
# 3. Description: "Free, source-available kanban board built with Go, HTMX, and TailwindCSS"
# 4. Public repository
# 5. Do NOT initialize with README (you have one)
# 6. Do NOT add .gitignore (you have one)
# 7. Choose "MIT License with Commons Clause" if available, or add LICENSE file later
# 8. Click "Create repository"
```

### 3. Connect Local to GitHub

```bash
# Add remote (replace with your username)
git remote add origin https://github.com/paarth-sharma/sudo.git

# Verify remote
git remote -v

# Rename master to main (if needed)
git branch -M main

# Push to GitHub
git push -u origin main
```

### 4. Create Develop Branch

```bash
# Create and switch to develop branch
git checkout -b develop

# Push develop branch to GitHub
git push -u origin develop

# Verify branches
git branch -a
```

### 5. Set Default Branch (Optional)

On GitHub.com:
1. Go to your repo → Settings → Branches
2. Change default branch to `develop`
3. This makes PRs target `develop` by default

---

## Daily Workflow

### Starting Your Day

```bash
# Make sure you're on develop
git checkout develop

# Pull latest changes (sync with GitHub)
git pull origin develop

# Check status
git status
```

### Making Changes

```bash
# Work on develop branch directly
git checkout develop

# Make your changes...
# Edit files, add features, fix bugs

# Check what changed
git status
git diff

# Stage changes
git add .

# Commit with descriptive message
git commit -m "Add task completion notifications

- Implement WebSocket notifications for task completion
- Add visual feedback in UI
- Update real-time service for broadcast"

# Push to GitHub
git push origin develop
```

### Testing Before Production

```bash
# Test thoroughly on develop branch:
# 1. Run local development server
air  # or go run cmd/server/main.go

# 2. Test all features
# - Authentication
# - Board creation
# - Task management
# - Real-time updates
# - Dark mode toggle

# 3. Run test suite
go test ./... -v

# 4. Check for issues
# - No console errors
# - All features working
# - Performance acceptable
```

### Deploying to Production

When you're ready to deploy:

```bash
# 1. Ensure you're on develop and everything is pushed
git checkout develop
git status  # Should show "nothing to commit"

# 2. Switch to main
git checkout main

# 3. Pull latest main (just in case)
git pull origin main

# 4. Merge develop into main
git merge develop

# 5. Push to main (triggers Railway deployment)
git push origin main

# 6. Switch back to develop for continued work
git checkout develop
```

Railway will automatically deploy within 2-3 minutes!

---

## Railway Deployment

### Configure Railway

1. **Connect GitHub Repository:**
   - Go to Railway dashboard
   - Create new project → "Deploy from GitHub repo"
   - Select `sudo` repository
   - Grant permissions

2. **Configure Production Deployment:**
   ```
   Settings → Service Settings:

   Branch: main
   Build Command: (auto-detected from Dockerfile)
   Start Command: (auto-detected from Dockerfile)

   Environment Variables:
   - SUPABASE_URL=your-url
   - SUPABASE_SERVICE_KEY=your-key
   - JWT_SECRET=your-secret
   - ENCRYPTION_MASTER_KEY=your-key
   - RESEND_API_KEY=your-key
   - FROM_EMAIL=noreply@yourdomain.com
   - APP_ENV=production
   ```

3. **Verify Auto-deploy Settings:**
   ```
   Settings → Deploys:
   ✅ Automatic deployments: Enabled
   Branch: main
   ```

### Monitor Deployments

```bash
# After pushing to main, watch Railway:
# 1. Railway dashboard → Your service
# 2. Click "Deployments" tab
# 3. Watch build logs in real-time
# 4. Wait for "Deployment successful"
# 5. Test your production URL
```

---

## Branch Protection Rules

Protect your production branch from accidents.

### Set Up on GitHub

1. Go to: `Settings → Branches → Add branch protection rule`

2. **For `main` branch:**
   ```
   Branch name pattern: main

   ✅ Require a pull request before merging
      - Required approvals: 0 (solo dev, optional)
   ✅ Require status checks to pass
      - Require branches to be up to date
   ✅ Do not allow bypassing the above settings

   ⚠️ For solo dev, you CAN allow yourself to bypass
      but it's good discipline to use PRs
   ```

3. **For `develop` branch (optional):**
   ```
   Branch name pattern: develop

   ✅ Require linear history (no merge commits)
   ✅ Include administrators (apply rules to you too)
   ```

### Using Pull Requests (Recommended)

Even as a solo developer, PRs are valuable:

```bash
# Instead of direct merge to main:
# 1. Push develop to GitHub
git push origin develop

# 2. On GitHub, create PR:
#    develop → main
#
# 3. Review your own changes
# 4. Check deployment preview (if configured)
# 5. Merge PR (triggers Railway deploy)
```

**Benefits:**
- Review your changes before production
- Track deployment history
- Automatic changelog
- Can revert easily

---

## Git Commands Cheat Sheet

### Daily Commands

```bash
# Check current branch
git branch

# Switch branches
git checkout develop
git checkout main

# View status
git status

# View changes
git diff

# Stage all changes
git add .

# Stage specific file
git add path/to/file.go

# Commit
git commit -m "Your message here"

# Push
git push origin develop
git push origin main

# Pull latest
git pull origin develop
git pull origin main

# View commit history
git log --oneline --graph --all

# View remote info
git remote -v
```

### Merging & Syncing

```bash
# Merge develop into main
git checkout main
git merge develop

# Merge main into develop (get production fixes)
git checkout develop
git merge main

# Abort a merge
git merge --abort

# Pull and rebase (cleaner history)
git pull --rebase origin develop
```

### Undo & Fix

```bash
# Discard local changes
git checkout -- filename

# Unstage file
git reset HEAD filename

# Undo last commit (keep changes)
git reset --soft HEAD~1

# Undo last commit (discard changes)
git reset --hard HEAD~1

# Revert a commit (safe for pushed commits)
git revert <commit-hash>

# View commit to revert
git log --oneline
git revert abc123
```

### Hotfixes (Emergency Production Fixes)

```bash
# If you need to fix production ASAP:

# 1. Create hotfix branch from main
git checkout main
git checkout -b hotfix/critical-bug-fix

# 2. Make the fix
# ... edit files ...

# 3. Commit
git commit -m "Hotfix: Fix critical login bug"

# 4. Merge to main
git checkout main
git merge hotfix/critical-bug-fix
git push origin main  # Deploys to Railway

# 5. Merge to develop too (so it doesn't get lost)
git checkout develop
git merge hotfix/critical-bug-fix
git push origin develop

# 6. Delete hotfix branch
git branch -d hotfix/critical-bug-fix
```

---

## Advanced Workflows (Optional)

### Feature Branches

If working on a large feature:

```bash
# Create feature branch from develop
git checkout develop
git checkout -b feature/task-comments

# Work on feature
# ... make changes ...
git add .
git commit -m "Add comment system to tasks"

# When done, merge to develop
git checkout develop
git merge feature/task-comments

# Delete feature branch
git branch -d feature/task-comments

# Push develop
git push origin develop
```

### Release Tags

Mark production releases:

```bash
# After deploying to main, tag the release
git checkout main
git tag -a v1.0.0 -m "Release v1.0.0

- Email-based authentication
- Real-time collaboration
- Nested boards
- Dark mode
- Full documentation"

# Push tags to GitHub
git push origin --tags

# View tags
git tag -l
```

### Stashing (Save Work for Later)

```bash
# Save current changes without committing
git stash

# List stashes
git stash list

# Apply stashed changes
git stash apply

# Apply and remove from stash
git stash pop

# Name your stash
git stash save "WIP: Task notifications feature"
```

---

## GitHub Best Practices

### Commit Message Guidelines

**Format:**
```
<type>: <subject>

<optional body>

<optional footer>
```

**Types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Formatting (no code change)
- `refactor:` - Code restructuring
- `test:` - Adding tests
- `chore:` - Maintenance tasks

**Examples:**

```bash
# Good commit messages:
git commit -m "feat: Add email notifications for task assignments"
git commit -m "fix: Resolve WebSocket connection timeout issue"
git commit -m "docs: Update self-hosting guide with nginx config"
git commit -m "refactor: Simplify authentication middleware logic"

# Bad commit messages:
git commit -m "changes"
git commit -m "fix stuff"
git commit -m "update"
```

### .gitignore Best Practices

Already configured! Your `.gitignore` now properly excludes:
- ✅ Environment files (.env*)
- ✅ Secrets (keys, certificates)
- ✅ Build artifacts (bin/, tmp/)
- ✅ Dependencies (node_modules/)
- ✅ IDE settings (personal .vscode configs)
- ✅ Generated files (*_templ.go)
- ✅ OS files (.DS_Store, Thumbs.db)

**What IS committed:**
- ✅ Source code (*.go, *.templ)
- ✅ Static files (CSS, JS)
- ✅ Documentation (*.md)
- ✅ Configuration templates (.env.example)
- ✅ Docker files (Dockerfile, docker-compose.yml)
- ✅ Learning resources (learning/)

---

## Troubleshooting

### Issue: "Your branch is behind 'origin/develop'"

```bash
# Pull latest changes
git pull origin develop

# If you have uncommitted changes:
git stash
git pull origin develop
git stash pop
```

### Issue: Merge Conflicts

```bash
# During merge, if conflicts occur:
# 1. Git will tell you which files have conflicts
git status

# 2. Open each file and look for:
<<<<<<< HEAD
your changes
=======
their changes
>>>>>>> develop

# 3. Edit to keep what you want, remove markers

# 4. Stage resolved files
git add conflicted-file.go

# 5. Complete merge
git commit -m "Merge develop into main - resolve conflicts"
```

### Issue: Pushed Wrong Code to Main

```bash
# If Railway already deployed, revert:
git checkout main
git revert HEAD
git push origin main

# This creates a new commit that undoes the bad one
# Railway will deploy the revert
```

### Issue: Need to Undo Local Commits

```bash
# Remove last commit (not pushed yet)
git reset --soft HEAD~1  # Keeps changes
# or
git reset --hard HEAD~1  # Discards changes

# If already pushed, use revert instead:
git revert HEAD
git push origin develop
```

### Issue: Accidentally Committed Secrets

```bash
# Remove from history (BEFORE pushing):
git rm --cached .env
git commit --amend -m "Remove .env file"

# If already pushed, you need to:
# 1. Rotate all secrets immediately!
# 2. Force push (dangerous):
git push --force origin develop

# Better: Use BFG Repo Cleaner or git-filter-repo
# See: https://rtyley.github.io/bfg-repo-cleaner/
```

### Issue: Railway Not Deploying

1. **Check Branch:**
   ```bash
   # Verify you pushed to main
   git branch
   git log origin/main --oneline -5
   ```

2. **Check Railway Settings:**
   - Dashboard → Service → Settings
   - Verify branch is set to `main`
   - Check deployment logs for errors

3. **Check Dockerfile:**
   ```bash
   # Test Dockerfile locally
   docker build -t sudo-test .
   docker run -p 8080:8080 --env-file .env sudo-test
   ```

---

## Quick Reference

### Your Typical Day

```bash
# Morning
git checkout develop
git pull origin develop

# Work
# ... make changes ...
git add .
git commit -m "feat: Add new feature"
git push origin develop

# Test locally
air

# When ready for production
git checkout main
git merge develop
git push origin main  # ← Railway deploys!
git checkout develop
```

### Emergency Hotfix

```bash
git checkout main
git checkout -b hotfix/urgent-fix
# ... fix bug ...
git commit -m "fix: Critical bug"
git checkout main
git merge hotfix/urgent-fix
git push origin main  # ← Railway deploys fix
git checkout develop
git merge hotfix/urgent-fix  # ← Don't forget develop!
git push origin develop
```

---

## Additional Resources

- [GitHub Flow Guide](https://docs.github.com/en/get-started/quickstart/github-flow)
- [Railway Deployment Docs](https://docs.railway.com/guides/deployments)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Pro Git Book](https://git-scm.com/book/en/v2)

---

## Summary

Your workflow is now optimized for:

✅ **Solo Development** - Simple, no unnecessary complexity
✅ **Safe Production** - Protected main branch
✅ **Automatic Deployments** - Railway CI/CD
✅ **Easy Rollbacks** - Git history preserved
✅ **Professional Standards** - Industry best practices

**The mantra:**
> Develop on `develop`, deploy from `main`, Railway does the rest!

---

**[⬆ Back to README](README.md)** | **[Security Guide →](SECURITY.md)** | **[Self-Hosting →](SELF_HOST.md)**
