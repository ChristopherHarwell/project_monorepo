# Personal Project Monorepo Aggregator

## Overview

This project is a Go application designed to aggregate all of your personal Git repositories from various sources (currently GitHub and GitLab) into a single monorepo. It provides tools to fetch repository information, select repositories, and integrate them into a central `monorepo` directory using either Git submodules or subtrees.

## Goal

The primary goal of this project is to simplify the management and accessibility of all personal coding projects. By consolidating them into one place, it becomes significantly easier to:

*   Get an overview of all projects.
*   Perform cross-repository operations or analysis.
*   **Reference all projects from a single source**, which is particularly useful for building a personal portfolio website. Instead of linking to multiple scattered repositories, the portfolio can simply point to or pull from this central monorepo.

## Features

*   Fetches repository information from GitHub (user repos and organization repos) and GitLab.
*   Caches repository metadata locally (`repo_cache.json`) to speed up subsequent runs.
*   Offers interactive selection of repositories to include in the monorepo.
*   Supports integration using either Git `submodule` or `subtree` methods.
*   Includes options for automatically adding all found repositories (`auto_mode`).
*   Provides functionality to update (`update_mode`) and push (`push_mode`) subtrees.
*   (Optional) Scans a local directory structure to identify existing Git repositories (`scan_local`).

## Setup

1.  **Configuration:** Create a `config.json` file in the same directory as the executable with the following structure:

    ```json
    {
      "github_token": "YOUR_GITHUB_PAT",
      "gitlab_token": "YOUR_GITLAB_PAT",
      "use_subtree": true, // or false for submodules
      "auto_mode": false,  // true to skip interactive selection
      "update_mode": false, // true to pull subtree updates
      "push_mode": false,   // true to push subtree changes
      "scan_local": false  // true to scan local directories first
    }
    ```

    *   Replace `YOUR_GITHUB_PAT` and `YOUR_GITLAB_PAT` with your Personal Access Tokens. Ensure the tokens have the necessary permissions (e.g., `repo` scope for GitHub, `read_api` for GitLab).

2.  **Build:** Compile the Go application:
    ```bash
    go build -o monorepo_aggregator .
    ```

3.  **Run:** Execute the compiled application:
    ```bash
    ./monorepo_aggregator
    ```

## Usage

1.  Run the application.
2.  If `scan_local` is true, it will first scan local directories (configure paths in `main.go` if needed) and save results to `local_repos.json`.
3.  It will then fetch remote repositories (or load from cache).
4.  If `auto_mode` is false, you will be prompted to select repositories interactively.
5.  If `auto_mode` is false and `use_subtree` wasn't set in `config.json`, you'll be asked to choose between submodules and subtrees.
6.  The application will initialize the `monorepo` directory (if it doesn't exist) and add the selected repositories into the `monorepo/repos/` subdirectory.
7.  If `update_mode` or `push_mode` are enabled (and `use_subtree` is true), it will perform subtree pull or push operations respectively.

The resulting `monorepo` directory will contain all your selected projects, ready for use. 

## Known Issues

*   **GitLab Integration:**
    *   The current implementation uses HTTPS Basic Authentication (embedding the Personal Access Token in the URL) for interacting with GitLab repositories during `add`, `pull`, and `push` operations. This method might be less reliable than SSH key authentication and could fail depending on GitLab instance settings (e.g., 2FA requirements) or token permissions/expiry.
    *   Fetching repositories from GitLab currently retrieves only the first 100 projects due to lack of pagination support.
    *   Ensure your GitLab PAT has the necessary scopes (`read_api`, `write_repository`) for the operations you intend to perform. 