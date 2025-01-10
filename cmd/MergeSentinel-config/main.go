// main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/manifoldco/promptui"
	"gitlab.com/gitlab-org/api/client-go"
)

type ProjectConfig struct {
	ProjectID    int      `json:"project_id"`
	Approvals    []string `json:"approvals"`
	WebhookToken string   `json:"webhook_token"`
	MinApprovals int      `json:"min_approv"`
}

type Config struct {
	GitlabToken  string          `json:"gitlab_token"`
	GitlabURL    string          `json:"gitlab_url"`
	WebhookToken string          `json:"webhook_token"`
	Projects     []ProjectConfig `json:"projects"`
	PsqlConnURL  string          `json:"psql_conn_url"`
}

func main() {
	// Define command line flags
	gitlabURL := flag.String("url", "", "GitLab instance URL (e.g., https://gitlab.example.com)")
	gitlabToken := flag.String("token", "", "GitLab personal access token")
	configFile := flag.String("config", "gitlab-config.json", "Path to configuration file")

	// Parse command line arguments
	flag.Parse()

	// Load existing configuration or create new
	config := loadConfig(*configFile)

	// Override config with command line arguments if provided
	if *gitlabURL != "" {
		config.GitlabURL = *gitlabURL
	}
	if *gitlabToken != "" {
		config.GitlabToken = *gitlabToken
	}

	// Validate required configuration
	if config.GitlabURL == "" || config.GitlabToken == "" {
		fmt.Println("Error: GitLab URL and token are required.")
		fmt.Println("Please provide them either through the configuration file or command line arguments:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Initialize GitLab client using config values
	git, err := gitlab.NewClient(config.GitlabToken, gitlab.WithBaseURL(config.GitlabURL))
	if err != nil {
		log.Fatalf("Failed to create GitLab client: %v", err)
	}

	// Verify GitLab connection
	_, _, err = git.Users.ListUsers(&gitlab.ListUsersOptions{})
	if err != nil {
		log.Fatalf("Failed to connect to GitLab: %v", err)
	}

	fmt.Printf("Successfully connected to GitLab at %s\n", config.GitlabURL)

	for {
		action := promptAction()
		switch action {
		case "Add/Update Project":
			addProject(git, &config)
		case "View Configuration":
			viewConfig(config)
		case "Update Global Settings":
			updateGlobalSettings(&config)
		case "Exit":
			// Save any changes to configuration before exiting
			saveConfig(config, *configFile)
			return
		}
	}
}

func loadConfig(configFile string) Config {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new config with empty values
			return Config{
				Projects: make([]ProjectConfig, 0),
			}
		}
		log.Fatalf("Error reading config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}
	return config
}

func saveConfig(config Config, configFile string) {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		log.Fatalf("Error encoding config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		log.Fatalf("Error saving config file: %v", err)
	}
}

func promptAction() string {
	prompt := promptui.Select{
		Label: "Select Action",
		Items: []string{
			"Add/Update Project",
			"View Configuration",
			"Update Global Settings",
			"Exit",
		},
	}

	_, result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed: %v", err)
	}

	return result
}

func updateGlobalSettings(config *Config) {
	fmt.Println("\nUpdating Global Settings")

	// GitLab Token
	prompt := promptui.Prompt{
		Label:   "GitLab Token",
		Default: config.GitlabToken,
	}
	if result, err := prompt.Run(); err == nil {
		config.GitlabToken = result
	}

	// GitLab URL
	prompt = promptui.Prompt{
		Label:   "GitLab URL (e.g., https://gitlab.example.com)",
		Default: config.GitlabURL,
	}
	if result, err := prompt.Run(); err == nil {
		config.GitlabURL = result
	}

	// Webhook Token
	prompt = promptui.Prompt{
		Label:   "Global Webhook Token",
		Default: config.WebhookToken,
	}
	if result, err := prompt.Run(); err == nil {
		config.WebhookToken = result
	}

	// PostgreSQL Connection URL
	prompt = promptui.Prompt{
		Label:   "PostgreSQL Connection URL",
		Default: config.PsqlConnURL,
	}
	if result, err := prompt.Run(); err == nil {
		config.PsqlConnURL = result
	}

	saveConfig(*config)
	fmt.Println("Global settings updated successfully!")
}

func addProject(git *gitlab.Client, config *Config) {
	// List projects
	projects, _, err := git.Projects.ListProjects(&gitlab.ListProjectsOptions{})
	if err != nil {
		log.Printf("Error listing projects: %v", err)
		return
	}

	projectNames := make([]string, len(projects))
	for i, project := range projects {
		projectNames[i] = fmt.Sprintf("%s (ID: %d)", project.Name, project.ID)
	}

	prompt := promptui.Select{
		Label: "Select Project",
		Items: projectNames,
	}

	index, _, err := prompt.Run()
	if err != nil {
		log.Printf("Project selection failed: %v", err)
		return
	}

	selectedProject := projects[index]

	// Check if project already exists in config
	var projectConfig ProjectConfig
	projectIndex := -1
	for i, p := range config.Projects {
		if p.ProjectID == selectedProject.ID {
			projectConfig = p
			projectIndex = i
			break
		}
	}

	// If project doesn't exist, initialize with defaults
	if projectIndex == -1 {
		projectConfig = ProjectConfig{
			ProjectID:    selectedProject.ID,
			WebhookToken: config.WebhookToken, // Use global webhook token as default
			MinApprovals: 2,                   // Default minimum approvals
		}
	}

	// List users
	users, _, err := git.Users.ListUsers(&gitlab.ListUsersOptions{})
	if err != nil {
		log.Printf("Error listing users: %v", err)
		return
	}

	// Select approvers
	var selectedUsers []string
	for {
		userNames := make([]string, len(users))
		for i, user := range users {
			userNames[i] = user.Username
		}
		userNames = append(userNames, "Done")

		prompt := promptui.Select{
			Label: "Select Approvers (choose 'Done' when finished)",
			Items: userNames,
		}

		index, result, err := prompt.Run()
		if err != nil {
			log.Printf("User selection failed: %v", err)
			return
		}

		if result == "Done" {
			break
		}

		selectedUsers = append(selectedUsers, users[index].Username)
	}

	// Update project configuration
	projectConfig.Approvals = selectedUsers

	// Prompt for minimum approvals
	minApprovPrompt := promptui.Prompt{
		Label:   "Minimum Required Approvals",
		Default: strconv.Itoa(projectConfig.MinApprovals),
		Validate: func(input string) error {
			num, err := strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("please enter a valid number")
			}
			if num < 1 {
				return fmt.Errorf("minimum approvals must be at least 1")
			}
			if num > len(selectedUsers) {
				return fmt.Errorf("minimum approvals cannot be greater than number of approvers")
			}
			return nil
		},
	}

	if result, err := minApprovPrompt.Run(); err == nil {
		projectConfig.MinApprovals, _ = strconv.Atoi(result)
	}

	// Prompt for project-specific webhook token
	webhookPrompt := promptui.Prompt{
		Label:   "Project Webhook Token (press enter to use global token)",
		Default: projectConfig.WebhookToken,
	}

	if result, err := webhookPrompt.Run(); err == nil {
		if result != "" {
			projectConfig.WebhookToken = result
		}
	}

	// Update or add project to configuration
	if projectIndex >= 0 {
		config.Projects[projectIndex] = projectConfig
	} else {
		config.Projects = append(config.Projects, projectConfig)
	}

	saveConfig(*config)
	fmt.Println("Project configuration updated successfully!")
}

func viewConfig(config Config) {
	// Mask sensitive information
	displayConfig := config
	if displayConfig.GitlabToken != "" {
		displayConfig.GitlabToken = "********"
	}
	if displayConfig.WebhookToken != "" {
		displayConfig.WebhookToken = "********"
	}
	for i := range displayConfig.Projects {
		if displayConfig.Projects[i].WebhookToken != "" {
			displayConfig.Projects[i].WebhookToken = "********"
		}
	}
	// Mask PostgreSQL password in connection URL
	if displayConfig.PsqlConnURL != "" {
		displayConfig.PsqlConnURL = "********"
	}

	data, err := json.MarshalIndent(displayConfig, "", "    ")
	if err != nil {
		log.Printf("Error encoding config for display: %v", err)
		return
	}
	fmt.Println(string(data))
}
