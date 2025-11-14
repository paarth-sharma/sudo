# Git Repository Setup Guide

This guide will help you push your local repository to GitHub and configure branch protection rules.

## Current Status

- ✓ Git repository initialized
- ✓ Initial commit created on `main` branch
- ✓ `develop` branch created
- ✓ GitHub Actions workflows configured
- ✓ 103 files committed (37,075 lines)

## Step 1: Create GitHub Repository

1. Go to https://github.com/new
2. Configure your repository:
   - **Repository name**: `sudo` or `sudo-kanban`
   - **Description**: `Free, source-available kanban board built with Go, HTMX, and TailwindCSS`
   - **Visibility**: Public or Private (your choice)
   - **DO NOT** initialize with README, .gitignore, or license (you already have these)
3. Click "Create repository"

## Step 2: Connect Local Repository to GitHub

Replace `YOUR_USERNAME` with your actual GitHub username:

```bash
# Add GitHub as remote
git remote add origin https://github.com/YOUR_USERNAME/sudo.git

# Verify remote was added
git remote -v

# Push main branch
git push -u origin main

# Push develop branch
git push -u origin develop

# Verify branches
git branch -a
```

## Step 3: Set Default Branch to Develop (Optional)

On GitHub:
1. Go to repository → **Settings** → **Branches**
2. Change default branch from `main` to `develop`
3. This makes pull requests target `develop` by default

## Step 4: Configure Branch Protection Rules

### Protect Main Branch (Production)

1. Go to: **Settings** → **Branches** → **Add branch protection rule**

2. **Branch name pattern**: `main`

3. Enable these settings:

   **Merge Requirements**:
   - ✓ Require a pull request before merging
   - ✓ Require approvals: 0 (optional for solo dev, increase for teams)
   - ✓ Dismiss stale pull request approvals when new commits are pushed
   - ✓ Require review from Code Owners (optional)

   **Status Checks**:
   - ✓ Require status checks to pass before merging
   - ✓ Require branches to be up to date before merging
   - Add required status checks:
     - `test` (from prod.yml workflow)
     - `lint` (from dev.yml workflow)

   **Additional Settings**:
   - ✓ Require conversation resolution before merging
   - ✓ Require linear history (cleaner git log)
   - ✓ Include administrators (enforce rules for everyone)
   - ✗ Allow force pushes (keep disabled)
   - ✗ Allow deletions (keep disabled)

4. Click **Create** to save

### Protect Develop Branch (Optional but Recommended)

1. Add another branch protection rule

2. **Branch name pattern**: `develop`

3. Enable these settings:

   **Merge Requirements**:
   - ✓ Require a pull request before merging (optional for solo dev)
   - ✓ Require approvals: 0

   **Status Checks**:
   - ✓ Require status checks to pass before merging
   - Add required status checks:
     - `test` (from dev.yml workflow)
     - `lint` (from dev.yml workflow)

   **Additional Settings**:
   - ✓ Require linear history
   - ✗ Include administrators (allow direct commits to develop)

4. Click **Create** to save

## Step 5: Configure GitHub Actions Secrets (If Needed)

If you plan to use automated deployments:

1. Go to: **Settings** → **Secrets and variables** → **Actions**

2. Add repository secrets:
   - `CODECOV_TOKEN` (optional, for code coverage)
   - Any deployment credentials (if using automated deploy)

## Step 6: Enable GitHub Actions

1. Go to: **Actions** tab in your repository
2. Click **I understand my workflows, go ahead and enable them**
3. GitHub Actions will now run on pushes and pull requests

## Branching Strategy Summary

```
main (protected)           ← Production, Railway auto-deploys
  └── develop (protected)  ← Active development, requires PR to main
      └── feature/*        ← Feature branches (optional)
```

## Daily Workflow

### Working on Features

```bash
# Start on develop
git checkout develop
git pull origin develop

# Make changes
# ... edit files ...

# Commit changes
git add .
git commit -m "feat: Add new feature"

# Push to develop
git push origin develop
```

### Deploying to Production

```bash
# Option 1: Direct merge (if no branch protection on main)
git checkout main
git merge develop
git push origin main  # Triggers Railway deployment

# Option 2: Pull Request (recommended)
# 1. Push develop to GitHub
git push origin develop

# 2. On GitHub, create PR: develop → main
# 3. Review changes
# 4. Merge PR (triggers Railway deployment)
```

### Hotfix for Production

```bash
# Create hotfix from main
git checkout main
git checkout -b hotfix/critical-bug

# Fix the bug
# ... edit files ...

git commit -m "fix: Critical bug in authentication"

# Merge to main
git checkout main
git merge hotfix/critical-bug
git push origin main  # Deploys fix

# Also merge to develop
git checkout develop
git merge hotfix/critical-bug
git push origin develop

# Delete hotfix branch
git branch -d hotfix/critical-bug
```

## CI/CD Pipeline Overview

### Development Branch (`develop`)

When you push to `develop`:
1. **Lint** - Code quality checks with golangci-lint
2. **Test** - Full test suite with race detection
3. **Build** - Compile application
4. **Coverage** - Upload coverage to Codecov (if configured)

See: `.github/workflows/dev.yml`

### Production Branch (`main`)

When you push to `main`:
1. **Test** - Full test suite before deployment
2. **Build** - Production build with optimizations
3. **Deploy** - Railway automatically deploys
4. **Docker** - Build and push Docker image to GHCR

See: `.github/workflows/prod.yml`

### Pull Requests

For all PRs to `main` or `develop`:
1. **Validate** - Format checks, tests, build
2. **Security** - Trivy vulnerability scanning
3. **Coverage** - Test coverage reporting
4. **Comment** - Automated PR comments with results

See: `.github/workflows/pr.yml`

## Verification Checklist

After setup, verify everything works:

- [ ] Repository created on GitHub
- [ ] Both `main` and `develop` branches pushed
- [ ] Branch protection rules configured
- [ ] GitHub Actions enabled and running
- [ ] First workflow runs successfully
- [ ] Default branch set (if changed)
- [ ] Railway connected to `main` branch
- [ ] First deployment successful

## Troubleshooting

### Can't Push to Protected Branch

**Error**: `Protected branch update failed`

**Solution**: Create a pull request instead of pushing directly

### GitHub Actions Not Running

1. Check: **Settings** → **Actions** → **General**
2. Ensure "Allow all actions and reusable workflows" is selected
3. Check workflow file syntax with `yamllint`

### Railway Not Deploying

1. Verify Railway is connected to the correct branch (`main`)
2. Check Railway deployment logs
3. Ensure all environment variables are set in Railway

### Merge Conflicts

```bash
# Update your branch
git checkout develop
git pull origin develop

# If merging main into develop
git merge main

# Resolve conflicts in files
# ... edit conflicting files ...

# Complete merge
git add .
git commit -m "merge: Resolve conflicts from main"
git push origin develop
```

## Best Practices

1. **Commit Often**: Small, atomic commits are easier to review and revert
2. **Write Good Messages**: Use conventional commits (feat:, fix:, docs:, etc.)
3. **Test Before Merge**: Always run tests locally before pushing
4. **Review Your Code**: Even as solo dev, review diffs before committing
5. **Keep Branches Updated**: Regularly pull from origin to stay in sync
6. **Use Pull Requests**: For main branch, always use PRs (even solo)
7. **Tag Releases**: Tag production releases with semantic versioning

## Quick Commands Reference

```bash
# Check current branch and status
git branch
git status

# Switch branches
git checkout main
git checkout develop

# Create new branch
git checkout -b feature/new-feature

# Update from remote
git pull origin develop

# View commit history
git log --oneline --graph --all

# View remote info
git remote -v

# Undo last commit (keep changes)
git reset --soft HEAD~1

# Create release tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin --tags
```

## Resources

- [GitHub Flow Guide](https://docs.github.com/en/get-started/quickstart/github-flow)
- [Branch Protection Rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Conventional Commits](https://www.conventionalcommits.org/)

---

**Status**: Repository is ready to push to GitHub!

Run the commands in Step 2 to get started.
