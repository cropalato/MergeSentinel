package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gitlab.com/gitlab-org/api/client-go"
)

// Project e Config são as estruturas do seu JSON
type Project struct {
	ProjID       int      `json:"project_id"`
	Approvals    []string `json:"approvals"`
	WebhookToken string   `json:"webhook_token"`
	MinApprov    int      `json:"min_approv"`
}

type Config struct {
	GLabToken    string    `json:"gitlab_token"`
	GLabURL      string    `json:"gitlab_url"`
	WebhookToken string    `json:"webhook_token"`
	PgresConn    string    `json:psql_conn_url`
	Projects     []Project `json:"projects"`
}

const defaultConfigPath = "config.json"

// preview é um TextView global para facilitar a atualização
var preview *tview.TextView

// projectForm será o formulário usado para editar um project.
// Declaramos como variável global ou variável local no main, mas antes de as funções que a usam.
var projectForm *tview.Form
var projectsList *tview.List
var app *tview.Application
var footer *tview.TextView
var git *gitlab.Client

func main() {
	// Tenta carregar config
	config, err := loadConfig(defaultConfigPath)
	if err != nil {
		log.Printf("Não foi possível carregar '%s'. %s Usando config vazio.\n", defaultConfigPath, err)
		os.Exit(1)
		config = &Config{}
	}

	// Inicia a aplicação
	app = tview.NewApplication()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlQ:
			app.Stop()
		case tcell.KeyCtrlJ:
			clearProjectForm(projectForm)
			footer.SetText("[gray]Global ([white]Ctrl+G[gray]) |Projects ([white]Ctrl+P[gray]) | Quit ([white]Ctrl+Q[gray])")
			app.SetFocus(preview)
		}
		return event
	})
	footer = tview.NewTextView()
	footer.SetTextAlign(tview.AlignRight).SetDynamicColors(true).SetBorder(true)
	footer.SetText("[gray]Projects ([white]Ctrl+P[gray]) | Quit ([white]Ctrl+Q[gray])")

	// Formulário para campos globais
	glbField1 := tview.NewInputField().
		SetLabel("Gitlab URL:              ").
		SetText(config.GLabURL).
		SetChangedFunc(func(text string) {
			config.GLabURL = text
			refreshPreview(preview, config)
		})
	glbField2 := tview.NewInputField().
		SetLabel("Gitlab Token:  ").
		SetText(config.GLabToken).SetChangedFunc(func(text string) {
		config.GLabToken = text
		refreshPreview(preview, config)
	})
	glbField3 := tview.NewInputField().
		SetLabel("Postgres connection URL: ").
		SetText(config.PgresConn).SetChangedFunc(func(text string) {
		config.PgresConn = text
		refreshPreview(preview, config)
	})
	glbField4 := tview.NewInputField().
		SetLabel("Webhook Token: ").
		SetText(config.WebhookToken).SetChangedFunc(func(text string) {
		config.WebhookToken = text
		refreshPreview(preview, config)
	})
	fields := []*tview.InputField{glbField1, glbField2, glbField3, glbField4}

	// Set up Tab / Shift+Tab navigation using DoneFunc
	for i, f := range fields {
		idx := i
		f.SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyTab:
				// Move focus to the next field (wrap around)
				next := (idx + 1) % len(fields)
				app.SetFocus(fields[next])
			case tcell.KeyBacktab:
				// Move focus to the previous field (wrap around)
				prev := (idx - 1 + len(fields)) % len(fields)
				app.SetFocus(fields[prev])
			case tcell.KeyEnter:
				// Optional: If you want Enter to move forward
				next := (idx + 1) % len(fields)
				app.SetFocus(fields[next])
			case tcell.KeyCtrlP:
				app.SetFocus(projectsList)
			}
		})
		f.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyCtrlP:
				footer.SetText("[gray]Global ([white]Ctrl+G[gray]) |New project ([white]Ctrl+N[gray]) | Quit ([white]Ctrl+Q[gray])")
				app.SetFocus(projectsList)
				return event
			}
			return event
		})
	}
	// Create a Grid layout: 3 rows, 3 columns
	grid := tview.NewGrid().
		SetRows(1, 1).   // 3 rows
		SetColumns(0, 0) // 3 columns (expand to fit)
	grid.SetBorder(true).
		SetTitle(" Global ").
		SetTitleAlign(tview.AlignLeft)

	grid.AddItem(glbField1, 0, 0, 1, 1, 0, 0, true)
	grid.AddItem(glbField2, 0, 1, 1, 1, 0, 0, false)
	grid.AddItem(glbField3, 1, 0, 1, 1, 0, 0, false)
	grid.AddItem(glbField4, 1, 1, 1, 1, 0, 0, false)

	// Lista de projects
	projectsList = tview.NewList()
	projectsList.ShowSecondaryText(false).
		SetBorder(true).
		SetTitle("Projects").
		SetTitleAlign(tview.AlignLeft)

	// Inicializa o formulário de project
	projectForm = tview.NewForm()
	projectForm.SetBorder(true).
		SetTitle("Project").
		SetTitleAlign(tview.AlignLeft)

	// Área de pré-visualização do JSON
	preview = tview.NewTextView()
	preview.SetDynamicColors(true).
		SetBorder(true).
		SetTitle("JSON Preview").
		SetTitleAlign(tview.AlignLeft)
	preview.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlG:
			footer.SetText("[gray]Global ([white]Ctrl+G[gray]) |Projects ([white]Ctrl+P[gray]) | Quit ([white]Ctrl+Q[gray])")
			app.SetFocus(glbField1)
			return event
		case tcell.KeyCtrlP:
			app.SetFocus(projectsList)
			return event
		}
		// Default behavior for all other keys
		return event
	})

	refreshPreview(preview, config)

	git, err = gitlab.NewClient(config.GLabToken, gitlab.WithBaseURL(config.GLabURL))
	if err != nil {
		log.Fatalf("Failed configuring gitlab client. %v\n", err)
	}
	// Função para atualizar a lista com os itens do config
	projectsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlG:
			footer.SetText("[gray]Projects ([white]Ctrl+P[gray]) | Quit ([white]Ctrl+Q[gray])")
			app.SetFocus(glbField1)
			return event
		}
		// case tcell.KeyEnter:
		// 	// Block the default "select" if you want
		// 	// Or do something custom
		// 	// ...
		// 	// return nil to consume
		// 	return nil
		// }

		// Check for rune-based events
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case '+': // Move down
				novoProject := Project{
					ProjID:       0,
					Approvals:    []string{},
					WebhookToken: "",
					MinApprov:    0,
				}
				config.Projects = append(config.Projects, novoProject)
				loadProjectForm(projectForm, config, len(config.Projects)-1, preview)
				app.SetFocus(projectForm)
				refreshPreview(preview, config)
				return nil
			}
		}
		// Default behavior for all other keys
		return event
	})

	// Container vertical (Flex) para a lista + botões
	projectsBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(projectsList, 0, 1, true)

		/*
			// Botão Adicionar
			addProjectButton := tview.NewButton("[green::b]+ Adicionar Project").SetSelectedFunc(func() {
				novoProject := Project{
					ProjID:       0,
					Approvals:    []string{},
					WebhookToken: "",
					MinApprov:    0,
				}
				config.Projects = append(config.Projects, novoProject)
				populateProjectsList()
				refreshPreview(preview, config)
			})

			// Botão Remover
			removeProjectButton := tview.NewButton("[red::b]- Remover Project Selecionado").SetSelectedFunc(func() {
				index := projectsList.GetCurrentItem()
				if index >= 0 && index < len(config.Projects) {
					config.Projects = append(config.Projects[:index], config.Projects[index+1:]...)
					populateProjectsList()
					clearProjectForm(projectForm)
					refreshPreview(preview, config)
				}
			})

			projectsBox.AddItem(addProjectButton, 1, 0, false)
			projectsBox.AddItem(removeProjectButton, 1, 0, false)
		*/
	// Meio: lista à esquerda, form de project à direita
	middleFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(projectsBox, 0, 1, true).
		AddItem(projectForm, 0, 2, false)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(grid, 4, 1, true).
		AddItem(middleFlex, 15, 1, true)

	// Raiz: mainFlex em cima, preview embaixo
	rootFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainFlex, 19, 1, true).
		AddItem(preview, 0, 1, false).
		AddItem(footer, 3, 1, false)

	// Preenche a lista inicialmente
	populateProjectsList(config, projectsList)

	app.SetRoot(rootFlex, true).SetFocus(glbField1)

	// Executa o loop
	if err := app.Run(); err != nil {
		log.Fatalf("Erro ao executar tview: %v\n", err)
	}

	// Ao sair, salva a config
	if err := saveConfig(defaultConfigPath, config); err != nil {
		log.Printf("Erro ao salvar config: %v\n", err)
	}
}

// ----------------------------------------------------------------------------
// Funções auxiliares
// ----------------------------------------------------------------------------

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// sortProjectsByID sorts the Projects slice by ProjID in ascending order
	sortProjectsByID(&cfg)
	return &cfg, nil
}

func saveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// sortProjectsByID sorts the Projects slice by ProjID in ascending order
func sortProjectsByID(cfg *Config) {
	sort.Slice(cfg.Projects, func(i, j int) bool {
		return cfg.Projects[i].ProjID < cfg.Projects[j].ProjID
	})
}

// deleteProjectByID removes a project from cfg.Projects based on projID.
// It returns true if a project was found and deleted, false otherwise.
func deleteProjectByID(cfg *Config, projID int) bool {
	for i, p := range cfg.Projects {
		if p.ProjID == projID {
			// Remove the project at index i
			cfg.Projects = append(cfg.Projects[:i], cfg.Projects[i+1:]...)
			return true
		}
	}
	return false
}

// removeDuplicateProjects removes duplicates from cfg.Projects based on ProjID.
// Only the first occurrence of a given ProjID is kept.
func removeDuplicateProjects(cfg *Config) {
	seen := make(map[int]bool)
	var unique []Project

	for _, p := range cfg.Projects {
		if !seen[p.ProjID] {
			seen[p.ProjID] = true
			unique = append(unique, p)
		}
	}

	cfg.Projects = unique
}

func populateProjectsList(config *Config, projectsList *tview.List) {
	projectsList.Clear()
	for i, p := range config.Projects {
		idx := i
		projectPath := "[red]invalid project"
		pInfo, _, err := git.Projects.GetProject(p.ProjID, nil)
		if err == nil {
			projectPath = pInfo.PathWithNamespace
		}
		projectsList.AddItem(projectPath, "", 0, func() {
			loadProjectForm(projectForm, config, idx, preview)
			app.SetFocus(projectForm)
			refreshPreview(preview, config)
		})
	}
}

func refreshPreview(tv *tview.TextView, cfg *Config) {
	if tv == nil {
		return
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		tv.SetText("[red]Erro ao gerar JSON[/red]")
		return
	}
	tv.SetText(string(data))
}

func loadProjectForm(form *tview.Form, cfg *Config, idx int, preview *tview.TextView) {
	form.Clear(true)
	if idx < 0 || idx >= len(cfg.Projects) {
		return
	}
	footer.SetText("Quit ([white]Ctrl+Q[gray])")
	p := &cfg.Projects[idx]
	isNew := true

	if p.ProjID == 0 {
		form.AddInputField("Project id", strconv.Itoa(p.ProjID), 5, tview.InputFieldInteger, func(text string) {
			n, _ := strconv.Atoi(text)
			p.ProjID = n
			//refreshPreview(preview, cfg)
		})
	} else {
		isNew = false
		form.AddTextView("project id", strconv.Itoa(p.ProjID), 0, 1, true, false)
	}
	form.AddInputField("Webhook Token", p.WebhookToken, 30, nil, func(text string) {
		p.WebhookToken = text
		//refreshPreview(preview, cfg)
	})
	form.AddInputField("Min num of approvals", strconv.Itoa(p.MinApprov), 5, tview.InputFieldInteger, func(text string) {
		n, _ := strconv.Atoi(text)
		p.MinApprov = n
		//refreshPreview(preview, cfg)
	})
	tmp_app := strings.Join(p.Approvals, ",")
	form.AddInputField("Approvals", tmp_app, 30, nil, func(text string) {
		p.Approvals = strings.Split(text, ",")
		//refreshPreview(preview, cfg)
	})

	form.AddButton("Save", func() {
		populateProjectsList(cfg, projectsList)
		refreshPreview(preview, cfg)
		clearProjectForm(form)
		footer.SetText("[gray]Global ([white]Ctrl+G[gray]) |New project ([white]Ctrl+N[gray]) | Quit ([white]Ctrl+Q[gray])")
		app.SetFocus(projectsList)
	})
	if !isNew {
		form.AddButton("Delete", func() {
			deleteProjectByID(cfg, p.ProjID)
			populateProjectsList(cfg, projectsList)
			refreshPreview(preview, cfg)
			clearProjectForm(form)
			footer.SetText("[gray]Global ([white]Ctrl+G[gray]) |New project ([white]Ctrl+N[gray]) | Quit ([white]Ctrl+Q[gray])")
			app.SetFocus(projectsList)
		})
	}
	form.AddButton("Cancel", func() {
		clearProjectForm(form)
		footer.SetText("[gray]Global ([white]Ctrl+G[gray]) |New project ([white]Ctrl+N[gray]) | Quit ([white]Ctrl+Q[gray])")
		app.SetFocus(projectsList)
	})
}

func clearProjectForm(form *tview.Form) {
	form.Clear(true)
}
