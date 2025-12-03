# GitHub PR Commit Status Checks Implementation

## Overview
This implementation adds the capability to collect CI/CD status checks (GitHub Actions, external CI like Jenkins, etc.) associated with Pull Request commits in Apache DevLake's GitHub plugin using the REST API.

## Files Created/Modified

### 1. Database Model
**File**: `backend/plugins/github/models/commit_status.go`
- Defines `GithubCommitStatus` struct to store commit status check data
- Primary keys: `ConnectionId`, `CommitSha`, `Context`
- Stores state (success/failure/pending/error), description, target URL, creator info, and timestamps
- Table name: `_tool_github_commit_statuses`

### 2. Migration Script
**File**: `backend/plugins/github/models/migrationscripts/20251203_add_commit_status_table.go`
- Creates the database table for commit statuses
- Version: `20251203000001`
- Includes all necessary fields with proper data types and constraints

**File**: `backend/plugins/github/models/migrationscripts/register.go` (modified)
- Registered the new migration script `addCommitStatusTable`

### 3. Data Collection
**File**: `backend/plugins/github/tasks/commit_status_collector.go`
- Collects commit status data from GitHub REST API
- Endpoint: `GET /repos/{owner}/{repo}/commits/{sha}/statuses`
- Queries distinct commit SHAs from `GithubPrCommit` table
- Filters by repository and connection ID
- Supports incremental collection based on PR update timestamps
- Paginated collection with 100 items per page
- Raw data table: `github_api_commit_statuses`

### 4. Data Extraction
**File**: `backend/plugins/github/tasks/commit_status_extractor.go`
- Extracts raw commit status data into the tool layer table
- Parses GitHub API response including:
  - Status state and context
  - Description and target URL
  - Creator information (ID, login, avatar)
  - Timestamps (created_at, updated_at)
- Handles "Not Found" responses gracefully

### 5. Domain Layer Conversion
**File**: `backend/plugins/github/tasks/commit_status_convertor.go`
- Converts tool layer data into domain layer `CiCDPipelineCommit` entities
- Maps GitHub status states to DevLake CICD results:
  - `success` → `RESULT_SUCCESS` / `STATUS_DONE`
  - `failure` → `RESULT_FAILURE` / `STATUS_DONE`
  - `error` → `RESULT_FAILURE` / `STATUS_DONE`
  - `pending` → `RESULT_DEFAULT` / `STATUS_IN_PROGRESS`
- Generates proper domain IDs for pipelines and commits
- Links commits to repositories

### 6. Plugin Registration
**File**: `backend/plugins/github/impl/impl.go` (modified)
- Added `&models.GithubCommitStatus{}` to `GetTablesInfo()` method
- Ensures the model is registered with the plugin system

## Architecture

### Data Flow
```
GitHub API (REST)
    ↓
Collector (commit_status_collector.go)
    ↓
Raw Data Table (github_api_commit_statuses)
    ↓
Extractor (commit_status_extractor.go)
    ↓
Tool Layer Table (_tool_github_commit_statuses)
    ↓
Converter (commit_status_convertor.go)
    ↓
Domain Layer Table (cicd_pipeline_commits)
    ↓
Grafana Dashboard
```

### Subtask Registration
All subtasks are automatically registered via `init()` functions:
- `CollectApiCommitStatusesMeta` - Collection subtask
- `ExtractApiCommitStatusesMeta` - Extraction subtask
- `ConvertCommitStatusesMeta` - Conversion subtask

They are added to `SubTaskMetaList` and will be sorted by dependency resolver in `impl.go`.

## API Endpoints Used

### GitHub REST API
- **Commit Statuses**: `GET /repos/{owner}/{repo}/commits/{sha}/statuses`
  - Returns all status checks for a specific commit
  - Includes traditional status contexts and GitHub Actions check runs
  - Paginated response

## Database Schema

### Tool Layer Table: `_tool_github_commit_statuses`
```sql
CREATE TABLE _tool_github_commit_statuses (
    connection_id BIGINT NOT NULL,
    commit_sha VARCHAR(40) NOT NULL,
    context VARCHAR(255) NOT NULL,
    state VARCHAR(100),
    description TEXT,
    target_url VARCHAR(255),
    avatar_url VARCHAR(255),
    creator_id INT,
    creator_login VARCHAR(255),
    github_created_at TIMESTAMP,
    github_updated_at TIMESTAMP,
    PRIMARY KEY (connection_id, commit_sha, context)
);
```

### Domain Layer Table: `cicd_pipeline_commits`
- Reuses existing DevLake domain table
- Links commit SHAs to pipeline executions
- Stores execution results and status

## Usage

### Running the Collection
When a GitHub repository is configured as a data source in DevLake, the commit status collection subtasks will automatically run as part of the data collection pipeline.

### Configuration
No additional configuration is required. The feature uses the existing GitHub connection settings.

### Incremental Collection
The collector supports incremental updates:
- Only collects statuses for commits from recently updated PRs
- Based on PR's `github_updated_at` timestamp
- Significantly reduces API calls for subsequent collections

## Grafana Integration

### Available Data
The commit status data can be queried from:
- Tool layer: `_tool_github_commit_statuses` - Raw GitHub status data
- Domain layer: `cicd_pipeline_commits` - Normalized CICD data

### Sample Queries

#### Get PR commit statuses
```sql
SELECT
    pr.number,
    pr.title,
    cs.commit_sha,
    cs.context,
    cs.state,
    cs.description,
    cs.target_url,
    cs.github_updated_at
FROM _tool_github_pull_requests pr
JOIN _tool_github_pr_commits pc ON pr.github_id = pc.pull_request_id
JOIN _tool_github_commit_statuses cs ON pc.commit_sha = cs.commit_sha
WHERE pr.connection_id = ? AND pr.repo_id = ?
ORDER BY pr.number, cs.github_updated_at DESC;
```

#### Get open PRs with CI status
```sql
SELECT
    pr.number,
    pr.title,
    pr.state,
    pr.head_commit_sha,
    cs.context as ci_check,
    cs.state as check_status,
    cs.target_url
FROM _tool_github_pull_requests pr
LEFT JOIN _tool_github_commit_statuses cs
    ON pr.head_commit_sha = cs.commit_sha
WHERE pr.state = 'open'
    AND pr.connection_id = ?
    AND pr.repo_id = ?;
```

#### CI success rate by context
```sql
SELECT
    cs.context,
    COUNT(*) as total_checks,
    SUM(CASE WHEN cs.state = 'success' THEN 1 ELSE 0 END) as successful,
    SUM(CASE WHEN cs.state = 'failure' THEN 1 ELSE 0 END) as failed,
    SUM(CASE WHEN cs.state = 'pending' THEN 1 ELSE 0 END) as pending,
    (SUM(CASE WHEN cs.state = 'success' THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as success_rate
FROM _tool_github_commit_statuses cs
WHERE cs.connection_id = ?
    AND cs.github_created_at >= NOW() - INTERVAL 30 DAY
GROUP BY cs.context
ORDER BY total_checks DESC;
```

## Features

### Supported Status Types
- ✅ Traditional GitHub Status API (external CI/CD systems)
- ✅ GitHub Actions status contexts
- ✅ All status states: success, failure, error, pending
- ✅ Status descriptions and target URLs
- ✅ Creator/author information

### Collection Strategy
- Efficient: Only collects for commits associated with PRs
- Incremental: Supports time-based filtering
- Scalable: Paginated API calls
- Resilient: Handles API errors gracefully

## Testing

### Verify Installation
1. Run DevLake migrations:
   ```bash
   make migrate
   ```

2. Check that the table was created:
   ```sql
   SHOW TABLES LIKE '_tool_github_commit_statuses';
   ```

3. Verify subtasks are registered:
   - Check logs for "Collect Commit Statuses"
   - Check logs for "Extract Commit Statuses"
   - Check logs for "Convert Commit Statuses"

### Test Data Collection
1. Configure a GitHub repository with active PRs
2. Run the collection pipeline
3. Verify data in `_tool_github_commit_statuses` table
4. Check that statuses appear for PR commits

## Integration Points

### Dependencies
- **Collector depends on**: `GithubPrCommit` (PR commits must be collected first)
- **Converter depends on**: `GithubCommitStatus`, `GithubRepo`

### Domain Types
- `DOMAIN_TYPE_CICD` - Continuous Integration/Deployment data
- `DOMAIN_TYPE_CODE_REVIEW` - Code review related data
- `DOMAIN_TYPE_CODE` - Source code data

## Future Enhancements

Potential improvements for future iterations:
1. **Check Runs API**: Add support for GitHub Check Runs API (`/repos/{owner}/{repo}/commits/{sha}/check-runs`) for more detailed GitHub Actions data
2. **Check Suites**: Collect check suite information for grouped status reporting
3. **Annotations**: Store check run annotations for detailed error messages
4. **Deployment Status**: Link commit statuses to deployment records
5. **Status Trends**: Calculate status trends over time for reliability metrics
6. **Flaky Test Detection**: Identify tests that frequently alternate between pass/fail

## Troubleshooting

### No statuses collected
- Verify PRs exist in the repository
- Check that PR commits were collected successfully
- Ensure GitHub API token has `repo:status` scope
- Review API rate limits

### Missing statuses
- GitHub only returns the most recent 100 statuses per commit by default
- Consider filtering or pagination strategies for repos with many statuses

### Performance issues
- Large repos with many PRs may take time to collect
- Enable incremental collection to reduce subsequent collection times
- Consider adjusting page size or concurrency settings

## References

### GitHub API Documentation
- [Commit Statuses API](https://docs.github.com/en/rest/commits/statuses)
- [Check Runs API](https://docs.github.com/en/rest/checks/runs)
- [Status Checks](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/collaborating-on-repositories-with-code-quality-features/about-status-checks)

### DevLake Documentation
- [Plugin Development Guide](https://devlake.apache.org/docs/DeveloperManuals/PluginImplementation)
- [Domain Layer Models](https://devlake.apache.org/docs/DataModels/DevLakeDomainLayerSchema)
- [GitHub Plugin](https://devlake.apache.org/docs/Plugins/github)
