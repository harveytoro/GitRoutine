package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"bytes"

	"github.com/jroimartin/gocui"
)

const (
	repositoryView         = "repositoryView"
	summaryView            = "summaryView"
	terminalView           = "terminalView"
	selectedRepositoryView = "selectedRepositoryView"
	currentBranchView      = "currentBranchView"
	logView                = "logView"

	repositoryViewTitle = "Repositories"
	summaryViewTitle    = "Summary"
	logViewTitle        = "Log"
	terminalViewTitle   = "Terminal"

	selectedRepositoryViewPrefix = "Repository: "
	currentBranchViewPrefix      = "Current branch: "

	executableCommand    = "git"
	clearSummaryCommand  = "clear"
	commandInputPrefix   = "git > "
	currentBranchCommand = "git > rev-parse --abbrev-ref HEAD"
	logCommand           = "git > -c color.ui=always log --all --decorate --oneline --graph"

	configurationFileName = "config.json"
)

type uiManager struct {
	configuration
	hasLoaded bool
}

type repository struct {
	Name string
	Path string
}

type configuration struct {
	Repositories []repository
}

func main() {

	config, _ := loadConfiguration()

	uiMgr := &uiManager{}
	uiMgr.configuration = config

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Cursor = true
	g.Mouse = true

	g.SetManagerFunc(uiMgr.layoutManager)

	g.SetKeybinding(terminalView, gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, _ *gocui.View) error {
		v, err := g.View(terminalView)
		if err != nil {
			return err
		}

		inputTerminal := v.Buffer()
		v.Clear()
		fmt.Fprintln(v, commandInputPrefix)
		v.SetCursor(6, 0)
		p, err := g.View(summaryView)
		if err != nil {
			return err
		}
		p.SetCursor(0, 0)

		err = executeCommand(p, strings.TrimSpace(inputTerminal))

		if err != nil {
			return err
		}

		return nil
	})

	g.SetKeybinding(repositoryView, gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		_, cy := v.Cursor()
		line, _ := v.Line(cy)
		return uiMgr.setRepository(g, line)
	})

	g.SetKeybinding("", gocui.MouseLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		p, err := g.View(terminalView)
		if err != nil {
			return err
		}
		p.Clear()
		p.SetCursor(6, 0)
		fmt.Fprintln(p, commandInputPrefix)

		return nil
	})

	g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		v, _ = g.View(logView)
		if err != nil {
			return err
		}
		ox, oy := v.Origin()
		v.SetOrigin(ox, oy+1)

		return nil
	})

	g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		v, _ = g.View(logView)
		if err != nil {
			return err
		}
		ox, oy := v.Origin()
		v.SetOrigin(ox, oy-1)

		return nil
	})

	g.SetKeybinding(terminalView, gocui.KeyBackspace2, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		x, _ := v.Cursor()
		if x > 6 {
			v.EditDelete(true)
		}

		return nil
	})

	g.SetKeybinding(terminalView, gocui.KeyBackspace, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {

		x, _ := v.Cursor()
		if x > 6 {
			v.EditDelete(true)
		}

		return nil
	})

	g.SetKeybinding("", gocui.MouseWheelDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return nil
	})

	g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return nil
	})

	g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return nil
	})

	g.SetKeybinding("", gocui.MouseWheelUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return nil
	})

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func (mgr *uiManager) setRepository(g *gocui.Gui, repoName string) error {

	if repoName == "" && !mgr.hasLoaded {
		repoName = mgr.Repositories[0].Name
		mgr.hasLoaded = true
	} else if repoName == "" && mgr.hasLoaded {
		return nil
	}

	p, err := g.View(summaryView)
	repoNameView, err := g.View(selectedRepositoryView)
	branchView, err := g.View(currentBranchView)
	logView, err := g.View(logView)
	if err != nil {
		return err
	}

	p.Clear()
	p.SetCursor(0, 0)

	for _, repos := range mgr.Repositories {
		if repos.Name == repoName {
			os.Chdir(repos.Path)
			repoNameView.Clear()
			fmt.Fprintln(repoNameView, selectedRepositoryViewPrefix+repos.Name)
			branchView.Clear()
			logView.Clear()
			logView.SetOrigin(0, 0)
			fmt.Fprint(branchView, currentBranchViewPrefix)
			executeCommand(branchView, currentBranchCommand)
			executeCommand(logView, logCommand)
		}
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (mgr *uiManager) layoutManager(g *gocui.Gui) error {

	maxX, maxY := g.Size()
	if v, err := g.SetView(repositoryView, 1, 0, (maxX/2)/2, maxY-2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = repositoryViewTitle
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen

		for _, repo := range mgr.Repositories {
			fmt.Fprintln(v, repo.Name)
		}
	}

	if v, err := g.SetView(summaryView, ((maxX/2)/2)+1, 0, maxX-1, maxY/2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = summaryViewTitle
		v.Autoscroll = true
	}

	if v, err := g.SetView(selectedRepositoryView, ((maxX/2)/2)+1, (maxY/2)+1, (maxX+((maxX/2)/2)+1)/2, (maxY/2)+3); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(v, selectedRepositoryViewPrefix)
	}

	if v, err := g.SetView(currentBranchView, (maxX+((maxX/2)/2)+1)/2+1, (maxY/2)+1, maxX-1, (maxY/2)+3); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(v, currentBranchViewPrefix)
	}

	if v, err := g.SetView(logView, ((maxX/2)/2)+1, (maxY/2)+4, maxX-1, (maxY - 5)); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = logViewTitle
		v.Wrap = true
	}

	if v, err := g.SetView(terminalView, ((maxX/2)/2)+1, maxY-4, maxX-1, maxY-2); err != nil {

		if err != gocui.ErrUnknownView {
			return err
		}

		v.Editable = true
		v.Title = terminalViewTitle
		fmt.Fprintln(v, commandInputPrefix)
		v.SetCursor(6, 0)
	}

	g.SetCurrentView(terminalView)
	mgr.setRepository(g, "")
	return nil
}

func executeCommand(view *gocui.View, command string) error {

	var (
		output    bytes.Buffer
		errOutput bytes.Buffer
		err       error
	)

	s := splitSpaceQuotesAware(command)

	if s[2] == clearSummaryCommand {
		view.Clear()
		return nil
	}

	cmd := executableCommand
	execCmd := exec.Command(cmd, s[2:]...)
	execCmd.Stderr = &errOutput
	execCmd.Stdout = &output

	err = execCmd.Run()

	if err != nil {
		fmt.Fprintln(view, "Failed to execute command. Please try again.")
		return nil
	}

	fmt.Fprintln(view, output.String())

	return nil
}

func loadConfiguration() (configuration, error) {

	file, _ := os.Open(configurationFileName)
	decoder := json.NewDecoder(file)
	configuration := configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		return configuration, err
	}

	return configuration, nil
}

func splitSpaceQuotesAware(input string) []string {

	var ret []string
	var currentString string
	inQuotes := false
	for i := 0; i < len(input); i++ {

		element := input[i : i+1]

		if element == "\"" {
			inQuotes = !inQuotes
		}

		if (element != " " || inQuotes) && element != "\"" {
			currentString = currentString + element
		}

		if element == " " && !inQuotes || i == len(input)-1 {
			ret = append(ret, currentString)
			currentString = ""
		}

		if i == len(input)-1 && element == " " {
			ret = append(ret, currentString)
		}
	}
	return ret
}
