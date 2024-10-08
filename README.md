# MergeSentinel

**MergeSentinel** is a Go application designed to enhance the merge request (MR) process in GitLab. It listens for HTTP calls from GitLab project webhooks when a merge request action occurs. Depending on the rules configured, it will enable or disable the button used to accept the merge request, ensuring that all predefined criteria are met before a merge can be approved.

## Features

- **Webhook Listener:** Receives HTTP calls from GitLab project webhooks.
- **Rule Enforcement:** Checks if specific rules are met before allowing a merge request to be accepted.
- **Automatic Control:** Enables or disables the merge request button based on rule validation.

## Installation

To set up **MergeSentinel** locally:

1. **Clone the repository:**
   ```bash
   git clone https://github.com/cropalato/MergeSentinel.git
   ```

2. **Navigate to the project directory:**
   ```bash
   cd MergeSentinel
   ```

3. **Build the application:**
   ```bash
   # export CGO_ENABLED=0 # if you want to create a static build
   go build -a -o bin ./...
   ```

4. Run the application:
   ```bash
   ./MergeSentinel
   ```

## Configuration

**MergeSentinel** uses a JSON configuration file to manage its settings. Below is an example of the required structure:

```json
{
  "gitlab_token": "<token>",
  "gitlab_url": "https://<gitlab URL>",
  "webhook_token": "<webhook token>",
  "projects": [
    {
      "project_id": <ID>,
      "approvals": ["<user1>", "<user2>", "<user3>"],
      "webhook_token": "<webhook token>",
      "min_approv": 2
    }
  ],
  "psql_conn_url": "postgres://<user>:<pass>@<gitlab PostgreSQL fqdn>/gitlabhq_production?sslmode=disable"
}
```

### Configuration Fields

- **`gitlab_token`**: The personal access token for authenticating API requests to GitLab.
- **`gitlab_url`**: The base URL of your GitLab instance.
- **`webhook_token`**: The webhook token used in gitlab webhook calls. If not defined in project level, this one will be used.
- **`projects`**: An array of project-specific configurations:
    - **`project_id`**: The ID of the GitLab project.
    - **`approvals`**: A list of users required to approve the merge request.
    - **`min_approv`**: The minimum number of approvals required to allow the merge request to proceed.
    - **`webhook_token`**: The webhook token used in gitlab webhook calls.
- **`psql_conn_url`**: The PostgreSQL connection URL for accessing the GitLab database, including user credentials, the fully qualified domain name (FQDN) of the GitLab PostgreSQL server, and the name of the database (**`gitlabhq_production`**).

## Usage

After setting up the application, configure your GitLab project to send webhook events to the MergeSentinel server.

Example configuration:

1. In your GitLab project, navigate to **Settings > Webhooks"".
2. Add the URL where **MergeSentinel** is hosted.
3. Select the events you want to monitor, such as "Merge Request Events."
4. Save the webhook.

**MergeSentinel** will now monitor merge requests and enforce your rules.

## postgreSQL configuration

MergeSentinel will call gitlab postgreSQL server to update merge request table. It will be a SELECT and an UPDATE query, like:
```bash
SELECT * FROM merge_requests WHERE iid = 1 AND target_project_id = 1;
```
```bash
UPDATE merge_requests SET merge_status = 'cannot_be_merged', merge_error = ''Requires at least 2 approvals from [user1 user7]. WHERE iid = 1 AND target_project_id = 1;
```
You can want to create a user for that. can be something like that:
```bash
CREATE USER merge_sentinel WITH PASSWORD 'zoohoo6T';
GRANT SELECT, UPDATE (merge_status,merge_error) ON TABLE merge_requests TO merge_sentinel;
```

## Contributing

Contributions are welcome! Please follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) guidelines when making commits. Here's how you can contribute:

1. **Fork the repository.**
2. **Create a new branch:**
   - Use a descriptive name that follows the Conventional Commits format, such as `feat/add-new-feature` or `fix/bug-description`.
   - Example: 
     ```bash
     git checkout -b feat/your-feature-name
     ```
3. **Make your changes.**
   - Commit your changes using the Conventional Commits format:
     - `feat:` for new features.
     - `fix:` for bug fixes.
     - `docs:` for documentation changes.
     - `style:` for code style changes (e.g., formatting).
     - `refactor:` for code refactoring.
     - `test:` for adding or updating tests.
     - `chore:` for other changes that don't modify src or test files.
   - Example commit message:
     ```bash
     git commit -m "feat: add ability to validate merge rules"
     ```
4. **Push your changes:**
   ```bash
   git push origin feat/your-feature-name
   ```
5. **Open a pull request.**
   Please ensure your code follows the project's coding standards and includes relevant tests. Reviewers will give feedback as soon as possible.
   
## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Contact

If you have any questions, issues, or suggestions, feel free to open an issue.
