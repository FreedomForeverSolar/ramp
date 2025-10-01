package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"ramp/internal/config"
	"ramp/internal/scaffold"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new ramp project with interactive setup",
	Long: `Initialize a new ramp project by creating the necessary configuration
files and directory structure through an interactive setup process.

This is similar to 'npm init' - it will guide you through creating a
.ramp/ramp.yaml configuration file and optional setup scripts.

After initialization, use 'ramp install' to clone the configured repositories.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInit(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if ramp.yaml already exists
	if err := checkExistingProject(wd); err != nil {
		return err
	}

	// Show welcome message
	printWelcomeMessage()

	// Collect project information interactively
	projectData, err := collectProjectInfo(wd)
	if err != nil {
		return fmt.Errorf("failed to collect project information: %w", err)
	}

	// Create the project structure
	if err := scaffold.CreateProject(wd, projectData); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Show success message and next steps
	printSuccessMessage(wd, projectData)

	// Ask if user wants to clone repos now
	if err := promptInstallRepos(wd); err != nil {
		return err
	}

	return nil
}

func checkExistingProject(projectDir string) error {
	configPath := filepath.Join(projectDir, ".ramp", "ramp.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("ramp project already exists at %s\nUse 'ramp install' to clone repositories", configPath)
	}
	return nil
}

func printWelcomeMessage() {
	fmt.Println()
	fmt.Println("🚀 Welcome to Ramp!")
	fmt.Println("   Let's set up your multi-repository development environment.")
	fmt.Println()
}

func collectProjectInfo(wd string) (scaffold.ProjectData, error) {
	var data scaffold.ProjectData

	// Get default project name from current directory
	defaultName := filepath.Base(wd)

	// Collect basic project information
	if err := collectBasicInfo(&data, defaultName); err != nil {
		return data, err
	}

	// Collect repositories
	if err := collectRepositories(&data); err != nil {
		return data, err
	}

	// Collect optional features
	if err := collectOptionalFeatures(&data); err != nil {
		return data, err
	}

	// Collect custom commands
	if err := collectCustomCommands(&data); err != nil {
		return data, err
	}

	return data, nil
}

func collectBasicInfo(data *scaffold.ProjectData, defaultName string) error {
	var projectName string
	var branchPrefix string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Value(&projectName).
				Placeholder(defaultName),

			huh.NewInput().
				Title("Default branch prefix").
				Value(&branchPrefix).
				Placeholder("feature/"),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Use defaults if empty
	if projectName == "" {
		projectName = defaultName
	}
	if branchPrefix == "" {
		branchPrefix = "feature/"
	}

	data.Name = projectName
	data.BranchPrefix = branchPrefix

	return nil
}

func collectRepositories(data *scaffold.ProjectData) error {
	repoIndex := 1

	for {
		repo, err := collectSingleRepo(repoIndex)
		if err != nil {
			return err
		}
		data.Repos = append(data.Repos, repo)
		repoIndex++

		// Ask if they want to add another
		var addAnother bool
		addAnotherForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Add another repository?").
					Value(&addAnother).
					Affirmative("Yes").
					Negative("No"),
			),
		)

		if err := addAnotherForm.Run(); err != nil {
			return err
		}

		if !addAnother {
			break
		}
	}

	if len(data.Repos) == 0 {
		return fmt.Errorf("at least one repository is required")
	}

	return nil
}

func collectSingleRepo(index int) (scaffold.RepoData, error) {
	var repo scaffold.RepoData
	var gitURL string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Repository %d: Git URL", index)).
				Value(&gitURL).
				Placeholder("git@github.com:owner/repo.git").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("git URL is required")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return repo, err
	}

	repo.GitURL = gitURL
	repo.Path = "repos"

	return repo, nil
}

func collectOptionalFeatures(data *scaffold.ProjectData) error {
	// Default to true for setup and cleanup
	includeSetup := true
	includeCleanup := true
	enablePorts := false
	var basePortStr string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include setup script?").
				Description("Create scripts/setup.sh with sample setup logic").
				Value(&includeSetup).
				Affirmative("Yes").
				Negative("No"),

			huh.NewConfirm().
				Title("Include cleanup script?").
				Description("Create scripts/cleanup.sh with sample cleanup logic").
				Value(&includeCleanup).
				Affirmative("Yes").
				Negative("No"),

			huh.NewConfirm().
				Title("Enable port management?").
				Description("Allocate unique ports for each feature").
				Value(&enablePorts).
				Affirmative("Yes").
				Negative("No"),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	data.IncludeSetup = includeSetup
	data.IncludeCleanup = includeCleanup
	data.EnablePorts = enablePorts

	// If port management enabled, ask for base port
	if enablePorts {
		portForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Base port number").
					Value(&basePortStr).
					Placeholder("3000").
					Validate(func(s string) error {
						if s == "" {
							return nil
						}
						port, err := strconv.Atoi(s)
						if err != nil {
							return fmt.Errorf("must be a number")
						}
						if port < 1024 || port > 65535 {
							return fmt.Errorf("must be between 1024 and 65535")
						}
						return nil
					}),
			),
		)

		if err := portForm.Run(); err != nil {
			return err
		}

		basePort := 3000
		if basePortStr != "" {
			basePort, _ = strconv.Atoi(basePortStr)
		}
		data.BasePort = basePort
	}

	return nil
}

func collectCustomCommands(data *scaffold.ProjectData) error {
	addDoctor := true

	// Ask if they want to add a doctor command
	addCommandForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add a 'doctor' command for environment checks?").
				Description("Creates a sample command to verify your development environment").
				Value(&addDoctor).
				Affirmative("Yes").
				Negative("No"),
		),
	)

	if err := addCommandForm.Run(); err != nil {
		return err
	}

	if addDoctor {
		data.SampleCommands = append(data.SampleCommands, "doctor")
	}

	return nil
}

func printSuccessMessage(projectDir string, data scaffold.ProjectData) {
	fmt.Println()
	fmt.Println("✅ Project initialized successfully!")
	fmt.Println()
	fmt.Println("📁 Created structure:")
	fmt.Println("   .ramp/")
	fmt.Println("   ├── ramp.yaml")
	if data.IncludeSetup || data.IncludeCleanup || len(data.SampleCommands) > 0 {
		fmt.Println("   └── scripts/")
		if data.IncludeSetup {
			fmt.Println("       ├── setup.sh")
		}
		if data.IncludeCleanup {
			fmt.Println("       ├── cleanup.sh")
		}
		for i, cmd := range data.SampleCommands {
			if i == len(data.SampleCommands)-1 && !data.IncludeSetup && !data.IncludeCleanup {
				fmt.Printf("       └── %s.sh\n", cmd)
			} else {
				fmt.Printf("       ├── %s.sh\n", cmd)
			}
		}
	}
	fmt.Println("   repos/       (source repositories)")
	fmt.Println("   trees/       (ready for feature branches)")
	fmt.Println()
	fmt.Println("📋 Next steps:")
	step := 1
	fmt.Printf("   %d. Review and customize .ramp/ramp.yaml\n", step)
	step++
	if data.IncludeSetup {
		fmt.Printf("   %d. Customize scripts/setup.sh for your needs\n", step)
		step++
	}
	if len(data.SampleCommands) > 0 {
		fmt.Printf("   %d. Customize your custom command scripts in scripts/\n", step)
		step++
	}
	fmt.Printf("   %d. Run 'ramp install' to clone repositories\n", step)
	step++
	fmt.Printf("   %d. Run 'ramp up <feature-name>' to create a feature branch\n", step)
	if len(data.SampleCommands) > 0 {
		step++
		fmt.Printf("   %d. Run 'ramp run doctor -v' (example of running a custom command)\n", step)
	}
	fmt.Println()
}

func promptInstallRepos(projectDir string) error {
	shouldInstall := true

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Clone repositories now?").
				Description("Run 'ramp install' to clone all configured repositories").
				Value(&shouldInstall).
				Affirmative("Yes").
				Negative("No"),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if !shouldInstall {
		return nil
	}

	// Load the config we just created
	cfg, err := config.LoadConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Run install
	fmt.Println()
	return runInstallForProject(projectDir, cfg)
}
